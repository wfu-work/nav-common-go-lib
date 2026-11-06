package apis

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

type OsApi struct{}

// GetInfo 获取服务器运行状态。
//
// @Summary 获取服务器运行状态
// @Description 需要登录及接口权限。采集操作系统及 Go 运行时、CPU 使用率、内存与配置磁盘挂载点信息；采集失败返回业务错误。
// @Tags 服务器管理
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} response.Response{data=utils.Server} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /os/info [get]
func (i OsApi) GetInfo(c *gin.Context) {
	result, err := osService.GetServerInfo()
	if err != nil {
		global.NAV_LOG.Error("获取失败", zap.Error(err))
		response.FailWithDetailed(nil, err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// RestartService 重启指定系统服务。
//
// @Summary 重启指定系统服务
// @Description 需要登录及接口权限。执行 systemctl restart 重启指定 systemd 服务；service 省略时使用 raind。运行环境须支持 systemd 且进程具有相应权限，失败原因通过 msg 返回。
// @Tags 服务器管理
// @Security ApiKeyAuth
// @Produce json
// @Param service query string false "要重启的系统服务名" default(raind)
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /os/restart/service [post]
func (i OsApi) RestartService(c *gin.Context) {
	service := c.Query("service")
	if service == "" {
		service = "raind"
	}
	err := utils.RestartService(service)
	if err != nil {
		global.NAV_LOG.Error("重启失败", zap.Error(err))
		response.FailWithDetailed(false, err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// CheckLocalPort 检测本机 TCP 端口。
//
// @Summary 检测本机 TCP 端口
// @Description 需要登录及接口权限。尝试连接本机 127.0.0.1 指定 TCP 端口；data=true 表示连接成功，false 表示未能连接，参数缺失或无法解析时返回业务失败。
// @Tags 服务器管理
// @Security ApiKeyAuth
// @Produce json
// @Param port query int true "本机 TCP 端口号，通常为 1 至 65535"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /os/check/port [get]
func (i OsApi) CheckLocalPort(c *gin.Context) {
	port := c.Query("port")
	if port == "" {
		global.NAV_LOG.Error("端口不能为空")
		response.Fail(false, c)
		return
	}
	p, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		global.NAV_LOG.Error("端口转int失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	res := utils.CheckLocalPort(int(p))
	response.Ok(res, c)
}
