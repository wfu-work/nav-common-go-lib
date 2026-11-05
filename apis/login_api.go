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

// Captcha 获取验证码
// @Summary 获取验证码
// @Description 获取验证码
// @Tags 系统模块
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=vos.TokenVo,msg=string}  "验证码信息"
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

// LoginIn 登录
// @Summary 登录
// @Description 登录
// @Tags 系统模块
// @Accept json
// @Produce json
// @Param data body vos.LoginVo true "登录对象"
// @Success 200 {object} response.Response{data=vos.TokenVo,msg=string}  "登录成功返回token信息"
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
