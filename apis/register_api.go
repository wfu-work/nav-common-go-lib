package apis

import (
	"github.com/gin-gonic/gin"
	"navfirst.com/nav-common-go-lib/response"
	"navfirst.com/nav-common-go-lib/utils"
	"navfirst.com/nav-common-go-lib/vos"
)

type RegisterApi struct{}

// RegisterUser 系统用户注册
// @Summary 系统用户注册
// @Description 系统用户注册
// @Tags 系统模块
// @Accept json
// @Produce json
// @Param data body vos.LoginVo true "系统用户注册对象"
// @Success 200 {object} response.Response{data=bool,msg=string}  "注册成功返回值"
// @Router /register/user [post]
func (a RegisterApi) RegisterUser(c *gin.Context) {
	var entity vos.RegisterVo
	err := c.ShouldBindJSON(&entity)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(entity, utils.RegisterVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	res, err := registerService.RegisterUser(&entity)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(res, c)
}
