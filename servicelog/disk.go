package servicelog

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (r *Recorder) files() ([]string, error) {
	if r.opts.Directory == "" {
		return nil, nil
	}
	return filepath.Glob(filepath.Join(r.opts.Directory, "events-*.jsonl"))
}
func (r *Recorder) prepareDisk() error {
	if r.opts.Directory == "" {
		return nil
	}
	if err := os.MkdirAll(r.opts.Directory, 0700); err != nil {
		return err
	}
	return r.prune("")
}
func (r *Recorder) prune(active string) error {
	files, err := r.files()
	if err != nil {
		return err
	}
	sort.Strings(files)
	var total int64
	for _, p := range files {
		st, e := os.Stat(p)
		if e == nil {
			total += st.Size()
		}
	}
	for _, p := range files {
		if p == active {
			continue
		}
		st, e := os.Stat(p)
		if e != nil {
			continue
		}
		if time.Since(st.ModTime()) > r.opts.Retention || total > r.opts.MaxDiskBytes-r.opts.MaxFileBytes {
			if e = os.Remove(p); e != nil {
				return e
			}
			total -= st.Size()
		}
	}
	return nil
}
func (r *Recorder) readDisk(visit func(entry)) error {
	files, err := r.files()
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, p := range files {
		f, e := os.Open(p)
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return e
		}
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 4096), r.opts.MaxEventBytes+1024)
		for s.Scan() {
			var event Event
			if json.Unmarshal(s.Bytes(), &event) == nil && event.ID != "" {
				visit(entry{event, append([]byte(nil), s.Bytes()...)})
			}
		}
		e = s.Err()
		_ = f.Close()
		if e != nil {
			return fmt.Errorf("read log file: %w", e)
		}
	}
	return nil
}
func (r *Recorder) persist() {
	defer close(r.done)
	var file *os.File
	var size int64
	var day, path string
	part := 0
	closeFile := func() {
		if file != nil {
			r.diskError(file.Sync())
			r.diskError(file.Close())
			file = nil
		}
	}
	defer closeFile()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	pruneTicker := time.NewTicker(time.Minute)
	defer pruneTicker.Stop()
	for {
		select {
		case x, ok := <-r.queue:
			if !ok {
				return
			}
			if r.opts.Console != nil {
				_, _ = r.opts.Console.Write(append(append([]byte(nil), x.data...), '\n'))
			}
			if r.opts.Directory == "" {
				continue
			}
			today := time.Now().Format(time.DateOnly)
			if file == nil || day != today || size+int64(len(x.data)+1) > r.opts.MaxFileBytes {
				closeFile()
				day = today
				part++
				path = filepath.Join(r.opts.Directory, fmt.Sprintf("events-%s-%s-%06d.jsonl", day, strings.ReplaceAll(r.boot, ":", ""), part))
				var err error
				file, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				r.diskError(err)
				size = 0
				if file == nil {
					continue
				}
				r.diskError(r.prune(path))
			}
			n, err := file.Write(append(append([]byte(nil), x.data...), '\n'))
			size += int64(n)
			r.diskError(err)
		case <-ticker.C:
			if file != nil {
				r.diskError(file.Sync())
			}
		case <-pruneTicker.C:
			r.diskError(r.prune(path))
		}
	}
}
