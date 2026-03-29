package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type LoginRouter struct{}

func (s *LoginRouter) InitLoginRouter(Router *gin.RouterGroup, RouterPub *gin.RouterGroup) {
	routerLogger := Router.Group("login").Use(middlewares.LoginLogger())
	pubRouterLogger := RouterPub.Group("login").Use(middlewares.LoginLogger())
	pubRouter := RouterPub.Group("login")
	{
		pubRouterLogger.POST("in", loginApi.LoginIn) // 登录
		routerLogger.POST("out", loginApi.LoginOut)  // 登出
	}
	{
		pubRouter.GET("captcha", loginApi.Captcha) // 获取验证码
	}
}
