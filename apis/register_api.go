package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/response"
)

type RegisterApi struct{}

// RegisterUser 用户自助注册入口（已停用）。
//
// @Summary 用户自助注册入口（已停用）
// @Description 当前不提供自助注册，始终返回 code=400，提示通过企业用户管理创建账号；请求体不会被处理。默认装配仍受登录与接口权限校验。
// @Tags 账号注册
// @Security ApiKeyAuth
// @Produce json
// @Failure 200 {object} response.Response "该接口始终返回业务失败，code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Deprecated
// @Router /register/user [post]
func (RegisterApi) RegisterUser(c *gin.Context) {
	response.FailWithMessage("请通过企业用户管理创建账号", c)
}
