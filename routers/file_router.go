package routers

import (
	"github.com/gin-gonic/gin"
)

type FileRouter struct{}

func (s *FileRouter) InitFileRouter(Router *gin.RouterGroup) {
	router := Router.Group("file")
	{
		router.POST("upload", fileApi.Upload)              // 文件上传
		router.POST("upload/base64", fileApi.UploadBase64) // 文件上传
		router.GET("download", fileApi.Download)           // 文件下载
	}
}
