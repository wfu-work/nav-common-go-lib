package middlewares

import (
	"errors"
	"strconv"
	"time"

	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/services"
	"github.com/wfu-work/nav-common-go-lib/utils"
)

func publicEndpoint(method, p string) bool {
	return (method == "GET" && (p == "/configs" || p == "/configs/license" || p == "/configs/version" || p == "/login/captcha")) || (method == "POST" && (p == "/login/in" || p == "/secret/encrypt"))
}
func selfEndpoint(method, p string) bool {
	return (method == "GET" && (p == "/user/token" || p == "/company/current")) || (method == "PUT" && p == "/user/update/password") || (method == "POST" && p == "/login/out")
}

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		urlPath := c.Request.URL.Path
		urlPath = strings.TrimPrefix(urlPath, global.RequestConfig(c).System.RouterPrefix)
		if publicEndpoint(c.Request.Method, urlPath) && (urlPath != "/configs" || utils.GetToken(c) == "") {
			c.Next()
			return
		}

		// 我们这里jwt鉴权取头部信息 token 登录时回返回token信息 这里前端需要把token存储到cookie或者本地localStorage中 不过需要跟后端协商过期时间 可以约定刷新令牌或者重新登录
		token := utils.GetToken(c)
		if token == "" {
			response.NoAuth("未登录或非法访问", c)
			c.Abort()
			return
		}
		if isBlacklist(token) {
			response.NoAuth("您的帐户异地登陆或令牌失效", c)
			utils.ClearToken(c)
			c.Abort()
			return
		}
		j := utils.NewJWT()
		// parseToken 解析token包含的信息
		claims, err := j.ParseToken(token)
		if err != nil {
			if errors.Is(err, utils.TokenExpired) {
				response.NoAuth("授权已过期", c)
				utils.ClearToken(c)
				c.Abort()
				return
			}
			response.NoAuth(err.Error(), c)
			utils.ClearToken(c)
			c.Abort()
			return
		}

		a, loadErr := services.LoadAccess(global.NAV_DB.WithContext(c.Request.Context()), claims.UserGuid)
		if loadErr != nil || claims.AuthVersion != a.User.AuthVersion {
			response.NoAuth("会话已失效，请重新登录", c)
			c.Abort()
			return
		}
		claims.CompanyGuid = a.User.CompanyGuid
		claims.Username = a.User.Username
		claims.RoleGuids = strings.Join(a.User.RoleGuidList, ",")
		claims.RoleCodes = strings.Join(a.User.RoleCodeList, ",")
		c.Set("access", a)
		c.Set("claims", claims)
		c.Set("userGuid", claims.UserGuid)
		c.Set("userName", claims.Username)
		c.Set("companyGuid", claims.CompanyGuid)
		if claims.ExpiresAt.Unix()-time.Now().Unix() < claims.BufferTime {
			dr, _ := utils.ParseDuration(global.RequestConfig(c).JWT.ExpiresTime)
			claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(dr))
			newToken, err := j.CreateTokenByOldToken(token, *claims)
			if err != nil {
				response.NoAuth("令牌续期失败", c)
				c.Abort()
				return
			}
			newClaims, err := j.ParseToken(newToken)
			if err != nil {
				response.NoAuth("令牌续期失败", c)
				c.Abort()
				return
			}
			c.Header("new-token", newToken)
			c.Header("new-expires-at", strconv.FormatInt(newClaims.ExpiresAt.Unix(), 10))
			utils.SetToken(c, newToken, int(dr.Seconds()))
			if global.RequestConfig(c).System.UseMultipoint {
				_ = utils.SetRedisJWT(newToken, newClaims.Username)
			}
		}
		c.Next()

		if newToken, exists := c.Get("new-token"); exists {
			c.Header("new-token", newToken.(string))
		}
		if newExpiresAt, exists := c.Get("new-expires-at"); exists {
			c.Header("new-expires-at", newExpiresAt.(string))
		}
	}
}

// isBlacklist
// @description: 判断JWT是否在黑名单内部
// @param: jwt string
// @return: bool
func isBlacklist(jwt string) bool {
	if global.LOCAL_CACHE == nil {
		return false
	}
	_, ok := global.LOCAL_CACHE.Get(jwt)
	return ok
}
