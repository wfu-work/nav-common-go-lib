package middlewares

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/utils"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	syncedCachedEnforcer *casbin.SyncedCachedEnforcer
	once                 sync.Once
)

func createModel() (model.Model, error) {
	text := `
	[request_definition]
	r = roleGuid, url, method
	
	[policy_definition]
	p = roleGuid, url, v2
	
	[role_definition]
	g = _, _
	
	[policy_effect]
	e = some(where (p.eft == allow))
	
	[matchers]
	m = r.roleGuid == p.roleGuid && keyMatch2(r.url, p.url)
	`
	return model.NewModelFromString(text)
}

// GetCasbin 获取casbin实例
func GetCasbin() *casbin.SyncedCachedEnforcer {
	once.Do(func() {
		a, err := gormadapter.NewAdapterByDBWithCustomTable(global.NAV_DB, &domains.SysRole{}, "sys_role")
		if err != nil {
			zap.L().Error("适配数据库失败请检查casbin表是否为InnoDB引擎!", zap.Error(err))
			return
		}
		m, err := createModel()
		if err != nil {
			zap.L().Error("字符串加载模型失败!", zap.Error(err))
			return
		}
		syncedCachedEnforcer, err = casbin.NewSyncedCachedEnforcer(m, a)
		if err != nil {
			_ = fmt.Errorf("failed to initialize enforcer: %v", err)
			return
		}
		syncedCachedEnforcer.SetExpireTime(60 * 60)
		err = syncedCachedEnforcer.LoadPolicy()
		if err != nil {
			_ = fmt.Errorf("failed to load policy: %v", err)
		}
	})
	return syncedCachedEnforcer
}

// CasbinHandler 拦截器
func CasbinHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		urlPath := c.Request.URL.Path
		urlPath = strings.TrimPrefix(urlPath, global.NAV_CONFIG.System.RouterPrefix)
		method := c.Request.Method
		for _, white := range jwtWhiteList {
			if ok, _ := path.Match(white, urlPath); ok {
				c.Next()
				return
			}
		}

		waitUse, _ := utils.GetClaims(c)
		sub := waitUse.RoleGuids
		roleCode := waitUse.RoleCodes
		if roleCode != "" && strings.Contains(roleCode, "SUPER_ADMIN") {
			c.Next()
			return
		}
		e := GetCasbin()
		success, _ := e.Enforce(sub, urlPath, method)
		if !success {
			//response.FailWithDetailed(gin.H{}, "权限不足", c)
			//c.Abort()
			//return
		}
		c.Next()
	}
}
