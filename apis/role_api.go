package apis

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/services"
	"gorm.io/gorm"
)

type RoleApi struct{}

func (RoleApi) Save(c *gin.Context) {
	var req struct {
		Guid        string `json:"guid"`
		CompanyGuid string `json:"companyGuid"`
		Name        string `json:"name"`
		Code        string `json:"code"`
		Remark      string `json:"remark"`
		Enabled     *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		accessReply(c, nil, err)
		return
	}
	company, err := services.TargetCompany(services.RequestAccess(c), req.CompanyGuid)
	if err != nil {
		accessReply(c, nil, err)
		return
	}
	err = global.NAV_DB.Transaction(func(tx *gorm.DB) error {
		var role domains.SysRole
		if req.Guid != "" {
			if err := tx.Where("guid = ? AND company_guid = ?", req.Guid, company).First(&role).Error; err != nil {
				return err
			}
		}
		code := strings.ToUpper(strings.TrimSpace(req.Code))
		if code == "" || req.Name == "" || code == "SUPER_ADMIN" {
			return errors.New("角色名称或编码无效")
		}
		if role.Guid == "" && code == "COMPANY_ADMIN" {
			return errors.New("企业管理员是内置角色")
		}
		if role.BuiltIn && code != role.Code {
			return errors.New("内置角色编码不能修改")
		}
		if req.Enabled != nil && !*req.Enabled && role.Code == "COMPANY_ADMIN" {
			return errors.New("不能停用企业管理员角色")
		}
		if role.Guid == "" {
			role.Guid = uuid.NewString()
			role.Enabled = true
		}
		role.CompanyGuid = company
		role.Name = req.Name
		role.Code = code
		role.Remark = req.Remark
		if req.Enabled != nil {
			role.Enabled = *req.Enabled
		}
		enabled := role.Enabled
		if err := tx.Save(&role).Error; err != nil {
			return err
		}
		return tx.Model(&role).UpdateColumn("enabled", enabled).Error
	})
	if err == nil {
		refreshCasbinAfterPermissionChange(c)
	}
	if !c.IsAborted() {
		accessReply(c, true, err)
	}
}
func (RoleApi) DeleteByGuid(c *gin.Context) {
	var role domains.SysRole
	if err := services.AccessDB(c, &role).Where("guid = ?", c.Param("guid")).First(&role).Error; err != nil {
		accessReply(c, nil, err)
		return
	}
	if role.Code == "COMPANY_ADMIN" || role.Code == "SUPER_ADMIN" {
		response.FailWithMessage("不能删除内置管理员角色", c)
		return
	}
	err := global.NAV_DB.Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&domains.SysUserRole{}).Where("role_guid = ?", role.Guid).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return errors.New("请先移除角色关联的用户")
		}
		if err := tx.Where("role_guid = ?", role.Guid).Delete(&domains.SysRolePermission{}).Error; err != nil {
			return err
		}
		return tx.Delete(&role).Error
	})
	if err == nil {
		refreshCasbinAfterPermissionChange(c)
	}
	if !c.IsAborted() {
		accessReply(c, true, err)
	}
}
func (RoleApi) GetByGuid(c *gin.Context) {
	var role domains.SysRole
	err := services.AccessDB(c, &role).Where("guid = ?", c.Param("guid")).First(&role).Error
	accessReply(c, role, err)
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
