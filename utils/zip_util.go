package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Unzip 将 zip 文件解压到目标目录
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		// 如果是目录，则创建
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(path, os.ModePerm)
			continue
		}
		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		// 创建目标文件
		ofile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return err
		}
		defer ofile.Close()

		// 写入文件内容
		if _, err := io.Copy(ofile, rc); err != nil {
			return err
		}
	}
	return nil
}

func UnzipFlatten(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		name := f.Name
		if name == "__MACOSX/" ||
			strings.HasPrefix(name, "__MACOSX/") ||
			strings.HasSuffix(name, ".DS_Store") {
			continue
		}
		parts := strings.Split(name, "/")
		if len(parts) <= 1 {
			continue
		}
		relPath := filepath.Join(parts[1:]...)
		targetPath := filepath.Join(dest, relPath)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(dest)) {
			return fmt.Errorf("illegal file path: %s", targetPath)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		ofile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			_ = rc.Close()
			return err
		}
		_, err = io.Copy(ofile, rc)
		_ = rc.Close()
		_ = ofile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func UnzipWithCustomName(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// 获取不带扩展名的文件名作为目标目录名
	baseName := filepath.Base(src)
	dirName := baseName[:len(baseName)-len(filepath.Ext(baseName))]
	destDir := filepath.Join(dest, dirName)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		path := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(path, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		ofile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer ofile.Close()

		if _, err := io.Copy(ofile, rc); err != nil {
			return err
		}
	}
	return nil
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
