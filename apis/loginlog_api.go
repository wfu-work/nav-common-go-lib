package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

type LoginLogApi struct{}

// Save 直接写入登录日志；默认路由未注册此方法。
//
// 常规登录、登出日志由 LoginLogger 中间件自动记录。宿主如需手动挂载，
// 应自行定义路由及鉴权范围；此处不声明 Swagger 路由，避免生成不存在的接口。
func (i LoginLogApi) Save(c *gin.Context) {
	var entity domains.LoginLog
	err := c.ShouldBindJSON(&entity)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(entity, utils.RoleVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = loginLogService.Create(entity)
	if err != nil {
		global.NAV_LOG.Error("创建失败!", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// DeleteByGuid 删除登录日志。
//
// @Summary 删除登录日志
// @Description 需要登录及接口权限，按 GUID 软删除指定日志。处理器本身不额外按企业过滤，宿主应限制此管理接口的授权范围。
// @Tags 登录日志
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /login/log/{guid} [delete]
func (i LoginLogApi) DeleteByGuid(c *gin.Context) {
	guid := c.Param("guid")
	err := loginLogService.DeleteByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("删除失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// GetByGuid 查询登录日志详情。
//
// @Summary 查询登录日志详情
// @Description 需要登录及接口权限，按 GUID 查询日志；记录不存在时 data 为 null。处理器本身不额外按企业过滤，宿主应限制此管理接口的授权范围。
// @Tags 登录日志
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Success 200 {object} response.Response{data=domains.LoginLog} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /login/log/{guid} [get]
func (i LoginLogApi) GetByGuid(c *gin.Context) {
	guid := c.Param("guid")
	t, err := loginLogService.GetByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(t, c)
}

// List 分页查询登录日志。
//
// @Summary 分页查询登录日志
// @Description 需要登录及接口权限，默认按创建顺序倒序，支持 asc/desc 指定模型字段排序；page、size 必须提供非零值。处理器本身不额外按企业过滤。
// @Description 当前登录日志模型没有 operation 字段，请勿传入 content 搜索参数；此接口目前只支持分页与排序。
// @Tags 登录日志
// @Security ApiKeyAuth
// @Produce json
// @Param page query int true "页码，从 1 开始" default(1)
// @Param size query int true "每页条数，最多 100 条" default(10)
// @Param asc query string false "按指定模型字段升序；只允许模型字段名或列名"
// @Param desc query string false "按指定模型字段降序；与 asc 同传时优先使用 desc"
// @Success 200 {object} response.Response{data=domains.PageResult{data=[]domains.LoginLog}} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /login/log/list [get]
func (i LoginLogApi) List(c *gin.Context) {
	var pageInfo domains.PageInfo
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(pageInfo, utils.PageInfoVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := loginLogService.List(pageInfo, "operation")
	if err != nil {
		global.NAV_LOG.Error("获取失败!", zap.Error(err))
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
