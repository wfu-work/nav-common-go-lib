package apis

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"navfirst.com/nav-common-go-lib/domains"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/response"
	"navfirst.com/nav-common-go-lib/utils"
)

type RoleApi struct{}

// Save 创建系统角色
// @Summary 创建系统角色
// @Description 创建系统角色
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body domains.SysRole true "系统角色对象"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /role [post]
func (i RoleApi) Save(c *gin.Context) {
	var entity domains.SysRole
	err := c.ShouldBindJSON(&entity)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(entity, utils.RoleVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = roleService.Create(entity)
	if err != nil {
		global.NAV_LOG.Error("创建失败!", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// DeleteByGuid 根据guid删除系统角色
// @Summary 根据系统角色guid删除系统角色
// @Description 根据系统角色guid删除系统角色
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "系统角色guid"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /role/{guid} [delete]
func (i RoleApi) DeleteByGuid(c *gin.Context) {
	guid := c.Param("guid")
	err := roleService.DeleteByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("删除失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// GetByGuid 获取系统角色信息
// @Summary 根据系统角色guid获取系统角色
// @Description 根据系统角色guid获取系统角色
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "系统角色guid"
// @Success 200 {object} response.Response{data=domains.SysRole,msg=string}
// @Router /role/{guid} [get]
func (i RoleApi) GetByGuid(c *gin.Context) {
	guid := c.Param("guid")
	t, err := roleService.GetByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(t, c)
}

// List 分页获取系统角色列表
// @Summary 分页获取系统角色列表
// @Description 分页获取系统角色列表
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data  query domains.PageInfo true  "页码, 每页大小"
// @Success 200  {object}  response.Response{data=domains.PageResult,msg=string}  "分页获取系统角色列表,返回包括列表,总数,页码,每页数量"
// @Router /role/list [get]
func (i RoleApi) List(c *gin.Context) {
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
	list, total, err := roleService.List(pageInfo, "name")
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

// ListAll 获取所有角色列表
// @Summary 获取所有角色列表
// @Description 获取所有角色列表
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data  query any true  "参数"
// @Success 200  {object}  response.Response{data=[]any,msg=string}
// @Router /role/list/all [get]
func (i RoleApi) ListAll(c *gin.Context) {
	query := c.Request.URL.Query()
	params := make(map[string]string)
	for key, val := range query {
		if len(val) > 0 {
			params[key] = val[0]
		}
	}
	result, err := roleService.ListAll(params)
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(nil, c)
		return
	}
	response.Ok(result, c)
}
