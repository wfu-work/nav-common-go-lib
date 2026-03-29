package routers

import (
	"github.com/gin-gonic/gin"
)

type ConfigRouter struct{}

func (s *ConfigRouter) InitConfigRouter(pubRouter *gin.RouterGroup) {
	initRouter := pubRouter.Group("config")
	{
		initRouter.GET(":key", configApi.GetConfig)  // 获取配置
		initRouter.PUT(":key", configApi.SaveConfig) // 修改配置
	}
}
