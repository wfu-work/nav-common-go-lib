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

// Navigation 获取当前用户可见导航。
//
// @Summary 获取当前用户可见导航
// @Description 仅需有效登录身份。根据菜单可见性、实时用户身份、角色菜单分配与 ACL 生成分组导航，省略没有可见子菜单的分组。宿主须先注册导航目录。
// @Tags 导航菜单
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} response.Response{data=[]services.NavigationSection} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Router /navigation [get]
func (NavigationApi) Navigation(c *gin.Context) {
	s, ok := navigationRequest(c, false)
	if !ok {
		return
	}
	data, err := s.Navigation(services.RequestAccess(c))
	accessReply(c, data, err)
}

// Catalog 获取菜单管理目录。
//
// @Summary 获取菜单管理目录
// @Description 仅平台管理员可操作。返回平铺菜单 menus、宿主注册的页面 targets 和允许选择的 icons；页面路由与访问要求由宿主目录提供。
// @Tags 导航菜单
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} response.Response{data=services.NavigationCatalog} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /menu/catalog [get]
func (NavigationApi) Catalog(c *gin.Context) {
	s, ok := navigationRequest(c, true)
	if !ok {
		return
	}
	data, err := s.Catalog(services.RequestAccess(c))
	accessReply(c, data, err)
}

// Save 创建或更新导航分组及菜单。
//
// @Summary 创建或更新导航分组及菜单
// @Description 仅平台管理员可操作。key 为空时新建，否则更新已有节点；type 为 group 或 menu，已有节点不能改变类型。
// @Description name 必填且不超过 64 字，sort 为 0 至 9999，icon 必须来自目录 icons。group 只能位于顶层，parentKey、targetKey 为空；menu 须指定有效分组 parentKey 和已注册页面 targetKey。
// @Description visible 控制显示，省略为 false；内置菜单不能更换页面，受保护菜单及其分组必须显示。此接口仅修改展示配置，不直接授予角色权限。
// @Tags 导航菜单
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body domains.SysMenu true "菜单或分组配置；builtIn 由服务端维护"
// @Success 200 {object} response.Response{data=domains.SysMenu} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /menu [post]
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

// Delete 删除自定义菜单或空分组。
//
// @Summary 删除自定义菜单或空分组
// @Description 仅平台管理员可操作。内置菜单及受保护页面入口不可删除；分组下存在子菜单时，须先移动或删除子菜单。删除时清理关联的角色菜单授权。
// @Tags 导航菜单
// @Security ApiKeyAuth
// @Produce json
// @Param key path string true "菜单或分组的唯一键"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /menu/{key} [delete]
func (NavigationApi) Delete(c *gin.Context) {
	s, ok := navigationRequest(c, true)
	if !ok {
		return
	}
	err := s.DeleteMenu(services.RequestAccess(c), c.Param("key"))
	accessReply(c, true, err)
}
