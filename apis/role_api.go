package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/services"
)

type RoleApi struct{}

func (RoleApi) Save(c *gin.Context) {
	var req services.ManagedRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		accessReply(c, nil, err)
		return
	}
	role, err := services.RoleServiceApp.WithDB(global.NAV_DB.WithContext(c.Request.Context())).SaveManagedRole(services.RequestAccess(c), req)
	if err == nil && !refreshCasbinAfterPermissionChange(c) {
		return
	}
	accessReply(c, role, err)
}
func (RoleApi) DeleteByGuid(c *gin.Context) {
	err := services.RoleServiceApp.WithDB(global.NAV_DB.WithContext(c.Request.Context())).DeleteManagedRole(services.RequestAccess(c), c.Param("guid"))
	if err == nil && !refreshCasbinAfterPermissionChange(c) {
		return
	}
	accessReply(c, true, err)
}
func (RoleApi) GetByGuid(c *gin.Context) {
	role, err := services.RoleServiceApp.WithDB(global.NAV_DB.WithContext(c.Request.Context())).GetManagedRole(services.RequestAccess(c), c.Param("guid"))
	accessReply(c, role, err)
}
func (RoleApi) Authorization(c *gin.Context) {
	service, err := services.NavigationServiceFor(global.NAV_DB.WithContext(c.Request.Context()))
	if err != nil {
		accessReply(c, nil, err)
		return
	}
	data, err := service.RoleAuthorization(services.RequestAccess(c), c.Param("guid"))
	accessReply(c, data, err)
}
func (RoleApi) SaveAuthorization(c *gin.Context) {
	var req services.RoleAuthorizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		accessReply(c, nil, err)
		return
	}
	service, err := services.NavigationServiceFor(global.NAV_DB.WithContext(c.Request.Context()))
	if err != nil {
		accessReply(c, nil, err)
		return
	}
	err = service.SaveRoleAuthorization(services.RequestAccess(c), c.Param("guid"), req)
	if err == nil && !refreshCasbinAfterPermissionChange(c) {
		return
	}
	accessReply(c, true, err)
}
func (RoleApi) List(c *gin.Context) {
	var q domains.PageInfo
	if err := c.ShouldBindQuery(&q); err != nil {
		accessReply(c, nil, err)
		return
	}
	db := services.AccessDB(c, &domains.SysRole{}).Where("company_guid <> ''")
	if q.Content != "" {
		db = db.Where("(name LIKE ? OR code LIKE ?)", "%"+q.Content+"%", "%"+q.Content+"%")
	}
	var n int64
	if err := db.Count(&n).Error; err != nil {
		accessReply(c, nil, err)
		return
	}
	roles := []domains.SysRole{}
	err := db.Scopes(q.Paginate()).Order("sort ASC,id ASC").Find(&roles).Error
	accessReply(c, domains.PageResult{Data: roles, Total: n, Page: q.Page, Size: q.Size}, err)
}
func (RoleApi) ListAll(c *gin.Context) {
	roles := []domains.SysRole{}
	err := services.AccessDB(c, &domains.SysRole{}).Where("company_guid <> ''").Order("sort ASC,id ASC").Find(&roles).Error
	accessReply(c, roles, err)
}
