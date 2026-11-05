package middlewares

import (
	"errors"
	"strings"
	"sync"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/services"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

var (
	syncedCachedEnforcer *casbin.SyncedCachedEnforcer
	once                 sync.Once
	enforcerMu           sync.RWMutex
)

func createModel() (model.Model, error) {
	text := `
	[request_definition]
	r = roleGuid, url, method
	
	[policy_definition]
	p = roleGuid, url, method
	
	[role_definition]
	g = _, _
	
	[policy_effect]
	e = some(where (p.eft == allow))
	
	[matchers]
	m = r.roleGuid == p.roleGuid && keyMatch2(r.url, p.url) && (p.method == "*" || r.method == p.method)
	`
	return model.NewModelFromString(text)
}

// GetCasbin 获取casbin实例
func GetCasbin() *casbin.SyncedCachedEnforcer {
	once.Do(func() {
		if err := RefreshCasbin(); err != nil {
			global.NAV_LOG.Error("初始化Casbin失败", zap.Error(err))
			enforcer, fallbackErr := newEmptyCasbinEnforcer()
			if fallbackErr != nil {
				global.NAV_LOG.Error("创建空Casbin实例失败", zap.Error(fallbackErr))
				return
			}
			enforcerMu.Lock()
			syncedCachedEnforcer = enforcer
			enforcerMu.Unlock()
		}
	})
	enforcerMu.RLock()
	defer enforcerMu.RUnlock()
	return syncedCachedEnforcer
}

// RefreshCasbin 重新从业务权限表加载角色API策略。
func RefreshCasbin() error {
	enforcer, err := buildCasbinEnforcer()
	if err != nil {
		return err
	}
	enforcerMu.Lock()
	syncedCachedEnforcer = enforcer
	enforcerMu.Unlock()
	return nil
}

func buildCasbinEnforcer() (*casbin.SyncedCachedEnforcer, error) {
	if global.NAV_DB == nil {
		return nil, errors.New("database is not initialized")
	}
	m, err := createModel()
	if err != nil {
		return nil, err
	}
	enforcer, err := casbin.NewSyncedCachedEnforcer(m)
	if err != nil {
		return nil, err
	}
	enforcer.SetExpireTime(60 * 60)
	policies, err := services.PermissionServiceApp.RolePermissionPolicies()
	if err != nil {
		return nil, err
	}
	for _, policy := range policies {
		if policy.RoleGuid == "" || policy.Path == "" {
			continue
		}
		method := strings.ToUpper(strings.TrimSpace(policy.Method))
		if method == "" {
			method = "*"
		}
		if _, err := enforcer.AddPolicy(policy.RoleGuid, policy.Path, method); err != nil {
			return nil, err
		}
	}
	return enforcer, nil
}

func newEmptyCasbinEnforcer() (*casbin.SyncedCachedEnforcer, error) {
	m, err := createModel()
	if err != nil {
		return nil, err
	}
	enforcer, err := casbin.NewSyncedCachedEnforcer(m)
	if err != nil {
		return nil, err
	}
	enforcer.SetExpireTime(60 * 60)
	return enforcer, nil
}

// CasbinHandler 拦截器
func CasbinHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		urlPath := c.Request.URL.Path
		urlPath = strings.TrimPrefix(urlPath, global.NAV_CONFIG.System.RouterPrefix)
		method := strings.ToUpper(c.Request.Method)
		if publicEndpoint(method, urlPath) || selfEndpoint(method, urlPath) {
			c.Next()
			return
		}
		a := services.RequestAccess(c)
		if a == nil {
			response.NoAuth("未登录", c)
			c.Abort()
			return
		}
		if a.PlatformAdmin() {
			c.Next()
			return
		}
		// Uploaded assets live under an immutable company prefix. Static reads
		// still require a live authenticated user and never expose other tenants.
		oss := strings.TrimRight(global.NAV_CONFIG.Local.ProxyOssPath, "/")
		if oss != "" && strings.HasPrefix(urlPath, oss+"/") {
			relative := strings.TrimPrefix(urlPath, oss+"/")
			if strings.HasPrefix(relative, "companies/"+a.User.CompanyGuid+"/") && !strings.Contains(relative, "..") && !strings.Contains(relative, "\\") {
				c.Next()
				return
			}
			response.NoPermission("无权访问其他企业文件", c)
			c.Abort()
			return
		}
		if strings.HasPrefix(urlPath, "/file/") {
			if method == "GET" || a.CompanyAdmin() || a.Has("project.write") || a.Has("device.write") {
				c.Next()
				return
			}
			response.NoPermission("缺少上传权限", c)
			c.Abort()
			return
		}
		// System administration cannot be delegated through a forged role code or API wildcard.
		if strings.HasPrefix(urlPath, "/company") || strings.HasPrefix(urlPath, "/user") || strings.HasPrefix(urlPath, "/role") || strings.HasPrefix(urlPath, "/permission") {
			if !a.CompanyAdmin() {
				response.NoPermission("需要企业管理员权限", c)
				c.Abort()
				return
			}
			c.Next()
			return
		}
		if strings.HasPrefix(urlPath, "/config") || strings.HasPrefix(urlPath, "/secret") || strings.HasPrefix(urlPath, "/register") || strings.HasPrefix(urlPath, "/file") || strings.HasPrefix(urlPath, "/oss") || strings.HasPrefix(urlPath, "/os") || strings.HasPrefix(urlPath, "/logs") || strings.HasPrefix(urlPath, "/log") || strings.HasPrefix(urlPath, "/serial") {
			response.NoPermission("仅平台管理员可操作", c)
			c.Abort()
			return
		}
		waitUse, _ := utils.GetClaims(c)
		e := GetCasbin()
		if e == nil {
			response.NoPermission("权限系统未初始化", c)
			c.Abort()
			return
		}
		for _, roleGuid := range splitRoleGuids(waitUse.RoleGuids) {
			success, err := e.Enforce(roleGuid, urlPath, method)
			if err != nil {
				global.NAV_LOG.Error("Casbin鉴权失败", zap.Error(err))
				response.NoPermission("权限校验失败", c)
				c.Abort()
				return
			}
			if success {
				c.Next()
				return
			}
		}
		response.NoPermission("权限不足", c)
		c.Abort()
	}
}

func splitRoleGuids(value string) []string {
	return splitAccessValues(value)
}

func containsRoleCode(value string, target string) bool {
	for _, roleCode := range splitAccessValues(value) {
		if roleCode == target {
			return true
		}
	}
	return false
}

func splitAccessValues(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
