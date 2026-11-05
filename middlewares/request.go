package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/global"
)

// RequestLimits pins one configuration snapshot and bounds request bodies.
func RequestLimits() gin.HandlerFunc {
	return func(c *gin.Context) {
		config := global.Config()
		c.Request = c.Request.WithContext(global.ContextWithConfig(c.Request.Context(), config))
		limit := config.System.MaxRequestBytes
		if limit <= 0 {
			limit = 16 << 20
		}
		if strings.HasPrefix(c.Request.URL.Path, config.System.RouterPrefix+"/file/upload") {
			limit = config.System.MaxUploadBytes
			if limit <= 0 {
				limit = 32 << 20
			}
		}
		if c.Request.ContentLength > limit {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
