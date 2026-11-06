package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/services"
	"go.uber.org/zap"
)

type FileApi struct{}

// Upload 上传 multipart 文件。
//
// @Summary 上传 multipart 文件
// @Description 企业用户仅访问当前企业 companies/<companyGuid>/ 范围；平台管理员指定 companyGuid 时使用该企业范围，不指定时使用存储根目录。以 multipart/form-data 的 file 字段上传单个文件，返回可访问路径；文件名由服务端生成，保留扩展名。
// @Description 默认上传请求及文件大小上限为 32 MiB；请求上限包含 multipart 开销，可由 system.max-upload-bytes 调整。
// @Tags 文件管理
// @Security ApiKeyAuth
// @Accept mpfd
// @Produce json
// @Param file formData file true "上传文件"
// @Param companyGuid query string false "仅平台管理员可选：限定目标企业；企业用户始终限定当前企业"
// @Success 200 {object} response.Response{data=string} "上传成功，data 为文件访问路径"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Failure 413 "上传请求体超过大小限制，无响应体"
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

// UploadBase64 上传 Base64 文件。
//
// @Summary 上传 Base64 文件
// @Description 企业用户仅访问当前企业 companies/<companyGuid>/ 范围；平台管理员指定 companyGuid 时使用该企业范围，不指定时使用存储根目录。JSON 的 data 必填，必须是标准 Base64 编码字符串，不含 data URL 前缀；fileName 可选，仅用于提取扩展名，省略时使用 .png。
// @Description 默认请求及解码后文件大小上限为 32 MiB；请求上限包含 Base64 与 JSON 开销。
// @Tags 文件管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body object{data=string,fileName=string} true "data 为必填的纯 Base64 字符串；fileName 为可选原文件名"
// @Param companyGuid query string false "仅平台管理员可选：限定目标企业；企业用户始终限定当前企业"
// @Success 200 {object} response.Response{data=string} "上传成功，data 为文件访问路径"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Failure 413 "上传请求体超过大小限制，无响应体"
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

// Download 下载存储文件。
//
// @Summary 下载存储文件
// @Description 企业用户仅访问当前企业 companies/<companyGuid>/ 范围；平台管理员指定 companyGuid 时使用该企业范围，不指定时使用存储根目录。path 可使用上传返回的访问路径，或包含 companies/<companyGuid>/ 的存储相对路径。
// @Description 成功直接返回 application/octet-stream 二进制附件，不包装 Response；路径无效、文件不存在或越权时返回 JSON 业务错误。
// @Tags 文件管理
// @Security ApiKeyAuth
// @Produce octet-stream,json
// @Param path query string true "文件访问路径或存储相对路径"
// @Param companyGuid query string false "仅平台管理员可选：限定目标企业；企业用户始终限定当前企业"
// @Success 200 {file} file "成功返回二进制附件；业务失败时返回 JSON Response，code=400"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
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
