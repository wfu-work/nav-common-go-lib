package services

import (
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func navigationFixture(t *testing.T) (*gorm.DB, *NavigationService, *AccessContext, *AccessContext) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "navigation.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&domains.SysCompany{}, &domains.SysRole{}, &domains.SysPermission{}, &domains.SysRolePermission{}, &domains.SysUserRole{}); err != nil {
		t.Fatal(err)
	}
	for _, company := range []string{"a", "b"} {
		if err := db.Create(&domains.SysCompany{BaseDataEntity: domains.BaseDataEntity{Guid: company}, Name: company, Code: company, Enabled: true, Status: 1}).Error; err != nil {
			t.Fatal(err)
		}
	}
	definition := NavigationDefinition{Icons: []string{"group", "page"}, Groups: []NavigationSection{{Key: "work", Title: "Work", Icon: "group", Children: []NavigationTarget{
		{Key: "overview", Label: "Overview", Link: "/overview", Icon: "page", RequiredIdentity: NavigationPermission, ACL: []string{"summary.read"}},
		{Key: "projects", Label: "Projects", Link: "/project", Icon: "page", RequiredIdentity: NavigationPermission, ACL: []string{"project.read"}, ActionCodes: []string{"project.write"}},
		{Key: "roles", Label: "Roles", Link: "/roles", Icon: "page", RequiredIdentity: NavigationCompanyAdmin},
		{Key: "menus", Label: "Menus", Link: "/menus", Icon: "page", RequiredIdentity: NavigationPlatformAdmin, Protected: true},
		{Key: "about", Label: "About", Link: "/about", Icon: "page", RequiredIdentity: NavigationAuthenticated},
	}}}}
	s, err := NewNavigationService(db, definition)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Initialize(); err != nil {
		t.Fatal(err)
	}
	permissions := []domains.SysPermission{
		{BaseDataEntity: domains.BaseDataEntity{Guid: "read"}, Code: "project.read", Name: "Read", Type: domains.PermissionTypeMenu, Scope: "company", Enabled: true},
		{BaseDataEntity: domains.BaseDataEntity{Guid: "summary"}, Code: "summary.read", Name: "Summary", Type: domains.PermissionTypeMenu, Scope: "company", Enabled: true, Dependencies: []string{"project.read"}},
		{BaseDataEntity: domains.BaseDataEntity{Guid: "write"}, Code: "project.write", Name: "Write", Type: domains.PermissionTypeButton, ParentGuid: "read", Scope: "company", Enabled: true, Dependencies: []string{"project.read"}},
		{BaseDataEntity: domains.BaseDataEntity{Guid: "api"}, Code: "api.POST.project", Name: "API", Type: domains.PermissionTypeAPI, ParentGuid: "write", Scope: "company", Enabled: true, Path: "/project", Method: "POST"},
		{BaseDataEntity: domains.BaseDataEntity{Guid: "host"}, Code: "host.write", Name: "Host", Type: domains.PermissionTypeButton, Scope: "platform", Enabled: true},
	}
	if err := PermissionServiceApp.WithDB(db).SyncCatalog(permissions); err != nil {
		t.Fatal(err)
	}
	admin := &AccessContext{User: domains.SysUser{BaseDataEntity: domains.BaseDataEntity{Guid: domains.BuiltinAdminGUID}, Username: "admin"}}
	owner := &AccessContext{User: domains.SysUser{CompanyGuid: "a"}, Roles: []domains.SysRole{{CompanyGuid: "a", Code: "COMPANY_ADMIN", BuiltIn: true, Enabled: true}}}
	return db, s, admin, owner
}
func makeManagedRole(t *testing.T, db *gorm.DB, actor *AccessContext, company, code string) domains.SysRole {
	t.Helper()
	role, err := RoleServiceApp.WithDB(db).SaveManagedRole(actor, ManagedRoleRequest{CompanyGuid: company, Name: code, Code: code})
	if err != nil {
		t.Fatal(err)
	}
	return role
}
func grantCodes(t *testing.T, db *gorm.DB, guid string) []string {
	t.Helper()
	permissions, err := PermissionServiceApp.WithDB(db).RolePermissions(guid)
	if err != nil {
		t.Fatal(err)
	}
	codes := []string{}
	for _, p := range permissions {
		codes = append(codes, p.Code)
	}
	sort.Strings(codes)
	return codes
}
func TestManagedRoleBoundaries(t *testing.T) {
	db, _, admin, owner := navigationFixture(t)
	service := RoleServiceApp.WithDB(db)
	role := makeManagedRole(t, db, owner, "", " custom ")
	if role.Code != "CUSTOM" || !role.MenusInitialized {
		t.Fatal(role)
	}
	if _, err := service.SaveManagedRole(owner, ManagedRoleRequest{Name: "duplicate", Code: "custom"}); err == nil {
		t.Fatal("duplicate accepted")
	}
	other := makeManagedRole(t, db, admin, "b", "CUSTOM")
	for _, id := range []string{other.Guid, "missing"} {
		if _, err := service.GetManagedRole(owner, id); err == nil {
			t.Fatal("cross-company role exposed")
		}
	}
	if _, err := service.SaveManagedRole(owner, ManagedRoleRequest{Guid: role.Guid, CompanyGuid: "b", Name: "move", Code: "CUSTOM"}); err == nil {
		t.Fatal("company reassignment accepted")
	}
	for _, code := range []string{"SUPER_ADMIN", "COMPANY_ADMIN"} {
		if _, err := service.SaveManagedRole(owner, ManagedRoleRequest{Name: code, Code: code}); err == nil {
			t.Fatal("reserved code accepted")
		}
	}
	if _, err := service.SaveManagedRole(&AccessContext{User: domains.SysUser{CompanyGuid: "a"}}, ManagedRoleRequest{Name: "x", Code: "x"}); err == nil {
		t.Fatal("ordinary user created role")
	}
}
func TestRoleAuthorizationValidationAndAtomicity(t *testing.T) {
	db, s, _, owner := navigationFixture(t)
	role := makeManagedRole(t, db, owner, "", "CUSTOM")
	valid := RoleAuthorizationRequest{MenuKeys: []string{"projects", "projects"}, PermissionGuids: []string{"write"}}
	if err := s.SaveRoleAuthorization(owner, role.Guid, valid); err != nil {
		t.Fatal(err)
	}
	expected := []string{"api.POST.project", "project.read", "project.write"}
	if got := grantCodes(t, db, role.Guid); !reflect.DeepEqual(got, expected) {
		t.Fatal(got)
	}
	for _, bad := range []RoleAuthorizationRequest{{MenuKeys: []string{"menus"}}, {MenuKeys: []string{"roles"}}, {MenuKeys: []string{"missing"}}, {MenuKeys: []string{"work"}}, {MenuKeys: []string{"projects"}, PermissionGuids: []string{"host"}}, {PermissionGuids: []string{"write"}}, {MenuKeys: []string{"overview"}, PermissionGuids: []string{"write"}}, {MenuKeys: []string{"projects"}, PermissionGuids: []string{"api"}}} {
		if err := s.SaveRoleAuthorization(owner, role.Guid, bad); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
		if got := grantCodes(t, db, role.Guid); !reflect.DeepEqual(got, expected) {
			t.Fatal("partial grants", got)
		}
	}
	db.Callback().Create().Before("gorm:create").Register("test:fail-menu", func(tx *gorm.DB) {
		if tx.Statement.Table == "sys_role_menu" {
			tx.AddError(errors.New("injected menu write failure"))
		}
	})
	if err := s.SaveRoleAuthorization(owner, role.Guid, RoleAuthorizationRequest{MenuKeys: []string{"overview"}}); err == nil {
		t.Fatal("write should fail")
	}
	db.Callback().Create().Remove("test:fail-menu")
	if got := grantCodes(t, db, role.Guid); !reflect.DeepEqual(got, expected) {
		t.Fatal("rollback failed", got)
	}
	data, err := s.RoleAuthorization(owner, role.Guid)
	if err != nil || !reflect.DeepEqual(data.MenuKeys, []string{"projects"}) {
		t.Fatal(data, err)
	}
	renamed, err := RoleServiceApp.WithDB(db).SaveManagedRole(owner, ManagedRoleRequest{Guid: role.Guid, Name: "Renamed", Code: role.Code})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveRoleAuthorization(owner, role.Guid, RoleAuthorizationRequest{}); err != nil {
		t.Fatal(err)
	}
	data, _ = s.RoleAuthorization(owner, role.Guid)
	if data.Role.Name != renamed.Name || len(data.MenuKeys) != 0 || len(data.PermissionGuids) != 0 {
		t.Fatal(data)
	}
}
func TestNavigationUsesExplicitMenusAndPermissionUnion(t *testing.T) {
	db, s, admin, owner := navigationFixture(t)
	one := makeManagedRole(t, db, owner, "", "ONE")
	two := makeManagedRole(t, db, owner, "", "TWO")
	if err := s.SaveRoleAuthorization(owner, one.Guid, RoleAuthorizationRequest{MenuKeys: []string{"overview"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveRoleAuthorization(owner, two.Guid, RoleAuthorizationRequest{MenuKeys: []string{"projects"}}); err != nil {
		t.Fatal(err)
	}
	actor := &AccessContext{User: domains.SysUser{CompanyGuid: "a"}, Roles: []domains.SysRole{one}}
	actor.Permissions, _ = PermissionServiceApp.WithDB(db).RolePermissions(one.Guid)
	keys := func(actor *AccessContext) []string {
		sections, err := s.Navigation(actor)
		if err != nil {
			t.Fatal(err)
		}
		keys := []string{}
		for _, section := range sections {
			for _, item := range section.Children {
				keys = append(keys, item.Key)
			}
		}
		sort.Strings(keys)
		return keys
	}
	if got := keys(actor); !reflect.DeepEqual(got, []string{"about", "overview"}) {
		t.Fatal("dependency leaked menu", got)
	}
	actor.Roles = append(actor.Roles, two)
	if got := keys(actor); !reflect.DeepEqual(got, []string{"about", "overview", "projects"}) {
		t.Fatal(got)
	}
	if err := db.Model(&two).UpdateColumn("enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	if got := keys(actor); !reflect.DeepEqual(got, []string{"about", "overview"}) {
		t.Fatal("disabled role leaked menu", got)
	}
	actor.Roles = nil
	if got := keys(actor); !reflect.DeepEqual(got, []string{"about"}) {
		t.Fatal(got)
	}
	if got := keys(admin); len(got) != 5 {
		t.Fatal(got)
	}
	if (&AccessContext{User: domains.SysUser{Username: "admin"}}).PlatformAdmin() {
		t.Fatal("forged admin")
	}
}
func TestBuiltinMenuInitializationDoesNotResetEmpty(t *testing.T) {
	db, s, _, owner := navigationFixture(t)
	previous := navigationDefinition.Swap(s.definition)
	t.Cleanup(func() { navigationDefinition.Store(previous) })
	if err := InitializeCompanyRoles(db, "a"); err != nil {
		t.Fatal(err)
	}
	if err := s.InitializeRoleMenus("a"); err != nil {
		t.Fatal(err)
	}
	var role domains.SysRole
	db.Where("company_guid = ? AND code = ?", "a", "COMPANY_ADMIN").First(&role)
	data, err := s.RoleAuthorization(owner, role.Guid)
	if err != nil || len(data.MenuKeys) != 3 {
		t.Fatal(data.MenuKeys, err)
	}
	if err := s.SaveRoleAuthorization(owner, role.Guid, RoleAuthorizationRequest{}); err != nil {
		t.Fatal(err)
	}
	if err := InitializeCompanyRoles(db, "a"); err != nil {
		t.Fatal(err)
	}
	if err := s.InitializeRoleMenus("a"); err != nil {
		t.Fatal(err)
	}
	data, _ = s.RoleAuthorization(owner, role.Guid)
	if len(data.MenuKeys) != 0 || len(data.PermissionGuids) != 0 {
		t.Fatal("empty grants reset")
	}
	for _, node := range data.Menus {
		if node.Key == "roles" && !node.Assignable {
			t.Fatal("admin identity menu unavailable")
		}
		if node.Key == "menus" && (node.Assignable || node.RestrictionReason == "") {
			t.Fatal("platform menu assignable")
		}
	}
	if err := RoleServiceApp.WithDB(db).DeleteManagedRole(owner, role.Guid); err == nil {
		t.Fatal("deleted built-in")
	}
}
func TestMenuProtectionAndDeletionCleanup(t *testing.T) {
	db, s, admin, owner := navigationFixture(t)
	catalog, _ := s.Catalog(admin)
	nodes := map[string]domains.SysMenu{}
	for _, view := range catalog.Menus {
		nodes[view.Key] = view.SysMenu
	}
	group := nodes["work"]
	group.Visible = false
	if _, err := s.SaveMenu(admin, group); err == nil {
		t.Fatal("hidden protected group")
	}
	menu := nodes["menus"]
	menu.Visible = false
	if _, err := s.SaveMenu(admin, menu); err == nil {
		t.Fatal("hidden protected menu")
	}
	menu = nodes["menus"]
	menu.Key = ""
	if _, err := s.SaveMenu(admin, menu); err == nil {
		t.Fatal("duplicated protected target")
	}
	if _, err := s.Catalog(owner); err == nil {
		t.Fatal("tenant read admin catalog")
	}
	if _, err := s.SaveMenu(owner, nodes["projects"]); err == nil {
		t.Fatal("tenant wrote menu")
	}
	if err := s.DeleteMenu(admin, "menus"); err == nil {
		t.Fatal("deleted builtin menu")
	}
	customGroup, err := s.SaveMenu(admin, domains.SysMenu{Type: "group", Name: "Custom", Icon: "group", Visible: true})
	if err != nil {
		t.Fatal(err)
	}
	custom, err := s.SaveMenu(admin, domains.SysMenu{Type: "menu", Name: "Custom project", Icon: "page", TargetKey: "projects", ParentKey: customGroup.Key, Visible: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteMenu(admin, customGroup.Key); err == nil {
		t.Fatal("deleted populated group")
	}
	role := makeManagedRole(t, db, owner, "", "CLEANUP")
	if err := s.SaveRoleAuthorization(owner, role.Guid, RoleAuthorizationRequest{MenuKeys: []string{custom.Key}}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteMenu(admin, custom.Key); err != nil {
		t.Fatal(err)
	}
	data, _ := s.RoleAuthorization(owner, role.Guid)
	if len(data.MenuKeys) != 0 {
		t.Fatal("orphan menu grants")
	}
	if err := s.SaveRoleAuthorization(owner, role.Guid, RoleAuthorizationRequest{MenuKeys: []string{"projects"}}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domains.SysUserRole{RoleGuid: role.Guid, UserGuid: "user"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := RoleServiceApp.WithDB(db).DeleteManagedRole(owner, role.Guid); err == nil {
		t.Fatal("deleted role with users")
	}
	db.Where("role_guid = ?", role.Guid).Delete(&domains.SysUserRole{})
	if err := RoleServiceApp.WithDB(db).DeleteManagedRole(owner, role.Guid); err != nil {
		t.Fatal(err)
	}
	for _, model := range []any{&domains.SysRoleMenu{}, &domains.SysRolePermission{}} {
		var n int64
		db.Model(model).Where("role_guid = ?", role.Guid).Count(&n)
		if n != 0 {
			t.Fatal("orphan grants")
		}
	}
}
func TestNavigationDefinitionRejectsInvalidIdentity(t *testing.T) {
	for _, target := range []NavigationTarget{{Key: "bad", Label: "bad", Link: "//evil", Icon: "page", RequiredIdentity: NavigationAuthenticated}, {Key: "bad", Label: "bad", Link: "/bad", Icon: "page", RequiredIdentity: NavigationPermission}, {Key: "bad", Label: "bad", Link: "/bad", Icon: "page", RequiredIdentity: NavigationCompanyAdmin, ActionCodes: []string{"write"}}} {
		if _, err := NewNavigationService(nil, NavigationDefinition{Icons: []string{"page"}, Groups: []NavigationSection{{Key: "g", Title: "g", Icon: "page", Children: []NavigationTarget{target}}}}); err == nil {
			t.Fatal("accepted invalid definition")
		}
	}
}
