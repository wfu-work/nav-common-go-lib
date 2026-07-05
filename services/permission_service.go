package services

import (
	"errors"
	"strings"

	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PermissionService struct {
	CrudService[domains.SysPermission]
}

type RolePermissionPolicy struct {
	RoleGuid string
	Path     string
	Method   string
}

var PermissionServiceApp = new(PermissionService)

func (s *PermissionService) WithDB(db *gorm.DB) *PermissionService {
	cloned := *s
	cloned.CrudService = *s.CrudService.WithDB(db)
	return &cloned
}

func (s *PermissionService) Save(permission domains.SysPermission) error {
	permission.Name = strings.TrimSpace(permission.Name)
	permission.Code = strings.TrimSpace(permission.Code)
	permission.Type = normalizePermissionType(permission.Type)
	permission.Path = normalizePermissionPath(permission.Path)
	permission.Method = normalizePermissionMethod(permission.Method)
	permission.ParentGuid = strings.TrimSpace(permission.ParentGuid)
	permission.Remark = strings.TrimSpace(permission.Remark)
	if permission.Name == "" {
		return errors.New("permission name is required")
	}
	if permission.Code == "" {
		return errors.New("permission code is required")
	}
	if permission.Type == domains.PermissionTypeAPI && permission.Path == "" {
		return errors.New("permission path is required")
	}
	return s.Create(permission)
}

func (s *PermissionService) RolePermissions(roleGuid string) ([]domains.SysPermission, error) {
	roleGuid = strings.TrimSpace(roleGuid)
	if roleGuid == "" {
		return nil, errors.New("role guid is required")
	}
	var permissions []domains.SysPermission
	err := s.DB().
		Table(domains.SysPermission{}.TableName()+" AS p").
		Select("p.*").
		Joins("JOIN "+domains.SysRolePermission{}.TableName()+" AS rp ON rp.permission_guid = p.guid").
		Where("rp.role_guid = ?", roleGuid).
		Order("p.sort ASC, p.id ASC").
		Find(&permissions).Error
	return permissions, err
}

func (s *PermissionService) DeletePermission(guid string) error {
	guid = strings.TrimSpace(guid)
	if guid == "" {
		return errors.New("permission guid is required")
	}
	return s.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("permission_guid = ?", guid).Delete(&domains.SysRolePermission{}).Error; err != nil {
			return err
		}
		return s.WithDB(tx).DeleteByGuid(guid)
	})
}

func (s *PermissionService) SetRolePermissions(roleGuid string, permissionGuids []string) error {
	roleGuid = strings.TrimSpace(roleGuid)
	if roleGuid == "" {
		return errors.New("role guid is required")
	}
	permissionGuids = normalizeGuidList(permissionGuids)
	return s.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_guid = ?", roleGuid).Delete(&domains.SysRolePermission{}).Error; err != nil {
			return err
		}
		if len(permissionGuids) == 0 {
			return nil
		}
		relations := make([]domains.SysRolePermission, 0, len(permissionGuids))
		for _, permissionGuid := range permissionGuids {
			relations = append(relations, domains.SysRolePermission{
				RoleGuid:       roleGuid,
				PermissionGuid: permissionGuid,
			})
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&relations).Error
	})
}

func (s *PermissionService) GrantRolePermission(roleGuid string, permissionGuid string) error {
	roleGuid = strings.TrimSpace(roleGuid)
	permissionGuid = strings.TrimSpace(permissionGuid)
	if roleGuid == "" || permissionGuid == "" {
		return errors.New("role guid and permission guid are required")
	}
	return s.DB().Clauses(clause.OnConflict{DoNothing: true}).Create(&domains.SysRolePermission{
		RoleGuid:       roleGuid,
		PermissionGuid: permissionGuid,
	}).Error
}

func (s *PermissionService) RevokeRolePermission(roleGuid string, permissionGuid string) error {
	roleGuid = strings.TrimSpace(roleGuid)
	permissionGuid = strings.TrimSpace(permissionGuid)
	if roleGuid == "" || permissionGuid == "" {
		return errors.New("role guid and permission guid are required")
	}
	return s.DB().
		Where("role_guid = ? AND permission_guid = ?", roleGuid, permissionGuid).
		Delete(&domains.SysRolePermission{}).Error
}

func (s *PermissionService) RolePermissionPolicies() ([]RolePermissionPolicy, error) {
	var policies []RolePermissionPolicy
	err := s.DB().
		Table(domains.SysRolePermission{}.TableName()+" AS rp").
		Select("rp.role_guid AS role_guid, p.path AS path, p.method AS method").
		Joins("JOIN "+domains.SysPermission{}.TableName()+" AS p ON p.guid = rp.permission_guid").
		Where("p.enabled = ? AND p.type = ? AND p.path <> ?", true, domains.PermissionTypeAPI, "").
		Find(&policies).Error
	for i := range policies {
		policies[i].Path = normalizePermissionPath(policies[i].Path)
		policies[i].Method = normalizePermissionMethod(policies[i].Method)
	}
	return policies, err
}

func normalizePermissionType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return domains.PermissionTypeAPI
	}
	return value
}

func normalizePermissionPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "*" {
		return value
	}
	if !strings.HasPrefix(value, "/") {
		return "/" + value
	}
	return value
}

func normalizePermissionMethod(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "*"
	}
	return value
}

func normalizeGuidList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
