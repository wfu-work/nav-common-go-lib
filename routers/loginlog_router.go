package routers

import (
	"github.com/gin-gonic/gin"
	"navfirst.com/nav-common-go-lib/middlewares"
)

type LoginLogRouter struct{}

func (s *LoginLogRouter) InitLoginLogRouter(Router *gin.RouterGroup) {
	routerLogger := Router.Group("login/log").Use(middlewares.ApiLogger())
	router := Router.Group("login/log")
	{
		routerLogger.DELETE(":guid", loginLogApi.DeleteByGuid) // 删除登录日志
	}
	{
		router.GET(":guid", loginLogApi.GetByGuid) // 查询登录日志
		router.GET("list", loginLogApi.List)       // 分页获取登录日志列表
	}
}
