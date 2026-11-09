package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InitializeGlobalRoles upgrades legacy role references before installing shared
// defaults. The retained company/code unique index enforces global code uniqueness
// because every active role has an empty company GUID.
func InitializeGlobalRoles(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := migrateGlobalRoles(tx); err != nil {
			return err
		}
		for index, preset := range []struct{ code, name string }{{"COMPANY_ADMIN", "企业管理员"}, {"OPERATOR", "运维人员"}, {"VIEWER", "只读人员"}} {
			var role domains.SysRole
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("company_guid = '' AND code = ?", preset.code).First(&role).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				role = domains.SysRole{BaseDataEntity: domains.BaseDataEntity{Guid: uuid.NewString()}, Code: preset.code, Name: preset.name, Enabled: true, BuiltIn: true, Sort: index + 2}
				if err := tx.Create(&role).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
			if role.PermissionsInitialized {
				continue
			}
			var permissions []domains.SysPermission
			if err := tx.Where("scope = ? AND enabled = ? AND type <> ?", "company", true, domains.PermissionTypeAPI).Find(&permissions).Error; err != nil {
				return err
			}
			if len(permissions) == 0 {
				continue
			}
			selected := []string{}
			for _, p := range permissions {
				if (strings.HasPrefix(p.Code, "user.") || strings.HasPrefix(p.Code, "role.")) && preset.code != "COMPANY_ADMIN" {
					continue
				}
				if preset.code != "VIEWER" || p.Type == domains.PermissionTypeMenu || strings.HasSuffix(p.Code, ".read") || strings.HasSuffix(p.Code, ".download") {
					selected = append(selected, p.Guid)
				}
			}
			if err := PermissionServiceApp.WithDB(tx).SetRolePermissions(role.Guid, selected); err != nil {
				return err
			}
			if err := tx.Model(&role).UpdateColumn("permissions_initialized", true).Error; err != nil {
				return err
			}
		}
		return initializeRegisteredRoleMenus(tx, "")
	})
}

func migrateGlobalRoles(tx *gorm.DB) error {
	var roles []domains.SysRole
	// Prefer an explicitly configured global role, then the earliest configured
	// legacy role. Never union legacy grants, which could add privileges.
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Order("CASE WHEN company_guid = '' AND permissions_initialized = true THEN 0 WHEN permissions_initialized = true THEN 1 WHEN company_guid = '' THEN 2 ELSE 3 END, id ASC").Find(&roles).Error; err != nil {
		return err
	}
	presets := map[string]bool{"COMPANY_ADMIN": true, "OPERATOR": true, "VIEWER": true}
	canonical := map[string]domains.SysRole{}
	for _, role := range roles {
		if presets[role.Code] && (role.BuiltIn || role.CompanyGuid == "") && role.Enabled {
			if _, found := canonical[role.Code]; !found {
				canonical[role.Code] = role
			}
		}
	}
	// Deleted global seeds still participate in the unique index. Archive their
	// codes before promoting an active legacy role with that code.
	for code := range canonical {
		var retired []domains.SysRole
		if err := tx.Unscoped().Where("company_guid = '' AND code = ? AND deleted_time IS NOT NULL", code).Find(&retired).Error; err != nil {
			return err
		}
		for _, role := range retired {
			if err := tx.Unscoped().Model(&role).UpdateColumn("code", "retired_"+role.Guid).Error; err != nil {
				return err
			}
		}
	}
	for _, role := range roles {
		target, found := canonical[role.Code]
		if !found || target.Guid == role.Guid || !role.Enabled || (!role.BuiltIn && role.CompanyGuid != "") {
			continue
		}
		var bindings []domains.SysUserRole
		if err := tx.Where("role_guid = ?", role.Guid).Find(&bindings).Error; err != nil {
			return err
		}
		for _, binding := range bindings {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&domains.SysUserRole{UserGuid: binding.UserGuid, RoleGuid: target.Guid}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("role_guid = ?", role.Guid).Delete(&domains.SysUserRole{}).Error; err != nil {
			return err
		}
		// Archive the old definition and grants together for traceability. Rename
		// its code so a deleted global seed cannot block the canonical role.
		if err := tx.Model(&role).UpdateColumn("code", "retired_"+role.Guid).Error; err != nil {
			return err
		}
		if err := tx.Delete(&role).Error; err != nil {
			return err
		}
	}
	roles = nil
	if err := tx.Order("CASE WHEN company_guid = '' THEN 0 ELSE 1 END, id ASC").Find(&roles).Error; err != nil {
		return err
	}
	for _, role := range roles {
		code := role.Code
		target, isPreset := canonical[code]
		builtIn := role.BuiltIn
		if isPreset && target.Guid == role.Guid {
			builtIn = true
		} else if role.CompanyGuid != "" || isPreset {
			// Preserve custom and disabled roles independently, including grants.
			var count int64
			if err := tx.Unscoped().Model(&domains.SysRole{}).Where("company_guid = '' AND code = ? AND guid <> ?", code, role.Guid).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 || isPreset || code == "SUPER_ADMIN" {
				// Numeric IDs are unique; reserve at most 20 runes for the old code.
				prefix := []rune(code)
				if len(prefix) > 20 {
					prefix = prefix[:20]
				}
				for suffix := 1; ; suffix++ {
					code = fmt.Sprintf("%s_%d_%d", string(prefix), role.Id, suffix)
					if err := tx.Unscoped().Model(&domains.SysRole{}).Where("company_guid = '' AND code = ?", code).Count(&count).Error; err != nil {
						return err
					}
					if count == 0 {
						break
					}
				}
				builtIn = false
			}
		}
		if err := tx.Model(&role).Updates(map[string]any{"company_guid": "", "code": code, "built_in": builtIn}).Error; err != nil {
			return err
		}
	}
	return nil
}
