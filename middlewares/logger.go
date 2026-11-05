package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/services"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"github.com/wfu-work/nav-common-go-lib/utils/ip2geo"
	"github.com/wfu-work/nav-common-go-lib/vos"
	"go.uber.org/zap"
)

func ApiLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		var body []byte
		if c.Request.Method != http.MethodGet {
			var err error
			body, err = io.ReadAll(c.Request.Body)
			if err != nil {
				global.NAV_LOG.Error("read body from request error:", zap.Error(err))
			} else {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		} else {
			query := c.Request.URL.RawQuery
			query, _ = url.QueryUnescape(query)
			split := strings.Split(query, "&")
			paramsMap := make(map[string]string)
			for _, v := range split {
				kv := strings.Split(v, "=")
				if len(kv) == 2 {
					paramsMap[kv[0]] = kv[1]
				}
			}
			body, _ = json.Marshal(&paramsMap)
		}
		bodyStr := string(body)

		userGuid := utils.GetUserGuid(c)
		username := utils.GetUserName(c)
		companyGuid := utils.GetCompanyGuid(c)
		targetCompany := c.Query("companyGuid")
		if targetCompany == "" {
			var payload struct {
				CompanyGuid string `json:"companyGuid"`
			}
			if json.Unmarshal(body, &payload) == nil {
				targetCompany = payload.CompanyGuid
			}
		}
		if targetCompany == "" {
			targetCompany = companyGuid
		}

		path := c.Request.URL.Path
		path = strings.TrimPrefix(path, global.NAV_CONFIG.System.RouterPrefix)
		method := c.Request.Method
		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()
		var errorsStr string
		for _, err := range c.Errors.Errors() {
			errorsStr += err + "; "
		}

		c.Next()

		if target := c.GetString("targetCompanyGuid"); target != "" {
			targetCompany = target
		}
		domestic, provinces, glo, _ := ip2geo.GetIpLocation(ip)
		offTime := time.Since(start).Milliseconds()
		sysLog := domains.SysLog{
			CompanyGuid:       companyGuid,
			TargetCompanyGuid: targetCompany,
			Username:          username,
			UserGuid:          userGuid,
			Operation:         path,
			Method:            method,
			Params:            bodyStr,
			IP:                ip,
			Address:           glo + "|" + provinces + "|" + domestic,
			UserAgent:         userAgent,
			Exception:         errorsStr,
			Time:              offTime,
		}
		err := services.LoggerServiceApp.Create(sysLog)
		if err != nil {
			global.NAV_LOG.Error("error logger record: ", zap.Error(err))
		}
	}
}

func LoginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()
		var errorsStr string
		for _, err := range c.Errors.Errors() {
			errorsStr += err + "; "
		}
		path := c.Request.URL.Path
		path = strings.TrimPrefix(path, global.NAV_CONFIG.System.RouterPrefix)
		loginType := 0
		userGuid := ""
		username := ""
		if path != "/login/in" {
			loginType = 1
			userGuid = utils.GetUserGuid(c)
			username = utils.GetUserName(c)
		} else {
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				global.NAV_LOG.Error("read body from request error:", zap.Error(err))
			} else {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			}
			var loginVo vos.LoginVo
			err = json.Unmarshal(body, &loginVo)
			if err != nil {
				global.NAV_LOG.Error("read body from request error:", zap.Error(err))
			}
			username = loginVo.Username
			if username != "" {
				user, err := services.UserServiceApp.GetByLoginAccount(username)
				if user == nil {
					global.NAV_LOG.Error("get user by login account error:", zap.Error(err))
					c.Next()
					return
				}
				userGuid = user.Guid
			}
		}

		c.Next()

		domestic, provinces, glo, _ := ip2geo.GetIpLocation(ip)
		loginLog := domains.LoginLog{
			Username:  username,
			UserGuid:  userGuid,
			Ip:        ip,
			UserAgent: userAgent,
			Device:    utils.GetDeviceType(c),
			Address:   glo + "|" + provinces + "|" + domestic,
			LoginTime: time.Now().UnixMilli(),
			Platform:  "web",
			Type:      loginType,
		}
		err := services.LoginLogServiceApp.Create(loginLog)
		if err != nil {
			global.NAV_LOG.Error("error login logger record: ", zap.Error(err))
		}
	}
}
