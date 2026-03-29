package apis

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"navfirst.com/nav-common-go-lib/domains"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/response"
	"navfirst.com/nav-common-go-lib/utils"
)

type LogApi struct{}

// Save 创建系统日志
// @Summary 创建系统日志
// @Description 创建系统日志
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body domains.SysLog true "系统日志对象"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /log [post]
func (i LogApi) Save(c *gin.Context) {
	var entity domains.SysLog
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
	err = loggerService.Create(entity)
	if err != nil {
		global.NAV_LOG.Error("创建失败!", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// DeleteByGuid 根据guid删除系统日志
// @Summary 根据系统日志guid删除系统日志
// @Description 根据系统日志guid删除系统日志
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "系统日志guid"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /log/{guid} [delete]
func (i LogApi) DeleteByGuid(c *gin.Context) {
	guid := c.Param("guid")
	err := loggerService.DeleteByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("删除失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// GetByGuid 获取系统日志信息
// @Summary 根据系统日志guid获取系统日志
// @Description 根据系统日志guid获取系统日志
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "系统日志guid"
// @Success 200 {object} response.Response{data=domains.SysLog,msg=string}
// @Router /log/{guid} [get]
func (i LogApi) GetByGuid(c *gin.Context) {
	guid := c.Param("guid")
	t, err := loggerService.GetByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(t, c)
}

// List 分页获取系统日志列表
// @Summary 分页获取系统日志列表
// @Description 分页获取系统日志列表
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data  query domains.PageInfo true  "页码, 每页大小"
// @Success 200  {object}  response.Response{data=domains.PageResult,msg=string}  "分页获取系统日志列表,返回包括列表,总数,页码,每页数量"
// @Router /log/list [get]
func (i LogApi) List(c *gin.Context) {
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
	list, total, err := loggerService.List(pageInfo, "operation")
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
