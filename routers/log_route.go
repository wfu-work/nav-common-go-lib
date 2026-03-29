package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type LogRouter struct{}

func (s *LogRouter) InitLogRouter(Router *gin.RouterGroup) {
	routerLogger := Router.Group("log").Use(middlewares.ApiLogger())
	router := Router.Group("log")
	{
		routerLogger.POST("", logApi.Save)                // 新增系统日志
		routerLogger.DELETE(":guid", logApi.DeleteByGuid) // 删除系统日志
	}
	{
		router.GET(":guid", logApi.GetByGuid) // 查询系统日志
		router.GET("list", logApi.List)       // 分页获取系统日志列表
	}
}
