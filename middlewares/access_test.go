package middlewares_test

import (
	"encoding/json"
	"fmt"
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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
