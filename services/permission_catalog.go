package services

import (
	"errors"

	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SyncCatalog installs the application's authoritative permission definitions.
// Business route classification and dependencies belong to the application.
func (s *PermissionService) SyncCatalog(definitions []domains.SysPermission) error {
	byCode := map[string]domains.SysPermission{}
	byGuid := map[string]domains.SysPermission{}
	for _, p := range definitions {
		if p.Guid == "" || p.Code == "" || p.Name == "" || (p.Scope != "company" && p.Scope != "platform") {
			return errors.New("invalid permission definition")
		}
		if _, ok := byCode[p.Code]; ok {
			return errors.New("duplicate permission code")
		}
		if _, ok := byGuid[p.Guid]; ok {
			return errors.New("duplicate permission guid")
		}
		switch p.Type {
		case domains.PermissionTypeMenu, domains.PermissionTypeButton:
		case domains.PermissionTypeAPI:
			if p.Path == "" || p.Method == "" {
				return errors.New("invalid API permission")
			}
		default:
			return errors.New("invalid permission type")
		}
		byCode[p.Code], byGuid[p.Guid] = p, p
	}
	for _, p := range definitions {
		if p.ParentGuid != "" {
			if _, ok := byGuid[p.ParentGuid]; !ok {
				return errors.New("permission parent is missing")
			}
		}
		for _, code := range p.Dependencies {
			dep, ok := byCode[code]
			if !ok || p.Scope == "company" && dep.Scope != "company" {
				return errors.New("invalid permission dependency")
			}
		}
	}
	return s.DB().Transaction(func(tx *gorm.DB) error {
		codes := []string{}
		for _, p := range definitions {
			codes = append(codes, p.Code)
			p.Enabled = true
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoUpdates: clause.AssignmentColumns([]string{"guid", "name", "remark", "enabled", "type", "parent_guid", "path", "method", "scope", "dependencies", "sort", "deleted_time"})}).Create(&p).Error; err != nil {
				return err
			}
		}
		query := tx.Model(&domains.SysPermission{})
		if len(codes) > 0 {
			query = query.Where("code NOT IN ?", codes)
		} else {
			query = query.Where("1 = 1")
		}
		return query.Update("enabled", false).Error
	})
}

// RefreshRoleGrants expands current business grants to the current API catalog.
func (s *PermissionService) RefreshRoleGrants() error {
	return s.DB().Transaction(func(tx *gorm.DB) error {
		roles := []domains.SysRole{}
		if err := tx.Find(&roles).Error; err != nil {
			return err
		}
		for _, role := range roles {
			grants, err := s.WithDB(tx).RolePermissions(role.Guid)
			if err != nil {
				return err
			}
			ids := []string{}
			for _, p := range grants {
				if p.Enabled && p.Type != domains.PermissionTypeAPI && (role.Code == "SUPER_ADMIN" || p.Scope == "company") {
					ids = append(ids, p.Guid)
				}
			}
			if err := s.WithDB(tx).SetRolePermissions(role.Guid, ids); err != nil {
				return err
			}
		}
		return nil
	})
}
