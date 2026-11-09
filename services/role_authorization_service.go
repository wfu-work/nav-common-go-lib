package services

import (
	"errors"

	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoleAuthorizationRequest struct {
	MenuKeys        []string `json:"menuKeys"`
	PermissionGuids []string `json:"permissionGuids"`
}
type RoleAuthorizationData struct {
	Role            domains.SysRole         `json:"role"`
	Menus           []NavigationMenuView    `json:"menus"`
	Permissions     []domains.SysPermission `json:"permissions"`
	MenuKeys        []string                `json:"menuKeys"`
	PermissionGuids []string                `json:"permissionGuids"`
}

func (s *NavigationService) RoleAuthorization(actor *AccessContext, guid string) (RoleAuthorizationData, error) {
	result := RoleAuthorizationData{MenuKeys: []string{}, PermissionGuids: []string{}, Permissions: []domains.SysPermission{}}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		role, err := RoleServiceApp.WithDB(tx).GetManagedRole(actor, guid)
		if err != nil {
			return err
		}
		result.Role = role
		catalog, err := s.WithDB(tx).catalog()
		if err != nil {
			return err
		}
		result.Menus = catalog.Menus
		targets := s.targets()
		codes := map[string]bool{}
		for i := range result.Menus {
			node := &result.Menus[i]
			if node.Type != "menu" {
				continue
			}
			t := targets[node.TargetKey]
			node.RestrictionReason = roleTargetRestriction(role, t)
			node.Assignable = node.RestrictionReason == ""
			for _, code := range t.ACL {
				codes[code] = true
			}
			for _, code := range t.ActionCodes {
				codes[code] = true
			}
		}
		var permissions []domains.SysPermission
		if err := tx.Where("scope = ? AND enabled = ? AND type <> ?", "company", true, domains.PermissionTypeAPI).Order("sort ASC,id ASC").Find(&permissions).Error; err != nil {
			return err
		}
		for _, p := range permissions {
			if codes[p.Code] {
				result.Permissions = append(result.Permissions, p)
			}
		}
		grants, err := PermissionServiceApp.WithDB(tx).RolePermissions(guid)
		if err != nil {
			return err
		}
		for _, p := range grants {
			if p.Enabled {
				result.PermissionGuids = append(result.PermissionGuids, p.Guid)
			}
		}
		rows := []domains.SysRoleMenu{}
		if err := tx.Where("role_guid = ?", guid).Order("menu_key ASC").Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			result.MenuKeys = append(result.MenuKeys, row.MenuKey)
		}
		return nil
	})
	return result, err
}
func (s *NavigationService) SaveRoleAuthorization(actor *AccessContext, guid string, req RoleAuthorizationRequest) error {
	if err := managedRoleAccess(actor); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		nodes, err := s.WithDB(tx.Clauses(clause.Locking{Strength: "UPDATE"})).nodes()
		if err != nil {
			return err
		}
		role, err := RoleServiceApp.WithDB(tx.Clauses(clause.Locking{Strength: "UPDATE"})).GetManagedRole(actor, guid)
		if err != nil {
			return err
		}
		permissions := []domains.SysPermission{}
		if err := tx.Where("enabled = ?", true).Find(&permissions).Error; err != nil {
			return err
		}
		byCode, byGuid := map[string]domains.SysPermission{}, map[string]domains.SysPermission{}
		for _, p := range permissions {
			byCode[p.Code], byGuid[p.Guid] = p, p
		}
		byKey := map[string]domains.SysMenu{}
		for _, node := range nodes {
			byKey[node.Key] = node
		}
		targets := s.targets()
		allowedActions := map[string]bool{}
		grants := []string{}
		keys := normalizeGuidList(req.MenuKeys)
		for _, key := range keys {
			node, exists := byKey[key]
			if !exists || node.Type != "menu" {
				return errors.New("所选菜单不存在，请刷新页面")
			}
			target, exists := targets[node.TargetKey]
			if !exists {
				return errors.New("菜单页面不存在")
			}
			if reason := roleTargetRestriction(role, target); reason != "" {
				return errors.New(reason)
			}
			for _, code := range target.ACL {
				p, exists := byCode[code]
				if !exists || p.Scope != "company" || p.Type != domains.PermissionTypeMenu {
					return errors.New("菜单访问权限不存在或已停用")
				}
				grants = append(grants, p.Guid)
			}
			for _, code := range target.ActionCodes {
				allowedActions[code] = true
			}
		}
		for _, id := range normalizeGuidList(req.PermissionGuids) {
			p, exists := byGuid[id]
			if !exists || p.Scope != "company" || p.Type != domains.PermissionTypeButton || !allowedActions[p.Code] {
				return errors.New("操作权限必须属于已选择菜单的有效操作")
			}
			grants = append(grants, id)
		}
		if err := PermissionServiceApp.WithDB(tx).SetRolePermissions(role.Guid, grants); err != nil {
			return err
		}
		if err := tx.Where("role_guid = ?", role.Guid).Delete(&domains.SysRoleMenu{}).Error; err != nil {
			return err
		}
		for _, key := range keys {
			if err := tx.Create(&domains.SysRoleMenu{RoleGuid: role.Guid, MenuKey: key}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&role).Updates(map[string]any{"menus_initialized": true, "permissions_initialized": true}).Error
	})
}
