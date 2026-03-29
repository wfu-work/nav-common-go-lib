package routers

import (
	"github.com/gin-gonic/gin"
	"navfirst.com/nav-common-go-lib/middlewares"
)

type RoleRouter struct{}

func (s *RoleRouter) InitRoleRouter(Router *gin.RouterGroup) {
	routerLogger := Router.Group("role").Use(middlewares.ApiLogger())
	router := Router.Group("role")
	{
		routerLogger.POST("", roleApi.Save)                // 新增系统角色
		routerLogger.DELETE(":guid", roleApi.DeleteByGuid) // 删除系统角色
	}
	{
		router.GET(":guid", roleApi.GetByGuid)  // 查询系统角色
		router.GET("list", roleApi.List)        // 分页获取系统角色列表
		router.GET("list/all", roleApi.ListAll) // 获取系统角色列表
	}
}
