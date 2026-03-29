package apis

import (
	"io"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/response"
	"navfirst.com/nav-common-go-lib/utils"
)

type SecretApi struct{}

// Encrypt 加密
// @Summary 加密
// @Description 加密
// @Tags 系统模块
// @Accept json
// @Produce json
// @Param data body string true "数据"
// @Success 200 {object} response.Response{data=string,msg=string}
// @Router /encrypt [post]
func (i *SecretApi) Encrypt(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		global.NAV_LOG.Error("读取body失败!", zap.Error(err))
		response.Fail(err.Error(), c)
		return
	}
	data := string(bodyBytes)
	result, err := utils.AesEncrypt(data)
	if err != nil {
		global.NAV_LOG.Error("加密失败!", zap.Error(err))
		response.Fail(err.Error(), c)
	}
	response.Ok(result, c)
}

// Decrypt 解密
// @Summary 解密
// @Description 解密
// @Tags 系统模块
// @Accept json
// @Produce json
// @Param data body string true "数据"
// @Success 200 {object} response.Response{data=string,msg=string}
// @Router /decrypt [post]
func (i *SecretApi) Decrypt(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		global.NAV_LOG.Error("读取body失败!", zap.Error(err))
		response.Fail(err.Error(), c)
		return
	}
	data := string(bodyBytes)
	result, err := utils.AesDecrypt(data)
	if err != nil {
		global.NAV_LOG.Error("解密失败!", zap.Error(err))
		response.Fail(err.Error(), c)
	}
	response.Ok(result, c)
}
