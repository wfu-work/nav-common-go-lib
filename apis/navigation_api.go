package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/services"
)

type NavigationApi struct{}

func navigationRequest(c *gin.Context, admin bool) (*services.NavigationService, bool) {
	actor := services.RequestAccess(c)
	if actor == nil {
		response.NoAuth("未登录", c)
		return nil, false
	}
	if admin && !actor.PlatformAdmin() {
		response.NoPermission("仅平台管理员可管理菜单", c)
		return nil, false
	}
	service, err := services.NavigationServiceFor(global.NAV_DB.WithContext(c.Request.Context()))
	if err != nil {
		accessReply(c, nil, err)
		return nil, false
	}
	return service, true
}
func (NavigationApi) Navigation(c *gin.Context) {
	s, ok := navigationRequest(c, false)
	if !ok {
		return
	}
	data, err := s.Navigation(services.RequestAccess(c))
	accessReply(c, data, err)
}
func (NavigationApi) Catalog(c *gin.Context) {
	s, ok := navigationRequest(c, true)
	if !ok {
		return
	}
	data, err := s.Catalog(services.RequestAccess(c))
	accessReply(c, data, err)
}
func (NavigationApi) Save(c *gin.Context) {
	s, ok := navigationRequest(c, true)
	if !ok {
		return
	}
	var req domains.SysMenu
	if err := c.ShouldBindJSON(&req); err != nil {
		accessReply(c, nil, err)
		return
	}
	data, err := s.SaveMenu(services.RequestAccess(c), req)
	accessReply(c, data, err)
}
func (NavigationApi) Delete(c *gin.Context) {
	s, ok := navigationRequest(c, true)
	if !ok {
		return
	}
	err := s.DeleteMenu(services.RequestAccess(c), c.Param("key"))
	accessReply(c, true, err)
}
