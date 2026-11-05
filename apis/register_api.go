package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/response"
)

type RegisterApi struct{}

func (RegisterApi) RegisterUser(c *gin.Context) {
	response.FailWithMessage("请通过企业用户管理创建账号", c)
}
