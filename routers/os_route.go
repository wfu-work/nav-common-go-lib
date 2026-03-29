package routers

import (
	"github.com/gin-gonic/gin"
)

type OsRouter struct{}

func (s *OsRouter) InitOsRouter(pubRouter *gin.RouterGroup) {
	osRouter := pubRouter.Group("os")
	{
		osRouter.GET("info", osApi.GetInfo)              // 获取服务器状态
		osRouter.GET("check/port", osApi.CheckLocalPort) // 获取端口状态
	}
	{
		osRouter.POST("restart/service", osApi.RestartService) // 重启服务
	}
}
