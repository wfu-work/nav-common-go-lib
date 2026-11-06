package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/services"
)

type RoleApi struct{}

// Save 创建或更新企业角色。
//
// @Summary 创建或更新企业角色
// @Description 仅平台管理员或企业管理员可操作。guid 为空时新建；平台管理员必须指定 companyGuid，企业管理员只能管理所属企业，角色不能转移企业。
// @Description name、code 必填且不超过 50 字，remark 不超过 500 字；code 自动转为大写并在企业内唯一，不能使用 SUPER_ADMIN 或创建 COMPANY_ADMIN。
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

// DeleteByGuid 删除企业角色。
//
// @Summary 删除企业角色
// @Description 仅平台管理员或企业管理员可操作，企业管理员仅管理所属企业角色。内置角色和仍关联用户的角色不能删除；成功时清理角色权限和菜单授权并刷新权限缓存。
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

// GetByGuid 查询企业角色详情。
//
// @Summary 查询企业角色详情
// @Description 仅平台管理员或企业管理员可操作，企业管理员仅查询所属企业角色。角色不存在或超出管理范围时返回业务失败。
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
// @Description 仅平台管理员或企业管理员可操作，企业管理员仅管理所属企业角色。返回角色与企业名称、菜单目录、可选权限、已选 menuKeys 和 permissionGuids。
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
// @Description 仅平台管理员或企业管理员可操作，企业管理员仅管理所属企业角色。menuKeys、permissionGuids 均为完整替换列表，空数组或省略表示清空。
// @Description menuKeys 只允许可分配的菜单项；permissionGuids 只允许所选菜单下启用的企业按钮权限。菜单访问权限及 API 依赖自动展开，不可授予平台专用权限。
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

// List 分页查询企业角色。
//
// @Summary 分页查询企业角色
// @Description 仅列出具有企业归属的角色。企业用户限定当前企业，平台管理员可查询全部或指定企业；按 sort、创建顺序升序排列，不支持自定义排序。
// @Tags 角色管理
// @Security ApiKeyAuth
// @Produce json
// @Param page query int false "页码，从 1 开始" default(1)
// @Param size query int false "每页条数，最多 100 条" default(10)
// @Param content query string false "模糊搜索：角色名称 name、编码 code"
// @Param companyGuid query string false "仅平台管理员可选：限定目标企业；企业用户始终限定当前企业"
// @Success 200 {object} response.Response{data=domains.PageResult{data=[]domains.SysRole}} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /role/list [get]
func (RoleApi) List(c *gin.Context) {
	var q domains.PageInfo
	if err := c.ShouldBindQuery(&q); err != nil {
		accessReply(c, nil, err)
		return
	}
	db := services.AccessDB(c, &domains.SysRole{}).Where("company_guid <> ''")
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

// ListAll 查询企业角色选项列表。
//
// @Summary 查询企业角色选项列表
// @Description 企业用户限定当前企业，平台管理员可查询全部或指定企业。仅列出具有企业归属的角色，包含已停用角色；按 sort、创建顺序升序排列，不分页。
// @Tags 角色管理
// @Security ApiKeyAuth
// @Produce json
// @Param companyGuid query string false "仅平台管理员可选：限定目标企业；企业用户始终限定当前企业"
// @Success 200 {object} response.Response{data=[]domains.SysRole} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /role/list/all [get]
func (RoleApi) ListAll(c *gin.Context) {
	var roles []domains.SysRole
	err := services.AccessDB(c, &domains.SysRole{}).Where("company_guid <> ''").Order("sort ASC,id ASC").Find(&roles).Error
	accessReply(c, roles, err)
}
