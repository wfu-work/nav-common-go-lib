package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type UserRouter struct{}

func (s *UserRouter) InitUserRouter(Router *gin.RouterGroup) {
	routerLogger := Router.Group("user").Use(middlewares.ApiLogger())
	router := Router.Group("user")
	{
		routerLogger.POST("", userApi.Save)                         // 新增用户
		routerLogger.DELETE(":guid", userApi.DeleteByGuid)          // 根据guid删除用户
		routerLogger.PUT("update/password", userApi.UpdatePassword) // 修改用户密码
	}
	{
		router.GET(":guid", userApi.GetByGuid)      // 根据guid查询用户
		router.GET("list", userApi.List)            // 分页获取用户列表
		router.GET("token", userApi.GetUserByToken) // 获取用户信息
	}
}
