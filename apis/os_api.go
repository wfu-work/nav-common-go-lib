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

// GetInfo 获取服务器的状态
// @Summary 获取服务器的状态
// @Description 获取服务器的状态
// @Tags 系统模块
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=bool,msg=string}
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

// RestartService 重启服务
// @Summary 重启服务
// @Description 重启服务
// @Tags 系统模块
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=bool,msg=string}
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

// CheckLocalPort 检测本地端口是否监听
// @Summary 检测本地端口是否监听
// @Description 检测本地端口是否监听
// @Tags 系统模块
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=bool,msg=string}
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
