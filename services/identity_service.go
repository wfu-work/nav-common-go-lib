package services

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

// InitializeIdentity is idempotent. It migrates legacy users into a default company,
// clones their role grants, and never turns a role code into platform identity.
func InitializeIdentity(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var users []domains.SysUser
		if err := tx.Find(&users).Error; err != nil {
			return err
		}
		seen := map[string]string{}
		for _, u := range users {
			key := strings.ToLower(strings.TrimSpace(u.Username))
			if key == "" {
				return errors.New("用户名不能为空")
			}
			if previous, ok := seen[key]; ok && previous != u.Guid {
				return fmt.Errorf("用户名重复，需先处理：%s", key)
			}
			seen[key] = u.Guid
			if key == "admin" && u.Guid != domains.BuiltinAdminGUID {
				return errors.New("admin 为系统保留账号，请先处理历史重名账号")
			}
			if err := tx.Model(&u).UpdateColumn("username", key).Error; err != nil {
				return err
			}
		}
		if !tx.Migrator().HasIndex(&domains.SysUser{}, "idx_sys_user_username") {
			if err := tx.Exec("CREATE UNIQUE INDEX idx_sys_user_username ON sys_user (username)").Error; err != nil {
				return err
			}
		}
		legacy := domains.SysCompany{BaseDataEntity: domains.BaseDataEntity{Guid: domains.DefaultCompanyGUID}, Name: "默认企业", Code: "LEGACY", Status: 1, Enabled: true}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&legacy).Error; err != nil {
			return err
		}
		for _, u := range users {
			if u.Guid == domains.BuiltinAdminGUID {
				if err := tx.Model(&u).Updates(map[string]any{"username": "admin", "company_guid": "", "enable": 1}).Error; err != nil {
					return err
				}
				continue
			}
			if u.CompanyGuid == "" {
				u.CompanyGuid = domains.DefaultCompanyGUID
				if err := tx.Model(&u).UpdateColumn("company_guid", u.CompanyGuid).Error; err != nil {
					return err
				}
			}
			var oldRoles []domains.SysRole
			if err := tx.Model(&domains.SysRole{}).Joins("JOIN sys_user_role ur ON ur.role_guid = sys_role.guid").Where("ur.user_guid = ? AND sys_role.company_guid = ''", u.Guid).Find(&oldRoles).Error; err != nil {
				return err
			}
			for _, old := range oldRoles {
				code := old.Code
				if code == "SUPER_ADMIN" {
					code = "VIEWER"
				}
				var role domains.SysRole
				err := tx.Where("company_guid = ? AND code = ?", u.CompanyGuid, code).First(&role).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					role = domains.SysRole{BaseDataEntity: domains.BaseDataEntity{Guid: uuid.NewString()}, CompanyGuid: u.CompanyGuid, Name: old.Name, Code: code, Enabled: true, BuiltIn: code == "COMPANY_ADMIN"}
					if err := tx.Create(&role).Error; err != nil {
						return err
					}
					if old.Code != "SUPER_ADMIN" {
						var grants []domains.SysRolePermission
						if err := tx.Where("role_guid = ?", old.Guid).Find(&grants).Error; err != nil {
							return err
						}
						for _, g := range grants {
							g.Id = 0
							g.RoleGuid = role.Guid
							if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&g).Error; err != nil {
								return err
							}
						}
					}
				} else if err != nil {
					return err
				}
				if err := tx.Where("user_guid = ? AND role_guid = ?", u.Guid, old.Guid).Delete(&domains.SysUserRole{}).Error; err != nil {
					return err
				}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&domains.SysUserRole{UserGuid: u.Guid, RoleGuid: role.Guid}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func InitializeCompanyRoles(db *gorm.DB, company string) error {
	for _, preset := range []struct{ code, name string }{{"COMPANY_ADMIN", "企业管理员"}, {"OPERATOR", "运维人员"}, {"VIEWER", "只读人员"}} {
		var role domains.SysRole
		err := db.Where("company_guid = ? AND code = ?", company, preset.code).First(&role).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			role = domains.SysRole{BaseDataEntity: domains.BaseDataEntity{Guid: uuid.NewString()}, CompanyGuid: company, Code: preset.code, Name: preset.name, Enabled: true, BuiltIn: true}
			if err := db.Create(&role).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if role.PermissionsInitialized {
			continue
		}
		var permissions []domains.SysPermission
		if err := db.Where("scope = ? AND enabled = ? AND type <> ?", "company", true, domains.PermissionTypeAPI).Find(&permissions).Error; err != nil {
			return err
		}
		if len(permissions) == 0 {
			continue
		}
		selected := []string{}
		for _, p := range permissions {
			if preset.code != "VIEWER" || p.Type == domains.PermissionTypeMenu || strings.HasSuffix(p.Code, ".read") || strings.HasSuffix(p.Code, ".download") {
				selected = append(selected, p.Guid)
			}
		}
		if err := PermissionServiceApp.WithDB(db).SetRolePermissions(role.Guid, selected); err != nil {
			return err
		}
		if err := db.Model(&role).UpdateColumn("permissions_initialized", true).Error; err != nil {
			return err
		}
	}
	return nil
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
	var saved string
	err := global.NAV_DB.Transaction(func(tx *gorm.DB) error {
		var u domains.SysUser
		existing := req.Guid != ""
		if existing {
			if err := tx.Where("guid = ?", req.Guid).First(&u).Error; err != nil {
				return err
			}
			if !a.PlatformAdmin() && u.CompanyGuid != a.User.CompanyGuid {
				return errors.New("无权访问其他企业")
			}
		}
		if u.PlatformAdmin() {
			return errors.New("内置 admin 请通过个人中心修改密码或资料")
		}
		company, err := TargetCompany(a, req.CompanyGuid)
		if err != nil {
			return err
		}
		if existing && u.CompanyGuid != company {
			return errors.New("用户企业归属不可直接修改")
		}
		name := strings.ToLower(strings.TrimSpace(req.Username))
		if name == "" || name == "admin" {
			return errors.New("用户名为空或为系统保留账号")
		}
		var n int64
		if err := tx.Model(&domains.SysUser{}).Where("LOWER(username) = ? AND guid <> ?", name, req.Guid).Count(&n).Error; err != nil {
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
		if err := tx.Model(&domains.SysUser{}).Where("guid <> ? AND (LOWER(username) IN ? OR (email <> '' AND LOWER(email) IN ?))", req.Guid, candidates, candidates).Count(&n).Error; err != nil {
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
			u.Password = utils.BcryptHash(plain)
		} else if !existing {
			return errors.New("请设置初始密码")
		}
		if existing {
			u.AuthVersion++
		}
		if err := tx.Save(&u).Error; err != nil {
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
		if err := tx.Where("guid IN ? AND company_guid = ? AND enabled = ?", guids, u.CompanyGuid, true).Find(&roles).Error; err != nil {
			return err
		}
		if len(roles) != len(guids) {
			return errors.New("只能分配本企业的有效角色")
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
	err := tx.Model(&domains.SysUser{}).Joins("JOIN sys_user_role ur ON ur.user_guid=sys_user.guid JOIN sys_role r ON r.guid=ur.role_guid").Where("sys_user.company_guid = ? AND sys_user.enable=1 AND r.company_guid = ? AND r.code = ? AND r.built_in = ? AND r.enabled = ? AND r.deleted_time IS NULL", company, company, "COMPANY_ADMIN", true, true).Count(&n).Error
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
	return global.NAV_DB.Transaction(func(tx *gorm.DB) error {
		var u domains.SysUser
		if err := tx.Where("guid = ?", guid).First(&u).Error; err != nil {
			return err
		}
		if u.PlatformAdmin() {
			return errors.New("不能删除内置 admin")
		}
		if !a.PlatformAdmin() && u.CompanyGuid != a.User.CompanyGuid {
			return errors.New("无权访问其他企业")
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
