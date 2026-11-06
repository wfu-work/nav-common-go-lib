package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/services"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PermissionApi struct{}

// Save 创建或更新权限定义。
//
// @Summary 创建或更新权限定义
// @Description 仅平台管理员可操作。guid 为空时创建，否则按 GUID 保存；name、code 必填，code 全局唯一。
// @Description type 省略默认 api，API 权限须提供 path；method 省略表示任意方法。scope 区分 platform/company，dependencies 引用权限编码。保存后刷新权限缓存。
// @Tags 权限管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body domains.SysPermission true "权限定义与依赖"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /permission [post]
// @Router /permission [put]
func (i PermissionApi) Save(c *gin.Context) {
	if !platformOnly(c) {
		return
	}
	var entity domains.SysPermission
	if err := c.ShouldBindJSON(&entity); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := permissionService.Save(entity); err != nil {
		global.NAV_LOG.Error("保存权限失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	if !refreshCasbinAfterPermissionChange(c) {
		return
	}
	response.Ok(true, c)
}

// DeleteByGuid 删除权限定义。
//
// @Summary 删除权限定义
// @Description 仅平台管理员可操作。删除角色与该权限的关联，再软删除权限定义，成功后刷新权限缓存。
// @Tags 权限管理
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /permission/{guid} [delete]
func (i PermissionApi) DeleteByGuid(c *gin.Context) {
	if !platformOnly(c) {
		return
	}
	guid := c.Param("guid")
	if err := permissionService.DeletePermission(guid); err != nil {
		global.NAV_LOG.Error("删除权限失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	if !refreshCasbinAfterPermissionChange(c) {
		return
	}
	response.Ok(true, c)
}

// List 分页查询权限定义。
//
// @Summary 分页查询权限定义
// @Description 平台管理员可查询全部权限；企业用户仅查询 scope=company 且 enabled=true 的权限。默认按创建顺序倒序，可用 asc/desc 指定模型字段排序。
// @Tags 权限管理
// @Security ApiKeyAuth
// @Produce json
// @Param page query int true "页码，从 1 开始" default(1)
// @Param size query int true "每页条数，最多 100 条" default(10)
// @Param content query string false "模糊搜索：名称 name、编码 code、路径 path、方法 method"
// @Param asc query string false "按指定模型字段升序；只允许模型字段名或列名"
// @Param desc query string false "按指定模型字段降序；与 asc 同传时优先使用 desc"
// @Success 200 {object} response.Response{data=domains.PageResult{data=[]domains.SysPermission}} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /permission/list [get]
func (i PermissionApi) List(c *gin.Context) {
	var pageInfo domains.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := utils.Verify(pageInfo, utils.PageInfoVerify); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	svc := permissionService.WithDB(permissionDB(c))
	list, total, err := svc.List(pageInfo, "name,code,path,method")
	if err != nil {
		global.NAV_LOG.Error("获取权限列表失败", zap.Error(err))
		response.Fail(nil, c)
		return
	}
	response.Ok(domains.PageResult{
		Data:  list,
		Total: total,
		Page:  pageInfo.Page,
		Size:  pageInfo.Size,
	}, c)
}

// ListAll 按字段查询权限列表。
//
// @Summary 按字段查询权限列表
// @Description 不分页。平台管理员可查询全部权限；企业用户仅查询已启用的企业权限。
// @Description 支持通过模型字段进行等值筛选，例如 code、type、scope、enabled、parentGuid、path、method；每个查询键取第一个值，多条件按 AND 组合，未知字段返回业务失败。
// @Tags 权限管理
// @Security ApiKeyAuth
// @Produce json
// @Param code query string false "权限编码，等值匹配"
// @Param type query string false "权限类型" Enums(api,menu,button)
// @Param scope query string false "权限范围" Enums(platform,company)
// @Param enabled query bool false "是否启用"
// @Param parentGuid query string false "父级权限 GUID"
// @Param path query string false "API 路径或前端路由，等值匹配"
// @Param method query string false "HTTP 方法或 *，等值匹配"
// @Success 200 {object} response.Response{data=[]domains.SysPermission} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /permission/list/all [get]
func (i PermissionApi) ListAll(c *gin.Context) {
	query := c.Request.URL.Query()
	params := make(map[string]any)
	for key, val := range query {
		if len(val) > 0 {
			params[key] = val[0]
		}
	}
	result, err := permissionService.WithDB(permissionDB(c)).ListByFields(params)
	if err != nil {
		global.NAV_LOG.Error("获取权限列表失败", zap.Error(err))
		response.Fail(nil, c)
		return
	}
	response.Ok(result, c)
}

func refreshCasbinAfterPermissionChange(c *gin.Context) bool {
	if err := middlewares.RefreshCasbin(); err != nil {
		global.NAV_LOG.Error("刷新Casbin策略失败", zap.Error(err))
		response.FailWithMessage("权限已保存，但刷新权限缓存失败："+err.Error(), c)
		c.Abort()
		return false
	}
	return true
}

func permissionDB(c *gin.Context) *gorm.DB {
	db := global.NAV_DB
	a := services.RequestAccess(c)
	if a == nil {
		return db.Where("1=0")
	}
	if !a.PlatformAdmin() {
		db = db.Where("scope = ? AND enabled = ?", "company", true)
	}
	return db
}
