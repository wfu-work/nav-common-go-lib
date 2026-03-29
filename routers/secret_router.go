package routers

import (
	"github.com/gin-gonic/gin"
)

type SecretRouter struct{}

func (s *SecretRouter) InitSecretRouter(RouterPub *gin.RouterGroup) {
	pubRouter := RouterPub.Group("secret")
	{
		pubRouter.POST("encrypt", secretApi.Encrypt) // 加密数据
		pubRouter.POST("decrypt", secretApi.Decrypt) // 解密数据
	}
}
