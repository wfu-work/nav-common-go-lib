package middlewares

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wfu-work/nav-common-go-lib/configs"
	"github.com/wfu-work/nav-common-go-lib/global"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Zap 获取 zap.Logger
func Zap() *zap.Logger {
	logger, _, err := ZapWithConfig(global.Config().Zap)
	if err != nil {
		return zap.NewNop()
	}
	return logger
}

// ZapWithConfig gives the resource owner an explicit writer cleanup function.
func ZapWithConfig(config configs.Zap) (*zap.Logger, func() error, error) {
	if err := os.MkdirAll(config.Director, 0755); err != nil {
		return nil, nil, err
	}
	var cores []zapcore.Core
	var cutters []*Cutter
	for _, level := range config.Levels() {
		cutter := NewCutter(config.Director, level.String(), config.RetentionDay, CutterWithLayout(time.DateOnly))
		cutters = append(cutters, cutter)
		var writer zapcore.WriteSyncer = cutter
		if config.LogInConsole {
			writer = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), cutter)
		}
		cores = append(cores, zapcore.NewCore(config.Encoder(), writer, zap.LevelEnablerFunc(func(l zapcore.Level) bool { return l == level })))
	}
	logger := zap.New(zapcore.NewTee(cores...))
	if config.ShowLine {
		logger = logger.WithOptions(zap.AddCaller())
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = removeNDaysFolders(config.Director, config.RetentionDay)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				_ = removeNDaysFolders(config.Director, config.RetentionDay)
			}
		}
	}()
	var once sync.Once
	var closeErr error
	cleanup := func() error {
		once.Do(func() {
			close(stop)
			<-done
			for _, cutter := range cutters {
				if err := cutter.Close(); err != nil {
					closeErr = err
				}
			}
		})
		return closeErr
	}
	return logger, cleanup, nil
}

type ZapCore struct {
	level zapcore.Level
	zapcore.Core
}

func NewZapCore(level zapcore.Level) *ZapCore {
	entity := &ZapCore{level: level}
	syncer := entity.WriteSyncer()
	levelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l == level
	})
	entity.Core = zapcore.NewCore(global.Config().Zap.Encoder(), syncer, levelEnabler)
	return entity
}

func (z *ZapCore) WriteSyncer(formats ...string) WriteSyncer {
	cutter := NewCutter(
		global.Config().Zap.Director,
		z.level.String(),
		global.Config().Zap.RetentionDay,
		CutterWithLayout(time.DateOnly),
		CutterWithFormats(formats...),
	)
	if global.Config().Zap.LogInConsole {
		multiSyncer := zapcore.NewMultiWriteSyncer(os.Stdout, cutter)
		return zapcore.AddSync(multiSyncer)
	}
	return zapcore.AddSync(cutter)
}

type WriteSyncer interface {
	io.Writer
	Sync() error
}

type writerWrapper struct {
	io.Writer
}

func (w writerWrapper) Sync() error {
	return nil
}

// Cutter 实现 io.Writer 接口
// 用于日志切割, strings.Join([]string{director,layout, formats..., level+".log"}, os.PathSeparator)
type Cutter struct {
	level        string   // 日志级别(debug, info, warn, error, dpanic, panic, fatal)
	layout       string   // 时间格式 2006-01-02 15:04:05
	formats      []string // 自定义参数([]string{Director,"2006-01-02", "business"(此参数可不写), level+".log"}
	director     string   // 日志文件夹
	retentionDay int      //日志保留天数
	filename     string
	closed       bool
	file         *os.File      // 文件句柄
	mutex        *sync.RWMutex // 读写锁
}

type CutterOption func(*Cutter)

func NewCutter(director string, level string, retentionDay int, options ...CutterOption) *Cutter {
	rotate := &Cutter{
		level:        level,
		director:     director,
		retentionDay: retentionDay,
		mutex:        new(sync.RWMutex),
	}
	for i := 0; i < len(options); i++ {
		options[i](rotate)
	}
	return rotate
}

// CutterWithLayout 时间格式
func CutterWithLayout(layout string) CutterOption {
	return func(c *Cutter) {
		c.layout = layout
	}
}

// CutterWithFormats 格式化参数
func CutterWithFormats(format ...string) CutterOption {
	return func(c *Cutter) {
		if len(format) > 0 {
			c.formats = format
		}
	}
}

// Write satisfies the io.Writer interface. It writes to the
// appropriate file handle that is currently being used.
// If we have reached rotation time, the target file gets
// automatically rotated, and also purged if necessary.
func (c *Cutter) Write(bytes []byte) (n int, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.closed {
		return 0, os.ErrClosed
	}
	now := time.Now()
	length := len(c.formats)
	values := make([]string, 0, 3+length)
	values = append(values, c.director)
	if c.layout != "" {
		values = append(values, now.Format(c.layout))
	}
	for i := 0; i < length; i++ {
		values = append(values, c.formats[i])
	}
	if c.layout != "" {
		values = append(values, c.level+"."+now.Format(c.layout)+".log")
	} else {
		values = append(values, c.level+".log")
	}
	filename := filepath.Join(values...)
	if c.file == nil || c.filename != filename {
		if c.file != nil {
			if err := c.file.Sync(); err != nil {
				return 0, err
			}
			if err := c.file.Close(); err != nil {
				return 0, err
			}
			c.file = nil
		}
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			return 0, err
		}
		c.file, err = os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return 0, err
		}
		c.filename = filename
	}

	return c.file.Write(bytes)
}

func (c *Cutter) Sync() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.file != nil {
		return c.file.Sync()
	}
	return nil
}

func (c *Cutter) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.file == nil {
		return nil
	}
	syncErr := c.file.Sync()
	closeErr := c.file.Close()
	c.file = nil
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}

// Cleanup only date directories owned by the logger; never walk arbitrary trees.
func removeNDaysFolders(dir string, days int) error {
	if days <= 0 {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	cutoff := time.Now().AddDate(0, 0, -days).Format(time.DateOnly)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := time.Parse(time.DateOnly, entry.Name()); err != nil {
			continue
		}
		if entry.Name() < cutoff {
			if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
