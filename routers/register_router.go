package routers

import (
	"github.com/gin-gonic/gin"
)

type RegisterRouter struct{}

func (s *RegisterRouter) InitRegisterRouter(RouterPub *gin.RouterGroup) {
	pubRouter := RouterPub.Group("register")
	{
		pubRouter.POST("user", registerApi.RegisterUser) // 注册用户
	}
}
