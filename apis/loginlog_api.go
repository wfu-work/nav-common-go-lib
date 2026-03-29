package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

type LoginLogApi struct{}

// Save 创建登录日志
// @Summary 创建登录日志
// @Description 创建登录日志
// @Tags 登录模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body domains.LoginLog true "登录日志对象"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /log [post]
func (i LoginLogApi) Save(c *gin.Context) {
	var entity domains.LoginLog
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
	err = loginLogService.Create(entity)
	if err != nil {
		global.NAV_LOG.Error("创建失败!", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// DeleteByGuid 根据guid删除登录日志
// @Summary 根据登录日志guid删除登录日志
// @Description 根据登录日志guid删除登录日志
// @Tags 登录模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "登录日志guid"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /log/{guid} [delete]
func (i LoginLogApi) DeleteByGuid(c *gin.Context) {
	guid := c.Param("guid")
	err := loginLogService.DeleteByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("删除失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// GetByGuid 获取登录日志信息
// @Summary 根据登录日志guid获取登录日志
// @Description 根据登录日志guid获取登录日志
// @Tags 登录模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "登录日志guid"
// @Success 200 {object} response.Response{data=domains.SysLog,msg=string}
// @Router /log/{guid} [get]
func (i LoginLogApi) GetByGuid(c *gin.Context) {
	guid := c.Param("guid")
	t, err := loginLogService.GetByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(t, c)
}

// List 分页获取登录日志列表
// @Summary 分页获取登录日志列表
// @Description 分页获取登录日志列表
// @Tags 登录模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data  query domains.PageInfo true  "页码, 每页大小"
// @Success 200  {object}  response.Response{data=domains.PageResult,msg=string}  "分页获取登录日志列表,返回包括列表,总数,页码,每页数量"
// @Router /log/list [get]
func (i LoginLogApi) List(c *gin.Context) {
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
	list, total, err := loginLogService.List(pageInfo, "operation")
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
