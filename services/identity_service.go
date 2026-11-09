package services

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InitializeCompanyRoles is retained for hosts using the former company API.
// Role definitions and defaults are shared globally.
func InitializeCompanyRoles(db *gorm.DB, _ string) error {
	return InitializeGlobalRoles(db)
}

type UserSaveRequest struct {
	Guid         string    `json:"guid"`
	CompanyGuid  string    `json:"companyGuid"`
	Username     string    `json:"username"`
	NickName     string    `json:"nickName"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email"`
	Enable       *int      `json:"enable"`
	Password     string    `json:"password"`
	RoleGuidList *[]string `json:"roleGuidList"`
}

func SaveManagedUser(a *AccessContext, req UserSaveRequest) (string, error) {
	if a == nil || !a.CompanyAdmin() {
		return "", errors.New("需要企业管理员权限")
	}
	company, err := TargetCompany(a, req.CompanyGuid)
	if err != nil {
		return "", err
	}
	var saved string
	err = global.NAV_DB.Transaction(func(tx *gorm.DB) error {
		if err := lockCompanyIdentity(tx, company); err != nil {
			return err
		}
		var u domains.SysUser
		existing := req.Guid != ""
		if existing {
			if err := tx.Where("guid = ?", req.Guid).First(&u).Error; err != nil {
				return err
			}
			if u.PlatformAdmin() {
				return errors.New("内置 admin 请通过个人中心修改密码")
			}
			if u.CompanyGuid != company {
				return errors.New("用户企业归属不可直接修改")
			}
		}
		name := strings.ToLower(strings.TrimSpace(req.Username))
		if name == "" || name == "admin" {
			return errors.New("用户名为空或为系统保留账号")
		}
		var n int64
		if err := tx.Unscoped().Model(&domains.SysUser{}).Where("username = ? AND guid <> ?", name, req.Guid).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return errors.New("用户名已存在")
		}
		// Login accepts username and email. Prevent ambiguity across both namespaces.
		email := strings.ToLower(strings.TrimSpace(req.Email))
		candidates := []string{name}
		if email != "" {
			candidates = append(candidates, email)
		}
		if err := tx.Unscoped().Model(&domains.SysUser{}).Where("guid <> ? AND (username IN ? OR (email <> '' AND email IN ?))", req.Guid, candidates, candidates).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return errors.New("登录账号或邮箱已被使用")
		}
		if !existing {
			u.Guid = uuid.NewString()
			u.Enable = 1
			u.AuthVersion = 1
		}
		u.CompanyGuid = company
		u.Username = name
		u.NickName = req.NickName
		u.Phone = req.Phone
		u.Email = email
		if req.Enable != nil {
			u.Enable = *req.Enable
		}
		if u.Enable != 0 && u.Enable != 1 {
			return errors.New("账号状态无效")
		}
		if req.Password != "" {
			plain, err := utils.AesDecrypt(req.Password)
			if err != nil {
				return errors.New("密码解密失败")
			}
			if len(plain) < 8 {
				return errors.New("密码至少8位")
			}
			u.Password, err = utils.BcryptHashE(plain)
			if err != nil {
				return err
			}
		} else if !existing {
			return errors.New("请设置初始密码")
		}
		if existing {
			u.AuthVersion++
		}
		enabled := u.Enable
		if err := tx.Save(&u).Error; err != nil {
			return err
		}
		if err := tx.Model(&u).UpdateColumn("enable", enabled).Error; err != nil {
			return err
		}
		if req.RoleGuidList != nil {
			if err := setUserRoles(tx, u, *req.RoleGuidList); err != nil {
				return err
			}
		}
		if err := ensureCompanyAdmin(tx, company); err != nil && existing {
			return err
		}
		saved = u.Guid
		return nil
	})
	return saved, err
}
func setUserRoles(tx *gorm.DB, u domains.SysUser, guids []string) error {
	if u.PlatformAdmin() {
		return errors.New("不能修改内置 admin 的平台身份")
	}
	guids = normalizeGuidList(guids)
	var roles []domains.SysRole
	if len(guids) > 0 {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("guid IN ? AND company_guid = '' AND code <> ? AND enabled = ?", guids, "SUPER_ADMIN", true).Find(&roles).Error; err != nil {
			return err
		}
		if len(roles) != len(guids) {
			return errors.New("只能分配有效的全局角色")
		}
	}
	if err := tx.Where("user_guid = ?", u.Guid).Delete(&domains.SysUserRole{}).Error; err != nil {
		return err
	}
	for _, role := range roles {
		if role.Code == "SUPER_ADMIN" {
			return errors.New("平台身份不可分配")
		}
		if err := tx.Create(&domains.SysUserRole{UserGuid: u.Guid, RoleGuid: role.Guid}).Error; err != nil {
			return err
		}
	}
	return nil
}
func ensureCompanyAdmin(tx *gorm.DB, company string) error {
	var n int64
	err := tx.Model(&domains.SysUser{}).Joins("JOIN sys_user_role ur ON ur.user_guid=sys_user.guid JOIN sys_role r ON r.guid=ur.role_guid").Where("sys_user.company_guid = ? AND sys_user.enable=1 AND r.company_guid = '' AND r.code = ? AND r.built_in = ? AND r.enabled = ? AND r.deleted_time IS NULL", company, "COMPANY_ADMIN", true, true).Count(&n).Error
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("必须保留至少一位启用的企业管理员")
	}
	return nil
}
func DeleteManagedUser(a *AccessContext, guid string) error {
	if a == nil || !a.CompanyAdmin() {
		return errors.New("需要企业管理员权限")
	}
	var target domains.SysUser
	if err := global.NAV_DB.Where("guid = ?", guid).First(&target).Error; err != nil {
		return err
	}
	if target.PlatformAdmin() {
		return errors.New("不能删除内置 admin")
	}
	if !a.PlatformAdmin() && target.CompanyGuid != a.User.CompanyGuid {
		return errors.New("无权访问其他企业")
	}
	return global.NAV_DB.Transaction(func(tx *gorm.DB) error {
		if err := lockCompanyIdentity(tx, target.CompanyGuid); err != nil {
			return err
		}
		var u domains.SysUser
		if err := tx.Where("guid = ? AND company_guid = ?", guid, target.CompanyGuid).First(&u).Error; err != nil {
			return err
		}
		if err := tx.Where("user_guid = ?", guid).Delete(&domains.SysUserRole{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&u).Error; err != nil {
			return err
		}
		return ensureCompanyAdmin(tx, u.CompanyGuid)
	})
}

// Mutations that can remove the last administrator serialize on this row before
// reading role assignments; SQLite serializes its writers itself.
func lockCompanyIdentity(tx *gorm.DB, company string) error {
	var c domains.SysCompany
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("guid = ?", company).First(&c).Error
}
