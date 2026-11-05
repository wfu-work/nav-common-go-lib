package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/services"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PermissionApi struct{}

type RolePermissionUpdateRequest struct {
	PermissionGuids []string `json:"permissionGuids"`
}

// Save 创建或更新系统权限
// @Summary 创建或更新系统权限
// @Description 创建或更新系统权限
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body domains.SysPermission true "系统权限对象"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /permission [post]
// @Router /permission [put]
func (i PermissionApi) Save(c *gin.Context) {
	if !platformOnly(c) {
		return
	}
	var entity domains.SysPermission
	if err := c.ShouldBindJSON(&entity); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := permissionService.Save(entity); err != nil {
		global.NAV_LOG.Error("保存权限失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	if !refreshCasbinAfterPermissionChange(c) {
		return
	}
	response.Ok(true, c)
}

// DeleteByGuid 根据guid删除系统权限
// @Summary 根据guid删除系统权限
// @Description 根据guid删除系统权限
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "系统权限guid"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /permission/{guid} [delete]
func (i PermissionApi) DeleteByGuid(c *gin.Context) {
	if !platformOnly(c) {
		return
	}
	guid := c.Param("guid")
	if err := permissionService.DeletePermission(guid); err != nil {
		global.NAV_LOG.Error("删除权限失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	if !refreshCasbinAfterPermissionChange(c) {
		return
	}
	response.Ok(true, c)
}

// List 分页获取系统权限列表
// @Summary 分页获取系统权限列表
// @Description 分页获取系统权限列表
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data query domains.PageInfo true "页码, 每页大小"
// @Success 200 {object} response.Response{data=domains.PageResult,msg=string}
// @Router /permission/list [get]
func (i PermissionApi) List(c *gin.Context) {
	var pageInfo domains.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := utils.Verify(pageInfo, utils.PageInfoVerify); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	svc := permissionService.WithDB(permissionDB(c))
	list, total, err := svc.List(pageInfo, "name,code,path,method")
	if err != nil {
		global.NAV_LOG.Error("获取权限列表失败", zap.Error(err))
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

// ListAll 获取所有权限列表
// @Summary 获取所有权限列表
// @Description 获取所有权限列表
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]domains.SysPermission,msg=string}
// @Router /permission/list/all [get]
func (i PermissionApi) ListAll(c *gin.Context) {
	query := c.Request.URL.Query()
	params := make(map[string]any)
	for key, val := range query {
		if len(val) > 0 {
			params[key] = val[0]
		}
	}
	result, err := permissionService.WithDB(permissionDB(c)).ListByFields(params)
	if err != nil {
		global.NAV_LOG.Error("获取权限列表失败", zap.Error(err))
		response.Fail(nil, c)
		return
	}
	response.Ok(result, c)
}

// RolePermissions 获取角色权限
// @Summary 获取角色权限
// @Description 获取角色权限
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "角色guid"
// @Success 200 {object} response.Response{data=[]domains.SysPermission,msg=string}
// @Router /role/{guid}/permissions [get]
func (i PermissionApi) RolePermissions(c *gin.Context) {
	if !checkRoleScope(c) {
		return
	}
	result, err := permissionService.RolePermissions(c.Param("guid"))
	if err != nil {
		global.NAV_LOG.Error("获取角色权限失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// SetRolePermissions 设置角色权限
// @Summary 设置角色权限
// @Description 设置角色权限
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "角色guid"
// @Param data body RolePermissionUpdateRequest true "权限GUID列表"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /role/{guid}/permissions [put]
func (i PermissionApi) SetRolePermissions(c *gin.Context) {
	if !checkRoleScope(c) {
		return
	}
	var req RolePermissionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := permissionService.SetRolePermissions(c.Param("guid"), req.PermissionGuids); err != nil {
		global.NAV_LOG.Error("设置角色权限失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	if !refreshCasbinAfterPermissionChange(c) {
		return
	}
	response.Ok(true, c)
}

func refreshCasbinAfterPermissionChange(c *gin.Context) bool {
	if err := middlewares.RefreshCasbin(); err != nil {
		global.NAV_LOG.Error("刷新Casbin策略失败", zap.Error(err))
		response.FailWithMessage("权限已保存，但刷新权限缓存失败："+err.Error(), c)
		c.Abort()
		return false
	}
	return true
}

func permissionDB(c *gin.Context) *gorm.DB {
	db := global.NAV_DB
	a := services.RequestAccess(c)
	if a == nil {
		return db.Where("1=0")
	}
	if !a.PlatformAdmin() {
		db = db.Where("scope = ? AND enabled = ?", "company", true)
	}
	return db
}
func checkRoleScope(c *gin.Context) bool {
	var role domains.SysRole
	if err := services.AccessDB(c, &role).Where("guid = ?", c.Param("guid")).First(&role).Error; err != nil || role.CompanyGuid == "" {
		response.NoPermission("无权操作此角色", c)
		return false
	}
	return true
}
