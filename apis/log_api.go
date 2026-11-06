package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

type LogApi struct{}

// Save 写入系统日志。
//
// @Summary 写入系统日志
// @Description 需要登录及接口权限，按提交的日志对象写入记录；guid 省略时自动生成，传入已有 guid 时按通用服务规则更新可变字段。常规审计日志由中间件自动记录，无需客户端重复提交。
// @Tags 系统日志
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body domains.SysLog true "系统日志对象"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /log [post]
func (i LogApi) Save(c *gin.Context) {
	var entity domains.SysLog
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
	err = loggerService.Create(entity)
	if err != nil {
		global.NAV_LOG.Error("创建失败!", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// DeleteByGuid 删除系统日志。
//
// @Summary 删除系统日志
// @Description 需要登录及接口权限，按 GUID 软删除指定日志。处理器本身不额外按企业过滤，宿主应限制此管理接口的授权范围。
// @Tags 系统日志
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /log/{guid} [delete]
func (i LogApi) DeleteByGuid(c *gin.Context) {
	guid := c.Param("guid")
	err := loggerService.DeleteByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("删除失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// GetByGuid 查询系统日志详情。
//
// @Summary 查询系统日志详情
// @Description 需要登录及接口权限，按 GUID 查询日志；记录不存在时 data 为 null。处理器本身不额外按企业过滤，宿主应限制此管理接口的授权范围。
// @Tags 系统日志
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Success 200 {object} response.Response{data=domains.SysLog} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /log/{guid} [get]
func (i LogApi) GetByGuid(c *gin.Context) {
	guid := c.Param("guid")
	t, err := loggerService.GetByGuid(guid)
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(t, c)
}

// List 分页查询系统日志。
//
// @Summary 分页查询系统日志
// @Description 需要登录及接口权限，默认按创建顺序倒序，支持 asc/desc 指定模型字段排序；page、size 必须提供非零值。处理器本身不额外按企业过滤。
// @Tags 系统日志
// @Security ApiKeyAuth
// @Produce json
// @Param page query int true "页码，从 1 开始" default(1)
// @Param size query int true "每页条数，最多 100 条" default(10)
// @Param content query string false "模糊搜索：操作路径 operation"
// @Param asc query string false "按指定模型字段升序；只允许模型字段名或列名"
// @Param desc query string false "按指定模型字段降序；与 asc 同传时优先使用 desc"
// @Success 200 {object} response.Response{data=domains.PageResult{data=[]domains.SysLog}} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /log/list [get]
func (i LogApi) List(c *gin.Context) {
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
	list, total, err := loggerService.List(pageInfo, "operation")
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
