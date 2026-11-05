package middlewares

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/services"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"github.com/wfu-work/nav-common-go-lib/utils/ip2geo"
)

const auditBodyLimit = 32 << 10

type replayBody struct {
	io.Reader
	io.Closer
}

// auditParams reads only a bounded prefix and replays it to the real handler.
// Binary, oversized and invalid JSON bodies are never copied into audit storage.
func auditParams(c *gin.Context) (string, map[string]any, error) {
	if c.Request.Method == http.MethodGet {
		params := make(map[string]any)
		for key, values := range c.Request.URL.Query() {
			params[key] = values
		}
		redact(params)
		data, _ := json.Marshal(params)
		if len(data) > auditBodyLimit {
			return "[query omitted: size limit]", nil, nil
		}
		return string(data), params, nil
	}
	contentType := strings.Split(c.GetHeader("Content-Type"), ";")[0]
	if contentType != "application/json" && !strings.HasSuffix(contentType, "+json") {
		return "[body omitted]", nil, nil
	}
	if c.Request.Body == nil {
		return "", nil, nil
	}
	original := c.Request.Body
	data, err := io.ReadAll(io.LimitReader(original, auditBodyLimit+1))
	c.Request.Body = replayBody{Reader: io.MultiReader(bytes.NewReader(data), original), Closer: original}
	if err != nil {
		return "", nil, err
	}
	if len(data) > auditBodyLimit {
		return "[body omitted: size limit]", nil, nil
	}
	var value any
	if json.Unmarshal(data, &value) != nil {
		return "[invalid JSON omitted]", nil, nil
	}
	redact(value)
	encoded, _ := json.Marshal(value)
	params, _ := value.(map[string]any)
	return string(encoded), params, nil
}

func redact(value any) {
	switch object := value.(type) {
	case map[string]any:
		for key, child := range object {
			name := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
			if strings.Contains(name, "password") || strings.Contains(name, "token") || strings.Contains(name, "secret") || strings.Contains(name, "apikey") || name == "authorization" || name == "signingkey" {
				object[key] = "[REDACTED]"
			} else {
				redact(child)
			}
		}
	case []any:
		for _, child := range object {
			redact(child)
		}
	}
}

func readAuditParams(c *gin.Context) (string, map[string]any, bool) {
	body, params, err := auditParams(c)
	if err == nil {
		return body, params, true
	}
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		c.AbortWithStatus(http.StatusRequestEntityTooLarge)
	} else {
		c.AbortWithStatus(http.StatusBadRequest)
	}
	return "", nil, false
}

func ApiLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		body, params, ok := readAuditParams(c)
		if !ok {
			return
		}
		config := global.RequestConfig(c)
		userGuid, username, company := utils.GetUserGuid(c), utils.GetUserName(c), utils.GetCompanyGuid(c)
		target := c.Query("companyGuid")
		if target == "" && params != nil {
			target, _ = params["companyGuid"].(string)
		}
		if target == "" {
			target = company
		}
		c.Next()
		if explicit := c.GetString("targetCompanyGuid"); explicit != "" {
			target = explicit
		}
		domestic, province, country, _ := ip2geo.GetIpLocation(c.ClientIP())
		recordAudit(auditEvent{operation: &domains.SysLog{
			CompanyGuid: company, TargetCompanyGuid: target, UserGuid: userGuid, Username: username,
			Operation: strings.TrimPrefix(c.Request.URL.Path, config.System.RouterPrefix), Method: c.Request.Method,
			Params: body, IP: c.ClientIP(), Address: country + "|" + province + "|" + domestic, UserAgent: c.Request.UserAgent(),
			Exception: strings.Join(c.Errors.Errors(), "; "), Time: time.Since(start).Milliseconds(),
		}})
	}
}

func LoginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, params, ok := readAuditParams(c)
		if !ok {
			return
		}
		config := global.RequestConfig(c)
		path := strings.TrimPrefix(c.Request.URL.Path, config.System.RouterPrefix)
		username, guid, kind := "", "", 0
		if path == "/login/in" {
			username, _ = params["username"].(string)
		} else {
			username, guid, kind = utils.GetUserName(c), utils.GetUserGuid(c), 1
		}
		c.Next()
		if guid == "" && username != "" {
			if user, err := services.UserServiceApp.GetByLoginAccount(username); err == nil && user != nil {
				guid = user.Guid
			}
		}
		domestic, province, country, _ := ip2geo.GetIpLocation(c.ClientIP())
		recordAudit(auditEvent{login: &domains.LoginLog{
			Username: username, UserGuid: guid, Ip: c.ClientIP(), UserAgent: c.Request.UserAgent(), Device: utils.GetDeviceType(c),
			Address: country + "|" + province + "|" + domestic, LoginTime: time.Now().UnixMilli(), Platform: "web", Type: kind,
		}})
	}
}
