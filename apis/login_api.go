package apis

import (
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/services"
	"gorm.io/gorm"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"github.com/wfu-work/nav-common-go-lib/vos"
	"go.uber.org/zap"
)

type LoginApi struct{}

// 当开启多服务器部署时，替换下面的配置，使用redis共享存储验证码
// var store = captcha.NewDefaultRedisStore()
var store = base64Captcha.DefaultMemStore

// Captcha 获取图片验证码。
//
// @Summary 获取图片验证码
// @Description 公开接口，返回 captchaId、Base64 图片 captchaImg、captchaLength 及是否需要验证码的 openCaptcha。openCaptcha 由配置阈值和客户端 IP 计数决定；当前登录请求只接收账号和密码，不在此接口校验验证码。
// @Tags 登录认证
// @Produce json
// @Success 200 {object} response.Response{data=vos.SysCaptcha} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Router /login/captcha [get]
func (a LoginApi) Captcha(c *gin.Context) {
	openCaptcha := global.RequestConfig(c).Captcha.OpenCaptcha
	openCaptchaTimeOut := global.RequestConfig(c).Captcha.OpenCaptchaTimeOut
	key := c.ClientIP()
	v, ok := global.LOCAL_CACHE.Get(key)
	if !ok {
		global.LOCAL_CACHE.Set(key, 1, time.Second*time.Duration(openCaptchaTimeOut))
	}
	var oc bool
	if openCaptcha == 0 || openCaptcha < int(utils.InterfaceToInt(v)) {
		oc = true
	}
	// 字符,公式,验证码配置
	// 生成默认数字的driver
	driver := base64Captcha.NewDriverDigit(global.RequestConfig(c).Captcha.ImgHeight, global.RequestConfig(c).Captcha.ImgWidth, global.RequestConfig(c).Captcha.KeyLong, 0.7, 80)
	cp := base64Captcha.NewCaptcha(driver, store)
	id, b64s, _, err := cp.Generate()
	if err != nil {
		global.NAV_LOG.Error("验证码获取失败!", zap.Error(err))
		response.Fail(nil, c)
		return
	}
	response.Ok(vos.SysCaptcha{
		CaptchaId:     id,
		CaptchaImg:    b64s,
		CaptchaLength: global.RequestConfig(c).Captcha.KeyLong,
		OpenCaptcha:   oc,
	}, c)
}

// LoginIn 账号密码登录。
//
// @Summary 账号密码登录
// @Description 公开接口。username 支持用户名或邮箱；password 必须先经 POST /secret/encrypt 加密。服务端校验密码及实时用户、企业状态。
// @Description 成功返回访问令牌并写入 Authorization Cookie；expired 为 Unix 毫秒时间戳，tokenType 为 Bearer，refreshToken 当前与 token 相同，不是独立刷新令牌。
// @Tags 登录认证
// @Accept json
// @Produce json
// @Param data body vos.LoginVo true "登录账号与加密密码"
// @Success 200 {object} response.Response{data=vos.TokenVo} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Router /login/in [post]
func (a LoginApi) LoginIn(c *gin.Context) {
	var entity vos.LoginVo
	err := c.ShouldBindJSON(&entity)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(entity, utils.LoginVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	tokenVo, err := loginService.LoginIn(&entity)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	utils.SetToken(c, tokenVo.Token, int((tokenVo.Expired-time.Now().UnixMilli())/1000))
	response.OkWithData(tokenVo, c)
}

// LoginOut 退出当前账号。
//
// @Summary 退出当前账号
// @Description 仅需有效登录身份。提升当前账号 auth_version，使该账号既有令牌失效，并清除 Authorization Cookie；该操作影响该账号的其他现有会话。
// @Tags 登录认证
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Router /login/out [post]
func (a LoginApi) LoginOut(c *gin.Context) {
	aCtx := services.RequestAccess(c)
	if aCtx == nil {
		response.NoAuth("未登录", c)
		return
	}
	global.NAV_DB.Model(&domains.SysUser{}).Where("guid = ?", aCtx.User.Guid).UpdateColumn("auth_version", gorm.Expr("auth_version + 1"))
	utils.ClearToken(c)
	response.Ok(true, c)
}
