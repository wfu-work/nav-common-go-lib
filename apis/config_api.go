package apis

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"go.uber.org/zap"
)

type ConfigApi struct{}

// SaveConfig 写入配置文件中的指定键。
//
// @Summary 写入配置文件中的指定键
// @Description 需要登录及接口权限。key 使用点号表示嵌套项，例如 jwt.expires-time；JSON 中 value 可为字符串、数字、布尔、数组或对象。
// @Description 请求示例：{"value":"7d"}。
// @Description 返回成功仅表示配置文件已原子保存。监听器仅对校验通过的 JWT 有效期、刷新窗口、CORS 和验证码变更热更新；其他配置需重启，非法或涉及重启的变更不会替换当前运行快照。
// @Tags 系统配置
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param key path string true "配置键，例如 jwt.expires-time"
// @Param data body object{value=interface{}} true "配置值对象，value 为要保存的值"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /config/{key} [put]
func (i ConfigApi) SaveConfig(c *gin.Context) {
	key := c.Param("key")
	key = strings.ReplaceAll(key, "/", ".")
	var req struct {
		Value interface{} `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		global.NAV_LOG.Error("请求格式错误", zap.Error(err))
		response.Fail(false, c)
		return
	}
	err := configService.SaveConfigValue(key, req.Value)
	if err != nil {
		global.NAV_LOG.Error("保存失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// GetConfig 读取配置文件中的指定键。
//
// @Summary 读取配置文件中的指定键
// @Description 需要登录及接口权限。key 使用点号表示嵌套项，例如 jwt.expires-time；返回文件配置合并默认值后的值，类型随键变化。
// @Description 读取的是磁盘配置，不保证与当前运行快照一致；值不存在时返回业务失败。
// @Tags 系统配置
// @Security ApiKeyAuth
// @Produce json
// @Param key path string true "配置键，例如 jwt.expires-time"
// @Success 200 {object} response.Response{data=interface{}} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /config/{key} [get]
func (i ConfigApi) GetConfig(c *gin.Context) {
	key := c.Param("key")
	key = strings.ReplaceAll(key, "/", ".")
	val, err := configService.GetConfigValue(key)
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.Fail(nil, c)
		return
	}
	if val == nil {
		global.NAV_LOG.Error("获取失败")
		response.Fail(nil, c)
		return
	}
	response.Ok(val, c)
}
