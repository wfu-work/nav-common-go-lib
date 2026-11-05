package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/services"
	"go.uber.org/zap"
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
	header, err := c.FormFile("file")
	if err != nil {
		global.NAV_LOG.Error("上传文件失败!", zap.Error(err))
		response.FailWithMessage("上传文件失败", c)
		return
	}
	scoped, err := requestFileService(c)
	if err != nil {
		accessReply(c, nil, err)
		return
	}
	path, err := scoped.UploadFile(header)
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
	scoped, err := requestFileService(c)
	if err != nil {
		accessReply(c, nil, err)
		return
	}
	path, err := scoped.UploadBase64(params)
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
	scoped, err := requestFileService(c)
	if err != nil {
		accessReply(c, nil, err)
		return
	}
	err = scoped.DownloadFile(path, c)
	if err != nil {
		response.FailWithDetailed(false, err.Error(), c)
		return
	}

}

func requestFileService(c *gin.Context) (services.FileService, error) {
	a := services.RequestAccess(c)
	if a.PlatformAdmin() && c.Query("companyGuid") == "" {
		return *services.FileServiceApp, nil
	}
	company, err := services.TargetCompany(a, c.Query("companyGuid"))
	return services.FileServiceApp.WithCompany("companies/" + company), err
}
