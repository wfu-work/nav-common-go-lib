package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/services"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"github.com/wfu-work/nav-common-go-lib/vos"
	"go.uber.org/zap"
)

type UserApi struct{}

// Save 创建用户
// @Summary 创建用户
// @Description 创建用户
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body domains.SysUser true "用户对象"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /user [post]
func (i UserApi) Save(c *gin.Context) {
	var sysUser domains.SysUser
	err := c.ShouldBindJSON(&sysUser)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(sysUser, utils.RoleVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if sysUser.Guid == "" {
		bean, _ := userService.GetByUserName(sysUser.Username)
		if bean != nil {
			response.FailWithMessage("用户名已存在", c)
			return
		}
	}
	err = services.UserServiceApp.Create(sysUser)
	if err != nil {
		global.NAV_LOG.Error("创建失败!", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// DeleteByGuid 根据guid删除用户
// @Summary 根据用户guid删除用户
// @Description 根据用户guid删除用户
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "用户guid"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /user/{guid} [delete]
func (i UserApi) DeleteByGuid(c *gin.Context) {
	guid := c.Param("guid")
	err := userService.DeleteByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("删除失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// GetByGuid 获取用户信息
// @Summary 根据用户guid获取用户
// @Description 根据用户guid获取用户
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "用户guid"
// @Success 200 {object} response.Response{data=domains.SysUser,msg=string}
// @Router /user/{guid} [get]
func (i UserApi) GetByGuid(c *gin.Context) {
	guid := c.Param("guid")
	t, err := userService.GetByGuid(guid)
	if err != nil || t == nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	role, err := userRoleService.GetByUserGuid(t.Guid)
	if err == nil && role != nil {
		t.RoleCodeList = []string{role.Code}
		t.RoleGuidList = []string{role.Guid}
	}
	response.Ok(t, c)
}

// List 分页获取用户列表
// @Summary 分页获取用户列表
// @Description 分页获取用户列表
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data  query domains.PageInfo true  "页码, 每页大小"
// @Success 200  {object}  response.Response{data=domains.PageResult,msg=string}  "分页获取用户列表,返回包括列表,总数,页码,每页数量"
// @Router /user/list [get]
func (i UserApi) List(c *gin.Context) {
	var pageInfo domains.PageInfo
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(pageInfo, utils.PageInfoVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := userService.List(pageInfo, "username,phone")
	if err != nil {
		global.NAV_LOG.Error("获取失败!", zap.Error(err))
		response.Fail(nil, c)
		return
	}
	response.Ok(domains.PageResult{
		Data:  list,
		Total: total,
		Page:  pageInfo.Page,
		Size:  pageInfo.Size,
	}, c)
}

// GetUserByToken 根据token获取用户信息
// @Summary 根据token获取用户信息
// @Description 根据token获取用户信息
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=domains.SysUser,msg=string}
// @Router /user/token [get]
func (i UserApi) GetUserByToken(c *gin.Context) {
	userGuid := utils.GetUserGuid(c)
	if userGuid == "" {
		global.NAV_LOG.Error("未获取到token中的用户guid")
		response.Fail(nil, c)
		return
	}
	t, err := userService.GetByGuid(userGuid)
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(nil, c)
		return
	}
	role, err := userRoleService.GetByUserGuid(t.Guid)
	if err == nil && role != nil {
		t.RoleCodeList = []string{role.Code}
		t.RoleGuidList = []string{role.Guid}
	}
	response.Ok(t, c)
}

// UpdatePassword 修改用户密码
// @Summary 修改用户密码
// @Description 修改用户密码
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /user/update/password [put]
func (i UserApi) UpdatePassword(c *gin.Context) {
	userGuid := utils.GetUserGuid(c)
	if userGuid == "" {
		global.NAV_LOG.Error("未获取到token中的用户guid")
		response.Fail(false, c)
		return
	}
	var passwordVo vos.PasswordVo
	err := c.ShouldBindJSON(&passwordVo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(passwordVo, utils.PasswordVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, err := userService.UpdatePassword(userGuid, &passwordVo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(result, c)
}
