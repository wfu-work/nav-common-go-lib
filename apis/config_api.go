package apis

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/response"
)

type ConfigApi struct{}

// SaveConfig 修改配置
// @Summary 修改配置
// @Description 修改配置
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param key path string true "配置的key"
// @Success 200 {object} response.Response{data=bool,msg=string}
// @Router /config/{key} [post]
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
	global.Lock.Lock()
	global.NAV_VIPER.Set(key, req.Value)
	err := configService.SaveConfigToFile()
	global.Lock.Unlock()
	if err != nil {
		global.NAV_LOG.Error("保存失败", zap.Error(err))
		response.Fail(false, c)
		return
	}
	response.Ok(true, c)
}

// GetConfig 获取配置
// @Summary 获取配置
// @Description 获取配置
// @Tags 系统模块
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param key path string true "配置的key"
// @Success 200 {object} response.Response{data=string,msg=string}
// @Router /config/{key} [get]
func (i ConfigApi) GetConfig(c *gin.Context) {
	key := c.Param("key")
	key = strings.ReplaceAll(key, "/", ".")
	global.Lock.Lock()
	val := global.NAV_VIPER.Get(key)
	global.Lock.Unlock()
	if val == nil {
		global.NAV_LOG.Error("获取失败")
		response.Fail(nil, c)
		return
	}
	response.Ok(val, c)
}
