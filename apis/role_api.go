package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/services"
)

type RoleApi struct{}

// Save 创建或更新全局角色。
//
// @Summary 创建或更新全局角色
// @Description 仅平台管理员可操作，guid 为空时新建；角色全局共享，无需指定企业。
// @Description name、code 必填且不超过 50 字，remark 不超过 500 字；code 自动转为大写并全局唯一，不能使用 SUPER_ADMIN 或创建 COMPANY_ADMIN。
// @Description enabled 省略时新建默认 true、更新保留原值。内置角色不能改编码，企业管理员角色不能停用。保存后刷新权限缓存。
// @Tags 角色管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body services.ManagedRoleRequest true "角色资料"
// @Success 200 {object} response.Response{data=domains.SysRole} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /role [post]
func (RoleApi) Save(c *gin.Context) {
	var req services.ManagedRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		accessReply(c, nil, err)
		return
	}
	role, err := services.RoleServiceApp.WithDB(global.NAV_DB.WithContext(c.Request.Context())).SaveManagedRole(services.RequestAccess(c), req)
	if err == nil && !refreshCasbinAfterPermissionChange(c) {
		return
	}
	accessReply(c, role, err)
}

// DeleteByGuid 删除全局角色。
//
// @Summary 删除全局角色
// @Description 仅平台管理员可操作，全局角色供各企业用户共用。内置角色和仍关联用户的角色不能删除；成功时清理角色权限和菜单授权并刷新权限缓存。
// @Tags 角色管理
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /role/{guid} [delete]
func (RoleApi) DeleteByGuid(c *gin.Context) {
	err := services.RoleServiceApp.WithDB(global.NAV_DB.WithContext(c.Request.Context())).DeleteManagedRole(services.RequestAccess(c), c.Param("guid"))
	if err == nil && !refreshCasbinAfterPermissionChange(c) {
		return
	}
	accessReply(c, true, err)
}

// GetByGuid 查询全局角色详情。
//
// @Summary 查询全局角色详情
// @Description 仅平台管理员可操作。角色不存在或超出管理范围时返回业务失败。
// @Tags 角色管理
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Success 200 {object} response.Response{data=domains.SysRole} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /role/{guid} [get]
func (RoleApi) GetByGuid(c *gin.Context) {
	role, err := services.RoleServiceApp.WithDB(global.NAV_DB.WithContext(c.Request.Context())).GetManagedRole(services.RequestAccess(c), c.Param("guid"))
	accessReply(c, role, err)
}

// Authorization 查询角色授权详情与可选目录。
//
// @Summary 查询角色授权详情与可选目录
// @Description 仅平台管理员可操作，全局角色供各企业用户共用。返回角色、菜单目录、可选权限、已选 menuKeys 和 permissionGuids。
// @Description 菜单的 assignable、restrictionReason 表示是否允许分配及限制原因；需由宿主预先注册导航目录。
// @Tags 角色管理
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Success 200 {object} response.Response{data=services.RoleAuthorizationData} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /role/{guid}/authorization [get]
func (RoleApi) Authorization(c *gin.Context) {
	service, err := services.NavigationServiceFor(global.NAV_DB.WithContext(c.Request.Context()))
	if err != nil {
		accessReply(c, nil, err)
		return
	}
	data, err := service.RoleAuthorization(services.RequestAccess(c), c.Param("guid"))
	accessReply(c, data, err)
}

// SaveAuthorization 替换角色菜单与操作授权。
//
// @Summary 替换角色菜单与操作授权
// @Description 仅平台管理员可操作，全局角色供各企业用户共用。menuKeys、permissionGuids 均为完整替换列表，空数组或省略表示清空。
// @Description menuKeys 只允许可分配的菜单项；permissionGuids 只允许所选菜单下启用的业务按钮权限。菜单访问权限及 API 依赖自动展开，不可授予平台专用权限。
// @Description 授权及菜单关联在事务内保存，成功后刷新权限缓存；宿主须预先注册导航目录。
// @Tags 角色管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "记录 GUID"
// @Param data body services.RoleAuthorizationRequest true "完整菜单键列表和按钮权限 GUID 列表"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /role/{guid}/authorization [put]
func (RoleApi) SaveAuthorization(c *gin.Context) {
	var req services.RoleAuthorizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		accessReply(c, nil, err)
		return
	}
	service, err := services.NavigationServiceFor(global.NAV_DB.WithContext(c.Request.Context()))
	if err != nil {
		accessReply(c, nil, err)
		return
	}
	err = service.SaveRoleAuthorization(services.RequestAccess(c), c.Param("guid"), req)
	if err == nil && !refreshCasbinAfterPermissionChange(c) {
		return
	}
	accessReply(c, true, err)
}

// List 分页查询全局角色。
//
// @Summary 分页查询全局角色
// @Description 仅平台管理员可查询，列出全局业务角色，排除平台身份角色；按 sort、创建顺序升序排列，不支持自定义排序。
// @Tags 角色管理
// @Security ApiKeyAuth
// @Produce json
// @Param page query int false "页码，从 1 开始" default(1)
// @Param size query int false "每页条数，最多 100 条" default(10)
// @Param content query string false "模糊搜索：角色名称 name、编码 code"
// @Success 200 {object} response.Response{data=domains.PageResult{data=[]domains.SysRole}} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /role/list [get]
func (RoleApi) List(c *gin.Context) {
	actor := services.RequestAccess(c)
	if actor == nil || !actor.PlatformAdmin() {
		response.NoPermission("仅平台管理员可管理全局角色", c)
		return
	}
	var q domains.PageInfo
	if err := c.ShouldBindQuery(&q); err != nil {
		accessReply(c, nil, err)
		return
	}
	db := global.NAV_DB.WithContext(c.Request.Context()).Model(&domains.SysRole{}).Where("company_guid = '' AND code <> ?", "SUPER_ADMIN")
	if q.Content != "" {
		db = db.Where("(name LIKE ? OR code LIKE ?)", "%"+q.Content+"%", "%"+q.Content+"%")
	}
	var n int64
	if err := db.Count(&n).Error; err != nil {
		accessReply(c, nil, err)
		return
	}
	var roles []domains.SysRole
	err := db.Scopes(q.Paginate()).Order("sort ASC,id ASC").Find(&roles).Error
	accessReply(c, domains.PageResult{Data: roles, Total: n, Page: q.Page, Size: q.Size}, err)
}

// ListAll 查询全局角色选项列表。
//
// @Summary 查询全局角色选项列表
// @Description 平台管理员或企业管理员可查询，列出全局业务角色，排除平台身份角色，包含已停用角色；按 sort、创建顺序升序排列，不分页。
// @Tags 角色管理
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} response.Response{data=[]domains.SysRole} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /role/list/all [get]
func (RoleApi) ListAll(c *gin.Context) {
	actor := services.RequestAccess(c)
	if actor == nil || !actor.CompanyAdmin() {
		response.NoPermission("需要企业管理员权限", c)
		return
	}
	var roles []domains.SysRole
	err := global.NAV_DB.WithContext(c.Request.Context()).Where("company_guid = '' AND code <> ?", "SUPER_ADMIN").Order("sort ASC,id ASC").Find(&roles).Error
	accessReply(c, roles, err)
}
