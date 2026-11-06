package apis

import (
	"io"

	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

type SecretApi struct{}

// Encrypt 加密原始文本。
//
// @Summary 加密原始文本
// @Description 默认鉴权白名单允许匿名调用。直接读取请求体字节作为待加密文本，使用现有 AES-CBC 协议返回 Base64 密文，供登录及密码管理接口使用。
// @Description 请求体为原始文本，不是 JSON 对象或带引号的 JSON 字符串；例如直接发送密码文本。
// @Tags 加解密
// @Accept plain
// @Produce json
// @Param data body string true "原始明文文本"
// @Success 200 {object} response.Response{data=string} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Router /secret/encrypt [post]
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

// Decrypt 解密原始密文文本。
//
// @Summary 解密原始密文文本
// @Description 需要登录及接口权限。请求体直接发送 AES-CBC 的 Base64 密文，不添加 JSON 包装或引号；成功返回明文，密文无效时返回业务失败。
// @Tags 加解密
// @Security ApiKeyAuth
// @Accept plain
// @Produce json
// @Param data body string true "Base64 密文文本"
// @Success 200 {object} response.Response{data=string} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /secret/decrypt [post]
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
