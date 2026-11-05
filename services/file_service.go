package services

import (
	"encoding/base64"
	"errors"
	"io"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

type FileService struct{ company string }

func (s FileService) WithCompany(company string) FileService { s.company = company; return s }
func (s FileService) storageRoot() string {
	return filepath.Join(global.Config().Local.OssPath, s.company)
}
func (s FileService) storageURL(name string) string {
	return strings.TrimRight(global.Config().System.RouterPrefix+global.Config().Local.ProxyOssPath, "/") + "/" + filepath.ToSlash(filepath.Join(s.company, name))
}

var FileServiceApp = new(FileService)

var mu sync.Mutex

func (s FileService) UploadFile(file *multipart.FileHeader) (string, error) {
	limit := global.Config().System.MaxUploadBytes
	if limit <= 0 {
		limit = 32 << 20
	}
	if file == nil || file.Size > limit {
		return "", errors.New("上传文件超过大小限制")
	}
	ext := filepath.Ext(file.Filename)
	filename := utils.GenerateGuid() + ext         // 拼接新文件名
	mkdirErr := os.MkdirAll(s.storageRoot(), 0755) // 尝试创建此路径
	if mkdirErr != nil {
		global.NAV_LOG.Error("function os.MkdirAll() failed", zap.Any("err", mkdirErr.Error()))
		return "", errors.New("function os.MkdirAll() failed, err:" + mkdirErr.Error())
	}
	p := filepath.Join(s.storageRoot(), filename)
	filePath := s.storageURL(filename)

	f, openError := file.Open()
	if openError != nil {
		global.NAV_LOG.Error("function file.Open() failed", zap.Any("err", openError.Error()))
		return "", errors.New("function file.Open() failed, err:" + openError.Error())
	}
	defer f.Close()

	out, createErr := os.Create(p)
	if createErr != nil {
		global.NAV_LOG.Error("function os.Create() failed", zap.Any("err", createErr.Error()))
		return "", errors.New("function os.Create() failed, err:" + createErr.Error())
	}
	defer out.Close()

	copyLimit := limit
	if copyLimit < math.MaxInt64 {
		copyLimit++
	}
	copied, copyErr := io.Copy(out, io.LimitReader(f, copyLimit))
	if copyErr == nil && copied > limit {
		copyErr = errors.New("上传文件超过大小限制")
	}
	if copyErr != nil {
		_ = out.Close()
		_ = os.Remove(p)
		global.NAV_LOG.Error("function io.Copy() failed", zap.Any("err", copyErr.Error()))
		return "", errors.New("function io.Copy() failed, err:" + copyErr.Error())
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(p)
		return "", err
	}
	return filePath, nil
}

func (s FileService) UploadBase64(params map[string]string) (string, error) {
	data := params["data"]
	if data == "" {
		global.NAV_LOG.Error("function UploadBase64() failed, data is empty")
		return "", errors.New("function UploadBase64() failed, data is empty")
	}
	limit := global.Config().System.MaxUploadBytes
	if limit <= 0 {
		limit = 32 << 20
	}
	if limit < math.MaxInt64/4 && int64(len(data)) > ((limit+2)/3)*4 {
		return "", errors.New("上传文件超过大小限制")
	}
	bytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", errors.New("decode base64 failed err:" + err.Error())
	}
	if int64(len(bytes)) > limit {
		return "", errors.New("上传文件超过大小限制")
	}
	mkdirErr := os.MkdirAll(s.storageRoot(), 0755)
	if mkdirErr != nil {
		global.NAV_LOG.Error("function os.MkdirAll() failed", zap.Any("err", mkdirErr.Error()))
		return "", errors.New("function os.MkdirAll() failed, err:" + mkdirErr.Error())
	}
	ext := ".png"
	fileName := params["fileName"]
	if fileName != "" {
		ext = filepath.Ext(fileName)
	}
	filename := utils.GenerateGuid() + ext
	p := filepath.Join(s.storageRoot(), filename)
	filePath := s.storageURL(filename)
	if err := os.WriteFile(p, bytes, 0644); err != nil {
		_ = os.Remove(p)
		return "", err
	}
	return filePath, nil
}

func (s FileService) DownloadFile(path string, c *gin.Context) error {
	if path == "" {
		return errors.New("路径不能为空")
	}
	if strings.Contains(path, "..") || strings.ContainsAny(path, `\:*?"<>|`) {
		return errors.New("非法的路径")
	}
	path = strings.TrimPrefix(path, global.Config().System.RouterPrefix+global.Config().Local.ProxyOssPath+"/")
	if s.company != "" && !strings.HasPrefix(path, s.company+"/") {
		return errors.New("无权访问其他企业文件")
	}
	root, err := filepath.EvalSymlinks(global.Config().Local.OssPath)
	if err == nil {
		root, err = filepath.Abs(root)
	}
	if err != nil {
		return err
	}
	fullPath, err := filepath.EvalSymlinks(filepath.Join(root, path))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, fullPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("非法路径")
	}
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return errors.New("文件不存在")
	}
	fileName := filepath.Base(fullPath)

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/octet-stream")

	file, err := os.Open(fullPath)
	if err != nil {
		return errors.New("无法打开文件")
	}
	defer file.Close()
	_, err = io.Copy(c.Writer, file)
	if err != nil {
		return errors.New("无法发送文件")
	}
	return nil
}

func (s FileService) DeleteFile(path string) error {
	if path == "" {
		return errors.New("路径不能为空")
	}
	if strings.Contains(path, "..") || strings.ContainsAny(path, `\/:*?"<>|`) {
		return errors.New("非法的路径")
	}
	p := filepath.Join(global.Config().Local.OssPath, path)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return errors.New("文件不存在")
	}

	mu.Lock()
	defer mu.Unlock()

	err := os.Remove(p)
	if err != nil {
		return errors.New("文件删除失败: " + err.Error())
	}
	return nil
}
