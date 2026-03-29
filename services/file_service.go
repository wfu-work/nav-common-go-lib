package services

import (
	"encoding/base64"
	"errors"
	"io"
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

type FileService struct{}

var FileServiceApp = new(FileService)

var mu sync.Mutex

func (s FileService) UploadFile(file *multipart.FileHeader) (string, error) {
	ext := filepath.Ext(file.Filename)             // 读取文件后缀
	name := strings.TrimSuffix(file.Filename, ext) // 读取文件名并加密
	name = utils.MD5V([]byte(name))
	filename := utils.GenerateGuid() + ext                                // 拼接新文件名
	mkdirErr := os.MkdirAll(global.NAV_CONFIG.Local.OssPath, os.ModePerm) // 尝试创建此路径
	if mkdirErr != nil {
		global.NAV_LOG.Error("function os.MkdirAll() failed", zap.Any("err", mkdirErr.Error()))
		return "", errors.New("function os.MkdirAll() failed, err:" + mkdirErr.Error())
	}
	p := global.NAV_CONFIG.Local.OssPath + utils.GetPathSeparator() + filename
	filePath := global.NAV_CONFIG.System.RouterPrefix + global.NAV_CONFIG.Local.ProxyOssPath + utils.GetPathSeparator() + filename

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

	_, copyErr := io.Copy(out, f)
	if copyErr != nil {
		global.NAV_LOG.Error("function io.Copy() failed", zap.Any("err", copyErr.Error()))
		return "", errors.New("function io.Copy() failed, err:" + copyErr.Error())
	}
	return filePath, nil
}

func (s FileService) UploadBase64(params map[string]string) (string, error) {
	data := params["data"]
	if data == "" {
		global.NAV_LOG.Error("function UploadBase64() failed, data is empty")
		return "", errors.New("function UploadBase64() failed, data is empty")
	}
	bytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", errors.New("decode base64 failed err:" + err.Error())
	}
	mkdirErr := os.MkdirAll(global.NAV_CONFIG.Local.OssPath, os.ModePerm)
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
	p := global.NAV_CONFIG.Local.OssPath + utils.GetPathSeparator() + filename
	filePath := global.NAV_CONFIG.System.RouterPrefix + global.NAV_CONFIG.Local.ProxyOssPath + utils.GetPathSeparator() + filename
	if err := os.WriteFile(p, bytes, 0644); err != nil {
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
	var fullPath = filepath.Join(global.NAV_CONFIG.Local.OssPath, path)
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
	p := filepath.Join(global.NAV_CONFIG.Local.OssPath, path)
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
