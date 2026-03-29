package apis

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/response"
)

type FileApi struct{}

// Upload 文件上传
// @Summary 文件上传
// @Description 文件上传
// @Tags 系统模块
// @Security ApiKeyAuth
// @accept multipart/form-data
// @Produce json
// @Param file  body any  file  "文件对象"
// @Success 200  {object}  response.Response{data=string,msg=string}
// @Router /file/upload [post]
func (i *FileApi) Upload(c *gin.Context) {
	_, header, err := c.Request.FormFile("file")
	if err != nil {
		global.NAV_LOG.Error("上传文件失败!", zap.Error(err))
		response.FailWithMessage("上传文件失败", c)
		return
	}
	path, err := fileService.UploadFile(header)
	if err != nil {
		global.NAV_LOG.Error("上传文件失败!", zap.Error(err))
		response.FailWithMessage("上传文件失败", c)
		return
	}
	response.Ok(path, c)
}

// UploadBase64 文件上传
// @Summary 文件上传
// @Description 文件上传
// @Tags 系统模块
// @Security ApiKeyAuth
// @accept multipart/form-data
// @Produce json
// @Param file  body any  file  "文件对象"
// @Success 200  {object}  response.Response{data=string,msg=string}
// @Router /file/upload/base64 [post]
func (i *FileApi) UploadBase64(c *gin.Context) {
	params := make(map[string]string)
	if err := c.ShouldBindJSON(&params); err != nil {
		response.FailWithMessage("请求参数错误", c)
		return
	}
	path, err := fileService.UploadBase64(params)
	if err != nil {
		global.NAV_LOG.Error("上传文件失败!", zap.Error(err))
		response.FailWithMessage("上传文件失败", c)
		return
	}
	response.Ok(path, c)
}

// Download 文件下载
// @Summary 文件下载
// @Description 文件下载
// @Tags 系统模块
// @Security ApiKeyAuth
// @accept multipart/form-data
// @Produce json
// @Param path query  string  true  "文件路径"
// @Success 200  {object}  response.Response{data=bool,msg=string}
// @Router /file/download [get]
func (i *FileApi) Download(c *gin.Context) {
	path := c.Query("path")
	err := fileService.DownloadFile(path, c)
	if err != nil {
		response.FailWithDetailed(false, err.Error(), c)
		return
	}
	response.Ok(true, c)
}
