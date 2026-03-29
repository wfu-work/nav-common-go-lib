package utils

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/wfu-work/nav-common-go-lib/global"
	"go.uber.org/zap"
)

// PathExists
// @description: 文件目录是否存在
// @param: path string
// @return: bool, error
func PathExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil {
		if fi.IsDir() {
			return true, nil
		}
		return false, errors.New("存在同名文件")
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// CreateDir
// @description: 批量创建文件夹
// @param: dirs ...string
// @return: err error
func CreateDir(dirs ...string) (err error) {
	for _, v := range dirs {
		exist, err := PathExists(v)
		if err != nil {
			return err
		}
		if !exist {
			global.NAV_LOG.Debug("create directory" + v)
			if err := os.MkdirAll(v, os.ModePerm); err != nil {
				global.NAV_LOG.Error("create directory"+v, zap.Any(" error:", err))
				return err
			}
		}
	}
	return err
}

// FileExist 判断文件是否存在
func FileExist(path string) bool {
	fi, err := os.Lstat(path)
	if err == nil {
		return !fi.IsDir()
	}
	return !os.IsNotExist(err)
}

// GetParentDirectory 获取父级目录
func GetParentDirectory(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(absPath)
	return parent, nil
}

// GetParentDir 获取父级目录
func GetParentDir(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	parent := filepath.Dir(absPath)
	return parent
}

// DelFile 删除文件
func DelFile(filePath string) error {
	return os.RemoveAll(filePath)
}

// GetPathSeparator returns the operating system-specific path separator.
func GetPathSeparator() string {
	return string(os.PathSeparator)
}

// ReadLastLine 从给定文件路径中读取并返回文件的最后一行内容。
func ReadLastLine(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", err
	}
	var (
		size     = stat.Size()
		buf      = make([]byte, 1)
		lastLine []byte
	)

	// 从文件末尾往前读，直到遇到 '\n'
	for i := size - 1; i >= 0; i-- {
		_, err := f.Seek(i, os.SEEK_SET)
		if err != nil {
			return "", err
		}
		_, err = f.Read(buf)
		if err != nil {
			return "", err
		}
		if buf[0] == '\n' && i != size-1 {
			break
		}
		lastLine = append([]byte{buf[0]}, lastLine...)
	}
	return string(bytes.TrimSpace(lastLine)), nil
}

// GetFileName 获取文件名
func GetFileName(path string) string {
	return filepath.Base(path)
}

// GetFileNameWithoutExt 获取文件名不带后缀
func GetFileNameWithoutExt(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// GetFileSize 返回给定路径文件的大小（字节），如果失败则返回 -1 和错误
func GetFileSize(path string) int64 {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return -1
	}
	return fileInfo.Size()
}

// ReadFileLines 读取文件为行切片
func ReadFileLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// AppendFile 追加文件内容
func AppendFile(dst *os.File, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(dst, in)
	return err
}

// FindFileWithExt 在指定目录下查找具有指定扩展名的文件
func FindFileWithExt(root string, ext string) (string, error) {
	var result string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ext) {
			result = path
			return errors.New("FOUND")
		}
		return nil
	})
	if err != nil && err.Error() != "FOUND" {
		return "", err
	}
	if result == "" {
		return "", errors.New("no " + ext + " file found")
	}
	return result, nil
}
