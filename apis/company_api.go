package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

type CompanyApi struct{}

// Save 创建企业
// @Summary 创建企业
// @Description 创建企业
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body domains.SysCompany true "企业对象"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /company [post]
func (i CompanyApi) Save(c *gin.Context) {
	var entity domains.SysCompany
	err := c.ShouldBindJSON(&entity)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(entity, utils.NameVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = companyService.Create(entity)
	if err != nil {
		global.NAV_LOG.Error("创建失败!", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// DeleteByGuid 根据guid删除企业
// @Summary 根据企业guid删除企业
// @Description 根据企业guid删除企业
// @Tags 系统模块
// @Security  ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "企业guid"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /company/{guid} [delete]
func (i CompanyApi) DeleteByGuid(c *gin.Context) {
	guid := c.Param("guid")
	err := companyService.DeleteByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("删除失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// GetByGuid 获取企业信息
// @Summary 根据企业guid获取企业
// @Description 根据企业guid获取企业
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "企业guid"
// @Success 200 {object} domains.SysCompany
// @Router /company/{guid} [get]
func (i CompanyApi) GetByGuid(c *gin.Context) {
	guid := c.Param("guid")
	t, err := companyService.GetByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(t, c)
}

// List 分页获取企业列表
// @Summary 分页获取企业列表
// @Description 分页获取企业列表
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data query domains.PageInfo true  "页码, 每页大小"
// @Success 200  {object}  response.Response{data=domains.PageResult,msg=string}  "分页获取企业列表,返回包括列表,总数,页码,每页数量"
// @Router /company/list [get]
func (i CompanyApi) List(c *gin.Context) {
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
	list, total, err := companyService.List(pageInfo, "name")
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

// ListAll 获取所有企业列表
// @Summary 获取所有企业列表
// @Description 获取所有企业列表
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data  query any true  "参数"
// @Success 200  {object}  response.Response{data=[]any,msg=string}
// @Router /company/list/all [get]
func (i CompanyApi) ListAll(c *gin.Context) {
	query := c.Request.URL.Query()
	params := make(map[string]string)
	for key, val := range query {
		if len(val) > 0 {
			params[key] = val[0]
		}
	}
	result, err := companyService.ListAll(params)
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(nil, c)
		return
	}
	response.Ok(result, c)
}

// Pass 企业审核通过
// @Summary 企业审核通过
// @Description 企业审核通过
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true  "企业guid"
// @Success 200  {object}  response.Response{data=bool,msg=string}
// @Router /company/pass/:guid [put]
func (i CompanyApi) Pass(c *gin.Context) {
	guid := c.Param("guid")
	t, err := companyService.GetByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	t.Status = 1
	err = companyService.Update(*t, "status", 1)
	if err != nil {
		global.NAV_LOG.Error("更新失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// Refuse 企业审核拒绝
// @Summary 企业审核拒绝
// @Description 企业审核拒绝
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true  "企业guid"
// @Success 200  {object}  response.Response{data=bool,msg=string}
// @Router /company/refuse/:guid [put]
func (i CompanyApi) Refuse(c *gin.Context) {
	guid := c.Param("guid")
	t, err := companyService.GetByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	t.Status = 2
	err = companyService.Update(*t, "status", 2)
	if err != nil {
		global.NAV_LOG.Error("更新失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}
