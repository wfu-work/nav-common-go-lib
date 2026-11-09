package services

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ManagedRoleRequest struct {
	Guid    string `json:"guid"`
	Name    string `json:"name"`
	Code    string `json:"code"`
	Remark  string `json:"remark"`
	Enabled *bool  `json:"enabled"`
}

func managedRoleAccess(actor *AccessContext) error {
	if actor == nil || !actor.PlatformAdmin() {
		return errors.New("仅平台管理员可管理全局角色")
	}
	return nil
}
func (s *RoleService) GetManagedRole(actor *AccessContext, guid string) (domains.SysRole, error) {
	var role domains.SysRole
	if err := managedRoleAccess(actor); err != nil {
		return role, err
	}
	query := s.DB().Where("guid = ? AND company_guid = '' AND code <> ?", guid, "SUPER_ADMIN")
	err := query.First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = errors.New("角色不存在或无权访问")
	}
	return role, err
}
func (s *RoleService) SaveManagedRole(actor *AccessContext, req ManagedRoleRequest) (domains.SysRole, error) {
	var saved domains.SysRole
	if err := managedRoleAccess(actor); err != nil {
		return saved, err
	}
	req.Name, req.Code, req.Remark = strings.TrimSpace(req.Name), strings.ToUpper(strings.TrimSpace(req.Code)), strings.TrimSpace(req.Remark)
	if req.Name == "" || utf8.RuneCountInString(req.Name) > 50 || req.Code == "" || utf8.RuneCountInString(req.Code) > 50 || req.Code == "SUPER_ADMIN" || utf8.RuneCountInString(req.Remark) > 500 {
		return saved, errors.New("请填写有效的角色名称、编码和备注")
	}
	err := s.DB().Transaction(func(tx *gorm.DB) error {
		role := domains.SysRole{}
		if req.Guid != "" {
			var err error
			role, err = s.WithDB(tx.Clauses(clause.Locking{Strength: "UPDATE"})).GetManagedRole(actor, req.Guid)
			if err != nil {
				return err
			}
		} else {
			if req.Code == "COMPANY_ADMIN" {
				return errors.New("企业管理员是内置角色")
			}
			role.Guid = uuid.NewString()
			role.Enabled = true
			role.PermissionsInitialized = true
			role.MenusInitialized = true
		}
		if role.BuiltIn && role.Code != req.Code {
			return errors.New("内置角色编码不能修改")
		}
		if role.BuiltIn && role.Code == "COMPANY_ADMIN" && req.Enabled != nil && !*req.Enabled {
			return errors.New("不能停用企业管理员角色")
		}
		var count int64
		if err := tx.Unscoped().Model(&domains.SysRole{}).Where("company_guid = '' AND code = ? AND guid <> ?", req.Code, role.Guid).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("角色编码已存在")
		}
		role.CompanyGuid, role.Name, role.Code, role.Remark = "", req.Name, req.Code, req.Remark
		if req.Enabled != nil {
			role.Enabled = *req.Enabled
		}
		enabled := role.Enabled
		if err := tx.Save(&role).Error; err != nil {
			return err
		}
		if err := tx.Model(&role).UpdateColumn("enabled", enabled).Error; err != nil {
			return err
		}
		role.Enabled = enabled
		saved = role
		return nil
	})
	return saved, err
}
func (s *RoleService) DeleteManagedRole(actor *AccessContext, guid string) error {
	return s.DB().Transaction(func(tx *gorm.DB) error {
		role, err := s.WithDB(tx.Clauses(clause.Locking{Strength: "UPDATE"})).GetManagedRole(actor, guid)
		if err != nil {
			return err
		}
		if role.BuiltIn {
			return errors.New("不能删除内置角色")
		}
		var count int64
		if err := tx.Model(&domains.SysUserRole{}).Where("role_guid = ?", role.Guid).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("请先移除角色关联的用户")
		}
		if err := tx.Where("role_guid = ?", role.Guid).Delete(&domains.SysRolePermission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_guid = ?", role.Guid).Delete(&domains.SysRoleMenu{}).Error; err != nil {
			return err
		}
		return tx.Delete(&role).Error
	})
}
