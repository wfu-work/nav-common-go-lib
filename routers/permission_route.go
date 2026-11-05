package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type PermissionRouter struct{}

func (s *PermissionRouter) InitPermissionRouter(Router *gin.RouterGroup) {
	routerLogger := Router.Group("permission").Use(middlewares.ApiLogger())
	router := Router.Group("permission")
	{
		routerLogger.POST("", permissionApi.Save)
		routerLogger.PUT("", permissionApi.Save)
		routerLogger.DELETE(":guid", permissionApi.DeleteByGuid)
	}
	{
		router.GET("list", permissionApi.List)
		router.GET("list/all", permissionApi.ListAll)
	}

}
