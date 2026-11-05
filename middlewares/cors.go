package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/configs"
	"github.com/wfu-work/nav-common-go-lib/global"
)

const corsHeaders = "Content-Type,Authorization,Token,X-Token,X-User-Id,X-CSRF-Token"
const corsMethods = "GET,POST,PUT,PATCH,DELETE,OPTIONS"
const corsExpose = "Content-Length,New-Token,New-Expires-At"

// Cors allows public cross-origin requests without credential sharing.
func Cors() gin.HandlerFunc        { return corsHandler(true) }
func CorsByRules() gin.HandlerFunc { return corsHandler(false) }

func corsHandler(allowAll bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		config := global.RequestConfig(c)
		origin := c.GetHeader("Origin")
		c.Writer.Header().Add("Vary", "Origin")
		if origin == "" {
			c.Next()
			return
		}
		var rule *configs.CORSWhitelist
		for _, item := range config.Cors.Whitelist {
			if item.AllowOrigin == origin {
				copy := item
				rule = &copy
				break
			}
		}
		if rule == nil && (allowAll || config.Cors.Mode == "allow-all" || config.Cors.Mode == "") {
			rule = &configs.CORSWhitelist{AllowOrigin: "*"}
		}
		if rule == nil {
			if config.Cors.Mode == "strict-whitelist" || c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}
		methods, headers, expose := rule.AllowMethods, rule.AllowHeaders, rule.ExposeHeaders
		if methods == "" {
			methods = corsMethods
		}
		if headers == "" {
			headers = corsHeaders
		}
		if expose == "" {
			expose = corsExpose
		}
		c.Header("Access-Control-Allow-Origin", rule.AllowOrigin)
		c.Header("Access-Control-Allow-Methods", methods)
		c.Header("Access-Control-Allow-Headers", headers)
		c.Header("Access-Control-Expose-Headers", expose)
		if rule.AllowCredentials && rule.AllowOrigin != "*" {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if c.Request.Method == http.MethodOptions {
			c.Writer.Header().Add("Vary", "Access-Control-Request-Method")
			c.Writer.Header().Add("Vary", "Access-Control-Request-Headers")
			requested := c.GetHeader("Access-Control-Request-Method")
			if requested != "" && !containsHeaderValue(methods, requested) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			for _, requested := range strings.Split(c.GetHeader("Access-Control-Request-Headers"), ",") {
				if requested = strings.TrimSpace(requested); requested != "" && !containsHeaderValue(headers, requested) {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func containsHeaderValue(values, value string) bool {
	for _, candidate := range strings.Split(values, ",") {
		if strings.EqualFold(strings.TrimSpace(candidate), value) {
			return true
		}
	}
	return false
}
