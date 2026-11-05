package middlewares_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/wfu-work/nav-common-go-lib/apis"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
	"github.com/wfu-work/nav-common-go-lib/services"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func fixture(t *testing.T) (*gorm.DB, *gin.Engine, map[string]string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	global.NAV_DB = db
	global.NAV_LOG = zap.NewNop()
	global.LOCAL_CACHE = cache.New(cache.NoExpiration, 0)
	global.NAV_CONFIG.System.RouterPrefix = "/api"
	global.NAV_CONFIG.JWT.SigningKey = "test-signing-key"
	global.NAV_CONFIG.JWT.ExpiresTime = "24h"
	global.NAV_CONFIG.JWT.BufferTime = "1h"
	if err := db.AutoMigrate(&domains.SysCompany{}, &domains.SysUser{}, &domains.SysRole{}, &domains.SysUserRole{}, &domains.SysPermission{}, &domains.SysRolePermission{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		if err := db.Create(&domains.SysCompany{BaseDataEntity: domains.BaseDataEntity{Guid: id}, Name: id, Code: id, Enabled: true, Status: 1}).Error; err != nil {
			t.Fatal(err)
		}
	}
	users := []domains.SysUser{{BaseDataEntity: domains.BaseDataEntity{Guid: domains.BuiltinAdminGUID}, Username: "admin", Enable: 1, AuthVersion: 1}, {BaseDataEntity: domains.BaseDataEntity{Guid: "ua"}, Username: "alice", CompanyGuid: "a", Enable: 1, AuthVersion: 1}, {BaseDataEntity: domains.BaseDataEntity{Guid: "ub"}, Username: "bob", CompanyGuid: "b", Enable: 1, AuthVersion: 1}}
	tokens := map[string]string{}
	for _, u := range users {
		if err := db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
		token, _, err := utils.LoginToken(&u)
		if err != nil {
			t.Fatal(err)
		}
		tokens[u.Username] = token
	}
	r := gin.New()
	g := r.Group("/api").Use(middlewares.JWTAuth(), middlewares.CasbinHandler())
	user := apis.UserApi{}
	company := apis.CompanyApi{}
	g.GET("/user/token", user.GetUserByToken)
	g.GET("/user/:guid", user.GetByGuid)
	g.POST("/user", user.Save)
	g.DELETE("/user/:guid", user.DeleteByGuid)
	g.POST("/configs", func(c *gin.Context) { c.Status(204) })
	g.GET("/business", func(c *gin.Context) { c.Status(204) })
	g.POST("/company", company.Save)
	return db, r, tokens
}
func request(r *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	r.ServeHTTP(w, req)
	return w
}
func TestAdminIdentityAndSessionRevocation(t *testing.T) {
	db, r, tokens := fixture(t)
	if got := request(r, "POST", "/api/configs", "", "").Code; got != http.StatusUnauthorized {
		t.Fatalf("anonymous config write: %d", got)
	}
	if got := request(r, "GET", "/api/business", tokens["admin"], "").Code; got != 204 {
		t.Fatalf("admin without any role: %d", got)
	}
	role := domains.SysRole{BaseDataEntity: domains.BaseDataEntity{Guid: "forged"}, CompanyGuid: "a", Name: "fake", Code: "SUPER_ADMIN", Enabled: true}
	db.Create(&role)
	db.Create(&domains.SysUserRole{UserGuid: "ua", RoleGuid: role.Guid})
	if got := request(r, "POST", "/api/configs", tokens["alice"], "").Code; got != 403 {
		t.Fatalf("role code escalated privileges: %d", got)
	}
	db.Model(&domains.SysUser{}).Where("guid = ?", "ua").UpdateColumn("auth_version", 2)
	if got := request(r, "GET", "/api/user/token", tokens["alice"], "").Code; got != 401 {
		t.Fatalf("old session survived revocation: %d", got)
	}
	db.Model(&domains.SysCompany{}).Where("guid = ?", "b").UpdateColumn("enabled", false)
	if got := request(r, "GET", "/api/user/token", tokens["bob"], "").Code; got != 401 {
		t.Fatalf("disabled company session: %d", got)
	}
}
func TestTenantIdentityManagement(t *testing.T) {
	db, r, tokens := fixture(t)
	role := domains.SysRole{BaseDataEntity: domains.BaseDataEntity{Guid: "manager"}, CompanyGuid: "a", Name: "manager", Code: "COMPANY_ADMIN", Enabled: true, BuiltIn: true}
	db.Create(&role)
	db.Create(&domains.SysUserRole{UserGuid: "ua", RoleGuid: role.Guid})
	w := request(r, "GET", "/api/user/ub", tokens["alice"], "")
	if strings.Contains(w.Body.String(), "bob") {
		t.Fatal("cross-company user leaked")
	}
	w = request(r, "DELETE", "/api/user/ua", tokens["alice"], "")
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != float64(400) {
		t.Fatalf("deleted last company admin: %s", w.Body.String())
	}
	var u domains.SysUser
	if err := db.Where("guid = ?", "ua").First(&u).Error; err != nil {
		t.Fatal("failed delete did not roll back")
	}
	w = request(r, "DELETE", "/api/user/"+domains.BuiltinAdminGUID, tokens["admin"], "")
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != float64(400) {
		t.Fatal("deleted builtin admin")
	}
}

func TestCompanyDisabledStateIsPreserved(t *testing.T) {
	db, r, tokens := fixture(t)
	password, err := utils.AesEncrypt("Tenant12345")
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"name": "Disabled company", "code": "DISABLED", "enabled": false, "adminUsername": "disabled_owner", "adminPassword": password})
	w := request(r, "POST", "/api/company", tokens["admin"], string(payload))
	var result struct {
		Code int
		Data string
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Code != 200 {
		t.Fatal("create disabled company", w.Body.String(), err)
	}
	var owner domains.SysUser
	if err := db.Where("company_guid = ?", result.Data).First(&owner).Error; err != nil {
		t.Fatal(err)
	}
	for i, enabled := range []bool{false, true, false} {
		if i > 0 {
			payload, _ = json.Marshal(map[string]any{"guid": result.Data, "name": "Disabled company", "code": "DISABLED", "status": 1, "enabled": enabled})
			w = request(r, "POST", "/api/company", tokens["admin"], string(payload))
			var updated struct{ Code int }
			if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil || updated.Code != 200 {
				t.Fatal("update company state", w.Body.String(), err)
			}
		}
		var company domains.SysCompany
		if err := db.Where("guid = ?", result.Data).First(&company).Error; err != nil || company.Enabled != enabled {
			t.Fatalf("company state: %+v %v, wanted %v", company, err, enabled)
		}
		if _, err := services.LoadAccess(db, owner.Guid); (err == nil) != enabled {
			t.Fatalf("company enabled=%v access error=%v", enabled, err)
		}
	}
}

func TestIdentityMigrationHandlesLegacyNullCompanyAndIsIdempotent(t *testing.T) {
	db, _, _ := fixture(t)
	permission := domains.SysPermission{BaseDataEntity: domains.BaseDataEntity{Guid: "legacy-read"}, Code: "project.read", Name: "read", Type: "menu", Scope: "company", Enabled: true}
	if err := db.Create(&permission).Error; err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct{ user, code string }{{"ua", "USER"}, {"ub", "SUPER_ADMIN"}} {
		role := domains.SysRole{BaseDataEntity: domains.BaseDataEntity{Guid: "legacy-" + entry.code}, Name: entry.code, Code: entry.code, Enabled: true}
		if err := db.Create(&role).Error; err != nil {
			t.Fatal(err)
		}
		// ALTER TABLE adds nullable company_guid to roles in pre-company schemas.
		if err := db.Model(&role).UpdateColumn("company_guid", nil).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&domains.SysRolePermission{RoleGuid: role.Guid, PermissionGuid: permission.Guid}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&domains.SysUserRole{UserGuid: entry.user, RoleGuid: role.Guid}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&domains.SysUser{}).Where("guid = ?", entry.user).UpdateColumn("company_guid", "").Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := services.InitializeIdentity(db); err != nil {
			t.Fatal(err)
		}
	}
	for _, entry := range []struct{ user, code string }{{"ua", "USER"}, {"ub", "VIEWER"}} {
		access, err := services.LoadAccess(db, entry.user)
		if err != nil || access.PlatformAdmin() || access.User.CompanyGuid != domains.DefaultCompanyGUID || len(access.Roles) != 1 || access.Roles[0].Code != entry.code {
			t.Fatalf("legacy identity %s: %+v %v", entry.user, access, err)
		}
		if access.Has("project.read") != (entry.code == "USER") {
			t.Fatalf("legacy grants were lost or escalated: %+v", access)
		}
	}
	admin, err := services.LoadAccess(db, domains.BuiltinAdminGUID)
	if err != nil || !admin.PlatformAdmin() {
		t.Fatalf("builtin identity changed: %+v %v", admin, err)
	}
}
func TestPermissionsDependenciesAndScope(t *testing.T) {
	db, _, _ := fixture(t)
	role := domains.SysRole{BaseDataEntity: domains.BaseDataEntity{Guid: "r"}, CompanyGuid: "a", Code: "OPERATOR", Name: "operator", Enabled: true}
	db.Create(&role)
	for _, p := range []domains.SysPermission{{BaseDataEntity: domains.BaseDataEntity{Guid: "read"}, Code: "project.read", Type: "menu", Scope: "company", Enabled: true}, {BaseDataEntity: domains.BaseDataEntity{Guid: "write"}, Code: "project.write", Type: "button", Scope: "company", Dependencies: []string{"project.read"}, Enabled: true}, {BaseDataEntity: domains.BaseDataEntity{Guid: "api"}, Code: "api.project.read", ParentGuid: "read", Type: "api", Scope: "company", Path: "/project/list", Method: "GET", Enabled: true}, {BaseDataEntity: domains.BaseDataEntity{Guid: "platform"}, Code: "platform.all", Scope: "platform", Type: "api", Path: "/*", Method: "*", Enabled: true}} {
		if err := db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := services.PermissionServiceApp.WithDB(db)
	if err := svc.SetRolePermissions("r", []string{"write"}); err != nil {
		t.Fatal(err)
	}
	grants, err := svc.RolePermissions("r")
	if err != nil || len(grants) != 3 {
		t.Fatalf("dependency expansion: %v %v", grants, err)
	}
	if err := svc.SetRolePermissions("r", []string{"platform"}); err == nil {
		t.Fatal("company role granted platform wildcard")
	}
	grants, _ = svc.RolePermissions("r")
	if len(grants) != 3 {
		t.Fatal("failed authorization changed grants")
	}
}

func TestCompanyAssetsCannotCrossTenantBoundaries(t *testing.T) {
	db, r, tokens := fixture(t)
	global.NAV_CONFIG.Local.OssPath = t.TempDir()
	global.NAV_CONFIG.Local.ProxyOssPath = "/oss"
	file := apis.FileApi{}
	group := r.Group("/api").Use(middlewares.JWTAuth(), middlewares.CasbinHandler())
	group.POST("/file/upload/base64", file.UploadBase64)
	group.GET("/file/download", file.Download)
	group.GET("/oss/*filepath", func(c *gin.Context) { c.Status(204) })
	role := domains.SysRole{BaseDataEntity: domains.BaseDataEntity{Guid: "uploader"}, CompanyGuid: "a", Code: "UPLOADER", Name: "upload", Enabled: true}
	db.Create(&role)
	permission := domains.SysPermission{BaseDataEntity: domains.BaseDataEntity{Guid: "upload"}, Code: "project.write", Type: "button", Name: "upload", Scope: "company", Enabled: true}
	db.Create(&permission)
	db.Create(&domains.SysRolePermission{RoleGuid: role.Guid, PermissionGuid: permission.Guid})
	db.Create(&domains.SysUserRole{UserGuid: "ua", RoleGuid: role.Guid})
	uploaded := request(r, "POST", "/api/file/upload/base64", tokens["alice"], `{"data":"aGVsbG8=","fileName":"test.png"}`)
	var result struct {
		Code int
		Data string
	}
	if err := json.Unmarshal(uploaded.Body.Bytes(), &result); err != nil || result.Code != 200 {
		t.Fatal("upload", uploaded.Body.String(), err)
	}
	if !strings.HasPrefix(result.Data, "/api/oss/companies/a/") {
		t.Fatal("missing ownership prefix", result.Data)
	}
	if got := request(r, "GET", result.Data, tokens["bob"], "").Code; got != 403 {
		t.Fatal("foreign static asset", got)
	}
	if got := request(r, "GET", result.Data, tokens["alice"], "").Code; got != 204 {
		t.Fatal("own static asset", got)
	}
	if got := request(r, "GET", result.Data, "", "").Code; got != 401 {
		t.Fatal("anonymous static asset", got)
	}
	own := request(r, "GET", "/api/file/download?path="+result.Data, tokens["alice"], "")
	if own.Body.String() != "hello" {
		t.Fatal("own download", own.Body.String())
	}
	foreign := request(r, "GET", "/api/file/download?path="+result.Data, tokens["bob"], "")
	if strings.Contains(foreign.Body.String(), "hello") {
		t.Fatal("foreign download")
	}
	if got := request(r, "GET", result.Data, tokens["admin"], "").Code; got != 204 {
		t.Fatal("platform asset access", got)
	}
}
