package services

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"gorm.io/gorm"
)

type AccessContext struct {
	User        domains.SysUser
	Roles       []domains.SysRole
	Permissions []domains.SysPermission
}

func (a *AccessContext) PlatformAdmin() bool { return a != nil && a.User.PlatformAdmin() }

func (a *AccessContext) Has(code string) bool {
	if a.PlatformAdmin() {
		return true
	}
	if a == nil {
		return false
	}
	for _, p := range a.Permissions {
		if p.Code == code {
			return true
		}
	}
	return false
}

func (a *AccessContext) CompanyAdmin() bool {
	if a.PlatformAdmin() {
		return true
	}
	if a == nil {
		return false
	}
	for _, r := range a.Roles {
		if r.BuiltIn && r.Code == "COMPANY_ADMIN" && r.CompanyGuid == a.User.CompanyGuid {
			return true
		}
	}
	return false
}

func RequestAccess(c *gin.Context) *AccessContext {
	if v, ok := c.Get("access"); ok {
		return v.(*AccessContext)
	}
	return nil
}

func LoadAccess(db *gorm.DB, guid string) (*AccessContext, error) {
	a := &AccessContext{}
	if err := db.Where("guid = ?", guid).First(&a.User).Error; err != nil {
		return nil, err
	}
	if a.User.Enable != 1 {
		return nil, errors.New("账号已停用")
	}
	a.User.IsPlatformAdmin = a.PlatformAdmin()
	a.User.DataScope = "company"
	if a.PlatformAdmin() {
		a.User.DataScope = "all"
		a.User.RoleCodeList = []string{"SUPER_ADMIN"}
		return a, nil
	}
	if a.User.CompanyGuid == "" {
		return nil, errors.New("账号未归属企业")
	}
	var company domains.SysCompany
	if err := db.Where("guid = ?", a.User.CompanyGuid).First(&company).Error; err != nil {
		return nil, errors.New("企业不存在")
	}
	if !company.Enabled || company.Status != 1 || (company.EndTime > 0 && company.EndTime < time.Now().UnixMilli()) {
		return nil, errors.New("企业未启用或已到期")
	}
	a.User.Company = &domains.CompanySummary{Guid: company.Guid, Name: company.Name, Logo: company.Logo, Enabled: company.Enabled}
	if err := db.Model(&domains.SysRole{}).Joins("JOIN sys_user_role ur ON ur.role_guid = sys_role.guid").Where("ur.user_guid = ? AND sys_role.company_guid = ? AND sys_role.enabled = ?", guid, a.User.CompanyGuid, true).Find(&a.Roles).Error; err != nil {
		return nil, err
	}
	var guids []string
	for _, role := range a.Roles {
		guids = append(guids, role.Guid)
		a.User.RoleGuidList = append(a.User.RoleGuidList, role.Guid)
		a.User.RoleCodeList = append(a.User.RoleCodeList, role.Code)
	}
	if len(guids) > 0 {
		if err := db.Model(&domains.SysPermission{}).Distinct("sys_permission.*").Joins("JOIN sys_role_permission rp ON rp.permission_guid = sys_permission.guid").Where("rp.role_guid IN ? AND sys_permission.enabled = ? AND sys_permission.scope = ?", guids, true, "company").Find(&a.Permissions).Error; err != nil {
			return nil, err
		}
	}
	a.User.AbilityList = []string{}
	for _, p := range a.Permissions {
		a.User.AbilityList = append(a.User.AbilityList, p.Code)
	}
	a.User.IsCompanyAdmin = a.CompanyAdmin()
	a.User.Abilities = a.User.AbilityList
	return a, nil
}

func AccessDB(c *gin.Context, model any) *gorm.DB {
	db := global.NAV_DB.WithContext(c.Request.Context()).Model(model)
	a := RequestAccess(c)
	if a == nil {
		return db.Where("1 = 0")
	}
	if a.PlatformAdmin() {
		if company := c.Query("companyGuid"); company != "" {
			return db.Where("company_guid = ?", company)
		}
		return db
	}
	return db.Where("company_guid = ?", a.User.CompanyGuid)
}

func TargetCompany(a *AccessContext, requested string) (string, error) {
	return targetCompanyDB(global.NAV_DB, a, requested)
}

func targetCompanyDB(db *gorm.DB, a *AccessContext, requested string) (string, error) {
	if a == nil {
		return "", errors.New("未登录")
	}
	if !a.PlatformAdmin() {
		if requested != "" && requested != a.User.CompanyGuid {
			return "", errors.New("无权访问其他企业")
		}
		return a.User.CompanyGuid, nil
	}
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "", errors.New("请选择企业")
	}
	var n int64
	if err := db.Model(&domains.SysCompany{}).Where("guid = ?", requested).Count(&n).Error; err != nil {
		return "", err
	}
	if n != 1 {
		return "", errors.New("企业不存在")
	}
	return requested, nil
}
