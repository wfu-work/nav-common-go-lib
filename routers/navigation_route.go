package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/apis"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

// InitNavigationRouter uses the authenticated chain, independent of application tenant callbacks.
func InitNavigationRouter(public *gin.RouterGroup) {
	api := apis.NavigationApi{}
	authenticated := public.Group("")
	authenticated.Use(middlewares.JWTAuth())
	authenticated.GET("navigation", api.Navigation)
	authenticated.GET("menu/catalog", api.Catalog)
	write := authenticated.Group("menu").Use(middlewares.ApiLogger())
	write.POST("", api.Save)
	write.DELETE(":key", api.Delete)
}
