package utils

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// UnzipOptions bounds the work and disk space consumed by an archive.
type UnzipOptions struct {
	MaxFiles int
	MaxBytes int64
	Flatten  bool
}

func Unzip(src, dest string) error        { return ExtractZip(src, dest, UnzipOptions{}) }
func UnzipFlatten(src, dest string) error { return ExtractZip(src, dest, UnzipOptions{Flatten: true}) }
func UnzipWithCustomName(src, dest string) error {
	base := filepath.Base(src)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if !filepath.IsLocal(name) {
		return errors.New("invalid archive name")
	}
	return Unzip(src, filepath.Join(dest, name))
}

// ExtractZip rejects traversal, links, special files and oversized archives.
// os.Root also prevents traversal through symlinks already present on disk.
func ExtractZip(src, dest string, options UnzipOptions) error {
	if options.MaxFiles <= 0 {
		options.MaxFiles = 10000
	}
	if options.MaxBytes <= 0 {
		options.MaxBytes = 1 << 30
	}
	archive, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer archive.Close()
	if len(archive.File) > options.MaxFiles {
		return errors.New("too many zip entries")
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	root, err := os.OpenRoot(dest)
	if err != nil {
		return err
	}
	defer root.Close()
	remaining := options.MaxBytes
	for _, file := range archive.File {
		name := strings.TrimSuffix(file.Name, "/")
		if strings.Contains(name, "\\") || !filepath.IsLocal(filepath.FromSlash(name)) {
			return fmt.Errorf("illegal zip path: %q", file.Name)
		}
		if file.Mode()&os.ModeSymlink != 0 || (!file.FileInfo().IsDir() && !file.Mode().IsRegular()) {
			return fmt.Errorf("unsupported zip entry: %q", file.Name)
		}
		if options.Flatten {
			if strings.HasPrefix(name, "__MACOSX/") || name == "__MACOSX" || strings.HasSuffix(name, ".DS_Store") {
				continue
			}
			_, rest, found := strings.Cut(name, "/")
			if !found || rest == "" {
				continue
			}
			name = rest
			if !filepath.IsLocal(filepath.FromSlash(name)) {
				return fmt.Errorf("illegal zip path: %q", file.Name)
			}
		}
		name = filepath.FromSlash(name)
		if file.FileInfo().IsDir() {
			if err := root.MkdirAll(name, 0755); err != nil {
				return err
			}
			continue
		}
		if file.UncompressedSize64 > uint64(remaining) {
			return errors.New("zip exceeds uncompressed size limit")
		}
		n, err := extractZipFile(root, name, file, remaining)
		if err != nil {
			return err
		}
		remaining -= n
	}
	return nil
}

func extractZipFile(root *os.Root, name string, entry *zip.File, remaining int64) (int64, error) {
	if err := root.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return 0, err
	}
	reader, err := entry.Open()
	if err != nil {
		return 0, err
	}
	defer reader.Close()
	temp := name + ".part-" + GenerateGuid()
	file, err := root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return 0, err
	}
	defer root.Remove(temp)
	copyLimit := remaining
	if copyLimit < math.MaxInt64 {
		copyLimit++
	}
	n, copyErr := io.Copy(file, io.LimitReader(reader, copyLimit))
	closeErr := file.Close()
	if copyErr != nil {
		return n, copyErr
	}
	if closeErr != nil {
		return n, closeErr
	}
	if n > remaining {
		return n, errors.New("zip exceeds uncompressed size limit")
	}
	return n, root.Rename(temp, name)
}

// ToZip 压缩文件夹到 Zip 文件
// srcPath: 要压缩的文件或文件夹路径
// zipFilePath: 生成的 zip 文件路径
// keepDirStructure: 是否保留文件夹结构
func ToZip(srcPath string, zipFilePath string, keepDirStructure bool) error {
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if !srcInfo.IsDir() {
		return AddFileToZip(zipWriter, srcPath, srcInfo.Name())
	}
	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == srcPath {
			return nil
		}
		var zipPath string
		if keepDirStructure {
			zipPath, _ = filepath.Rel(filepath.Dir(srcPath), path)
		} else {
			zipPath = info.Name()
		}
		if info.IsDir() {
			_, err := zipWriter.Create(zipPath + "/")
			return err
		}
		return AddFileToZip(zipWriter, path, zipPath)
	})
}

// AddFileToZip 添加单个文件到 zip
func AddFileToZip(zipWriter *zip.Writer, filePath, zipPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	w, err := zipWriter.Create(zipPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, file)
	return err
}
