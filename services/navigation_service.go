package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	NavigationAuthenticated = "authenticated"
	NavigationPermission    = "permission"
	NavigationCompanyAdmin  = "company_admin"
	NavigationPlatformAdmin = "platform_admin"
)

type NavigationTarget struct {
	Key                  string   `json:"key"`
	Label                string   `json:"label"`
	Link                 string   `json:"link"`
	Icon                 string   `json:"icon"`
	Match                []string `json:"match,omitempty"`
	ACL                  []string `json:"acl,omitempty"`
	RequiresLicense      bool     `json:"requiresLicense,omitempty"`
	RequiredIdentity     string   `json:"requiredIdentity"`
	HideForPlatformAdmin bool     `json:"hideForPlatformAdmin,omitempty"`
	Protected            bool     `json:"protected,omitempty"`
	ActionCodes          []string `json:"actionCodes"`
	Features             []string `json:"features"`
}
type NavigationSection struct {
	Key      string             `json:"key"`
	Title    string             `json:"title"`
	Icon     string             `json:"icon"`
	Children []NavigationTarget `json:"children"`
}
type NavigationDefinition struct {
	Groups []NavigationSection `json:"groups"`
	Icons  []string            `json:"icons"`
}
type NavigationMenuView struct {
	domains.SysMenu
	Link              string   `json:"link"`
	ACL               []string `json:"acl"`
	RequiresLicense   bool     `json:"requiresLicense"`
	Protected         bool     `json:"protected"`
	Features          []string `json:"features"`
	ActionCodes       []string `json:"actionCodes"`
	RequiredIdentity  string   `json:"requiredIdentity"`
	Assignable        bool     `json:"assignable"`
	DefaultVisible    bool     `json:"defaultVisible"`
	RestrictionReason string   `json:"restrictionReason"`
}
type NavigationCatalog struct {
	Menus   []NavigationMenuView `json:"menus"`
	Targets []NavigationTarget   `json:"targets"`
	Icons   []string             `json:"icons"`
}
type NavigationService struct {
	db         *gorm.DB
	definition *NavigationDefinition
}

var navigationDefinition atomic.Pointer[NavigationDefinition]

// RegisterNavigation freezes the application's trusted catalog before serving requests.
func RegisterNavigation(definition NavigationDefinition) error {
	s, err := NewNavigationService(nil, definition)
	if err != nil {
		return err
	}
	navigationDefinition.Store(s.definition)
	return nil
}
func NewNavigationService(db *gorm.DB, definition NavigationDefinition) (*NavigationService, error) {
	data, err := json.Marshal(definition)
	if err != nil {
		return nil, err
	}
	var frozen NavigationDefinition
	if err := json.Unmarshal(data, &frozen); err != nil {
		return nil, err
	}
	icons := map[string]bool{}
	for _, icon := range frozen.Icons {
		icons[icon] = true
	}
	keys := map[string]bool{}
	for _, group := range frozen.Groups {
		if group.Key == "" || keys[group.Key] || group.Title == "" || !icons[group.Icon] {
			return nil, errors.New("invalid navigation group")
		}
		keys[group.Key] = true
		for _, item := range group.Children {
			if item.Key == "" || keys[item.Key] || item.Label == "" || !strings.HasPrefix(item.Link, "/") || strings.HasPrefix(item.Link, "//") || !icons[item.Icon] {
				return nil, errors.New("invalid navigation target")
			}
			keys[item.Key] = true
			switch item.RequiredIdentity {
			case NavigationPermission:
				if len(item.ACL) == 0 {
					return nil, errors.New("permission target requires access codes")
				}
			case NavigationAuthenticated, NavigationCompanyAdmin, NavigationPlatformAdmin:
				if len(item.ACL) != 0 || len(item.ActionCodes) != 0 {
					return nil, errors.New("identity target cannot grant business permissions")
				}
			default:
				return nil, errors.New("invalid navigation identity")
			}
		}
	}
	return &NavigationService{db: db, definition: &frozen}, nil
}
func NavigationServiceFor(db *gorm.DB) (*NavigationService, error) {
	definition := navigationDefinition.Load()
	if definition == nil {
		return nil, errors.New("navigation catalog is not registered")
	}
	if db == nil {
		db = global.NAV_DB
	}
	return &NavigationService{db: db, definition: definition}, nil
}
func (s *NavigationService) WithDB(db *gorm.DB) *NavigationService {
	return &NavigationService{db: db, definition: s.definition}
}
func (s *NavigationService) targets() map[string]NavigationTarget {
	targets := map[string]NavigationTarget{}
	for _, group := range s.definition.Groups {
		for _, item := range group.Children {
			targets[item.Key] = item
		}
	}
	return targets
}
func (s *NavigationService) nodes() ([]domains.SysMenu, error) {
	nodes := []domains.SysMenu{}
	err := s.db.Order("sort ASC, menu_key ASC").Find(&nodes).Error
	return nodes, err
}
func (s *NavigationService) protected(nodes []domains.SysMenu) map[string]bool {
	result := map[string]bool{}
	targets := s.targets()
	for _, node := range nodes {
		if targets[node.TargetKey].Protected {
			result[node.Key] = true
			result[node.ParentKey] = true
		}
	}
	return result
}
func (s *NavigationService) Initialize() error {
	if err := s.db.AutoMigrate(&domains.SysMenu{}, &domains.SysRoleMenu{}, &domains.SysRole{}); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i, group := range s.definition.Groups {
			node := domains.SysMenu{Key: group.Key, Type: "group", Name: group.Title, Icon: group.Icon, Sort: (i + 1) * 10, Visible: true, BuiltIn: true}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&node).Error; err != nil {
				return err
			}
			for j, item := range group.Children {
				node := domains.SysMenu{Key: item.Key, ParentKey: group.Key, Type: "menu", Name: item.Label, Icon: item.Icon, TargetKey: item.Key, Sort: (j + 1) * 10, Visible: true, BuiltIn: true}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&node).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
func (s *NavigationService) catalog() (NavigationCatalog, error) {
	nodes, err := s.nodes()
	catalog := NavigationCatalog{Menus: []NavigationMenuView{}, Targets: []NavigationTarget{}, Icons: append([]string{}, s.definition.Icons...)}
	if err != nil {
		return catalog, err
	}
	targets, protected := s.targets(), s.protected(nodes)
	for _, group := range s.definition.Groups {
		catalog.Targets = append(catalog.Targets, group.Children...)
	}
	for _, node := range nodes {
		t := targets[node.TargetKey]
		catalog.Menus = append(catalog.Menus, NavigationMenuView{SysMenu: node, Link: t.Link, ACL: t.ACL, Features: t.Features, ActionCodes: t.ActionCodes, RequiresLicense: t.RequiresLicense, Protected: protected[node.Key], RequiredIdentity: t.RequiredIdentity, DefaultVisible: t.RequiredIdentity == NavigationAuthenticated})
	}
	return catalog, nil
}
func (s *NavigationService) Catalog(actor *AccessContext) (NavigationCatalog, error) {
	if actor == nil || !actor.PlatformAdmin() {
		return NavigationCatalog{}, errors.New("仅平台管理员可管理菜单")
	}
	return s.catalog()
}
func roleTargetRestriction(role domains.SysRole, target NavigationTarget) string {
	switch target.RequiredIdentity {
	case NavigationPlatformAdmin:
		return "平台专用 · admin，普通角色不可授权"
	case NavigationCompanyAdmin:
		if !role.BuiltIn || role.Code != "COMPANY_ADMIN" {
			return "企业管理员专用"
		}
	case NavigationAuthenticated:
		return "默认向已登录用户开放，无需单独授权"
	case NavigationPermission:
	default:
		return "页面未注册"
	}
	return ""
}
func targetGranted(role domains.SysRole, permissions []domains.SysPermission, target NavigationTarget) bool {
	if roleTargetRestriction(role, target) != "" {
		return false
	}
	if target.RequiredIdentity == NavigationCompanyAdmin {
		return true
	}
	for _, code := range target.ACL {
		found := false
		for _, p := range permissions {
			if p.Enabled && p.Scope == "company" && p.Code == code {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// InitializeRoleMenus seeds built-in roles once. Explicit empty grants stay empty.
func (s *NavigationService) InitializeRoleMenus(_ string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		roles := []domains.SysRole{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("company_guid = '' AND code <> ? AND built_in = ? AND menus_initialized = ? AND permissions_initialized = ?", "SUPER_ADMIN", true, false, true).Find(&roles).Error; err != nil {
			return err
		}
		nodes, err := s.WithDB(tx).nodes()
		if err != nil {
			return err
		}
		targets := s.targets()
		for _, role := range roles {
			permissions, err := PermissionServiceApp.WithDB(tx).RolePermissions(role.Guid)
			if err != nil {
				return err
			}
			for _, node := range nodes {
				if node.Type == "menu" && targetGranted(role, permissions, targets[node.TargetKey]) {
					if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&domains.SysRoleMenu{RoleGuid: role.Guid, MenuKey: node.Key}).Error; err != nil {
						return err
					}
				}
			}
			if err := tx.Model(&role).UpdateColumn("menus_initialized", true).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func initializeRegisteredRoleMenus(db *gorm.DB, company string) error {
	if navigationDefinition.Load() == nil {
		return nil
	}
	s, err := NavigationServiceFor(db)
	if err != nil {
		return err
	}
	return s.InitializeRoleMenus(company)
}
func (s *NavigationService) Navigation(actor *AccessContext) ([]NavigationSection, error) {
	result := []NavigationSection{}
	if actor == nil {
		return result, errors.New("未登录")
	}
	nodes, err := s.nodes()
	if err != nil {
		return nil, err
	}
	targets := s.targets()
	selected := map[string]bool{}
	if !actor.PlatformAdmin() {
		ids := []string{}
		for _, r := range actor.Roles {
			if r.Enabled && r.CompanyGuid == "" && r.Code != "SUPER_ADMIN" {
				ids = append(ids, r.Guid)
			}
		}
		if len(ids) > 0 {
			rows := []domains.SysRoleMenu{}
			if err := s.db.Model(&domains.SysRoleMenu{}).Joins("JOIN sys_role r ON r.guid = sys_role_menu.role_guid").Where("role_guid IN ? AND r.company_guid = '' AND r.code <> ? AND r.enabled = ? AND r.deleted_time IS NULL", ids, "SUPER_ADMIN", true).Find(&rows).Error; err != nil {
				return nil, err
			}
			for _, row := range rows {
				selected[row.MenuKey] = true
			}
		}
	}
	for _, group := range nodes {
		if group.Type != "group" || !group.Visible {
			continue
		}
		section := NavigationSection{Key: group.Key, Title: group.Name, Icon: group.Icon, Children: []NavigationTarget{}}
		for _, node := range nodes {
			t, exists := targets[node.TargetKey]
			if !exists || node.Type != "menu" || node.ParentKey != group.Key || !node.Visible {
				continue
			}
			allowed := actor.PlatformAdmin() && !t.HideForPlatformAdmin
			if !actor.PlatformAdmin() {
				switch t.RequiredIdentity {
				case NavigationAuthenticated:
					allowed = true
				case NavigationCompanyAdmin:
					allowed = selected[node.Key] && actor.CompanyAdmin()
				case NavigationPermission:
					allowed = selected[node.Key]
					for _, code := range t.ACL {
						allowed = allowed && actor.Has(code)
					}
				}
			}
			if !allowed {
				continue
			}
			t.Key, t.Label, t.Icon = node.Key, node.Name, node.Icon
			section.Children = append(section.Children, t)
		}
		if len(section.Children) > 0 {
			result = append(result, section)
		}
	}
	return result, nil
}

func (s *NavigationService) SaveMenu(actor *AccessContext, node domains.SysMenu) (domains.SysMenu, error) {
	if actor == nil || !actor.PlatformAdmin() {
		return node, errors.New("仅平台管理员可管理菜单")
	}
	db := s.db
	node.Name, node.ParentKey, node.TargetKey, node.Icon = strings.TrimSpace(node.Name), strings.TrimSpace(node.ParentKey), strings.TrimSpace(node.TargetKey), strings.TrimSpace(node.Icon)
	if node.Name == "" || utf8.RuneCountInString(node.Name) > 64 {
		return node, errors.New("菜单名称不能为空且不能超过64字")
	}
	if node.Sort < 0 || node.Sort > 9999 {
		return node, errors.New("排序应为0至9999的整数")
	}
	allowedIcon := false
	for _, icon := range s.definition.Icons {
		if icon == node.Icon {
			allowedIcon = true
		}
	}
	if !allowedIcon {
		return node, errors.New("请选择系统支持的图标")
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if _, err := s.WithDB(tx.Clauses(clause.Locking{Strength: "UPDATE"})).nodes(); err != nil {
			return err
		}
		if node.Key == "" {
			node.Key = "custom-" + uuid.NewString()
			node.BuiltIn = false
		} else {
			var existing domains.SysMenu
			if err := tx.Where("menu_key = ?", node.Key).First(&existing).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("菜单不存在，请刷新后重试")
				}
				return err
			}
			if node.Type != existing.Type {
				return errors.New("已保存菜单不能修改类型")
			}
			node.BuiltIn = existing.BuiltIn
			if existing.BuiltIn && node.TargetKey != existing.TargetKey {
				return errors.New("内置菜单不能更换页面")
			}
		}
		switch node.Type {
		case "group":
			if node.ParentKey != "" || node.TargetKey != "" {
				return errors.New("分组只能位于顶层，不能设置页面")
			}
		case "menu":
			target, exists := s.targets()[node.TargetKey]
			if !exists {
				return errors.New("请选择已有系统页面")
			}
			var copies int64
			if err := tx.Model(&domains.SysMenu{}).Where("target_key = ? AND menu_key <> ?", node.TargetKey, node.Key).Count(&copies).Error; err != nil {
				return err
			}
			if target.Protected && copies > 0 {
				return errors.New("菜单管理入口不能重复创建")
			}
			var parent domains.SysMenu
			if err := tx.Where("menu_key = ? AND type = ?", node.ParentKey, "group").First(&parent).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("请选择有效的所属分组")
				}
				return err
			}
			if target.Protected && (!node.Visible || !parent.Visible) {
				return errors.New("菜单管理入口及其分组必须保持显示")
			}
		default:
			return errors.New("菜单类型无效")
		}
		if node.Type == "group" && !node.Visible {
			nodes, err := s.WithDB(tx).nodes()
			if err != nil {
				return err
			}
			if s.protected(nodes)[node.Key] {
				return errors.New("受保护菜单所在分组必须保持显示")
			}
		}

		return tx.Save(&node).Error
	})
	return node, err
}
func (s *NavigationService) DeleteMenu(actor *AccessContext, key string) error {
	if actor == nil || !actor.PlatformAdmin() {
		return errors.New("仅平台管理员可管理菜单")
	}
	db := s.db
	return db.Transaction(func(tx *gorm.DB) error {
		if _, err := s.WithDB(tx.Clauses(clause.Locking{Strength: "UPDATE"})).nodes(); err != nil {
			return err
		}
		var node domains.SysMenu
		if err := tx.Where("menu_key = ?", key).First(&node).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("菜单不存在，请刷新后重试")
			}
			return err
		}
		if node.BuiltIn || s.targets()[node.TargetKey].Protected {
			return errors.New("内置菜单不能删除，可通过隐藏控制展示")
		}
		var count int64
		if err := tx.Model(&domains.SysMenu{}).Where("parent_key = ?", key).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("分组下还有%d个菜单，请先移动或删除子菜单", count)
		}
		if err := tx.Where("menu_key = ?", key).Delete(&domains.SysRoleMenu{}).Error; err != nil {
			return err
		}
		return tx.Delete(&node).Error
	})
}
