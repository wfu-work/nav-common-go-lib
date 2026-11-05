package utils

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/configs"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	SigningKey []byte
}

var (
	TokenHeader           = "Authorization"
	TokenValid            = errors.New("未知错误")
	TokenExpired          = errors.New("token已过期")
	TokenNotValidYet      = errors.New("token尚未激活")
	TokenMalformed        = errors.New("这不是一个token")
	TokenSignatureInvalid = errors.New("无效签名")
	TokenInvalid          = errors.New("无法处理此token")
)

func NewJWT() *JWT {
	return &JWT{
		[]byte(global.Config().JWT.SigningKey),
	}
}

func (j *JWT) CreateClaims(baseClaims configs.BaseClaims) configs.CustomClaims {
	bf, _ := ParseDuration(global.Config().JWT.BufferTime)
	ep, _ := ParseDuration(global.Config().JWT.ExpiresTime)
	claims := configs.CustomClaims{
		BaseClaims: baseClaims,
		BufferTime: int64(bf / time.Second), // 缓冲时间1天 缓冲时间内会获得新的token刷新令牌 此时一个用户会存在两个有效令牌 但是前端只留一个 另一个会丢失
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{"NAV-IPC-WEB"},           // 受众
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1000)), // 签名生效时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ep)),    // 过期时间 7天  配置文件
			Issuer:    global.Config().JWT.Issuer,                // 签名的发行者
		},
	}
	return claims
}

// CreateToken 创建一个token
func (j *JWT) CreateToken(claims configs.CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}

// CreateTokenByOldToken 旧token 换新token 使用归并回源避免并发问题
func (j *JWT) CreateTokenByOldToken(oldToken string, claims configs.CustomClaims) (string, error) {
	v, err, _ := global.NAV_Concurrency_Control.Do("JWT:"+oldToken, func() (interface{}, error) {
		return j.CreateToken(claims)
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

// ParseToken 解析 token
func (j *JWT) ParseToken(tokenString string) (*configs.CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &configs.CustomClaims{}, func(token *jwt.Token) (i interface{}, e error) {
		return j.SigningKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, TokenExpired
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, TokenMalformed
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, TokenSignatureInvalid
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, TokenNotValidYet
		default:
			return nil, TokenInvalid
		}
	}
	if token != nil {
		if claims, ok := token.Claims.(*configs.CustomClaims); ok && token.Valid {
			return claims, nil
		}
	}
	return nil, TokenValid
}

// SetRedisJWT
// @description: jwt存入redis并设置过期时间
// @param: jwt string, userName string
// @return: err error
func SetRedisJWT(jwt string, userName string) (err error) {
	dr, err := ParseDuration(global.Config().JWT.ExpiresTime)
	if err != nil {
		return err
	}
	timer := dr
	global.LOCAL_CACHE.Set(userName, jwt, timer)
	return err
}

func ParseDuration(d string) (time.Duration, error) { return configs.ParseDuration(d) }

func GetToken(c *gin.Context) string {
	var token string
	authHeader := c.Request.Header.Get(TokenHeader)
	if authHeader == "" {
		authHeader = c.Query(TokenHeader)
	}
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)
	}
	if token == "" {
		j := NewJWT()
		token, _ = c.Cookie(TokenHeader)
		if token == "" {
			return ""
		}
		claims, err := j.ParseToken(token)
		if err != nil {
			global.NAV_LOG.Error("重新写入cookie token失败,未能成功解析token,请检查请求头是否存在token且claims是否为规定结构")
			return token
		}
		SetToken(c, token, int(claims.ExpiresAt.Unix()-time.Now().Unix()))
	}
	return token
}

func SetToken(c *gin.Context, token string, maxAge int) {
	config := global.RequestConfig(c).JWT
	sameSite := http.SameSiteLaxMode
	switch config.CookieSameSite {
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}
	c.SetSameSite(sameSite)
	// Host-only cookies prevent subdomain scope from depending on an untrusted Host header.
	c.SetCookie(TokenHeader, token, maxAge, "/", "", config.CookieSecure || c.Request.TLS != nil, true)
}

func ClearToken(c *gin.Context) { SetToken(c, "", -1) }

func GetClaims(c *gin.Context) (*configs.CustomClaims, error) {
	if v, ok := c.Get("claims"); ok {
		return v.(*configs.CustomClaims), nil
	}
	token := GetToken(c)
	j := NewJWT()
	claims, err := j.ParseToken(token)
	if err != nil {
		global.NAV_LOG.Error("从Gin的Context中获取从jwt解析信息失败, 请检查请求头是否存在token且claims是否为规定结构")
	}
	return claims, err
}

// GetUserGuid 从Gin的Context中获取从jwt解析出来的用户UUID
func GetUserGuid(c *gin.Context) string {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return ""
		} else {
			return cl.UserGuid
		}
	} else {
		waitUse := claims.(*configs.CustomClaims)
		return waitUse.UserGuid
	}
}

// GetCompanyGuid 从Gin的Context中获取从jwt解析出来的企业guid
func GetCompanyGuid(c *gin.Context) string {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return ""
		} else {
			return cl.CompanyGuid
		}
	} else {
		waitUse := claims.(*configs.CustomClaims)
		return waitUse.CompanyGuid
	}
}

// GetRoleGuids 从Gin的Context中获取从jwt解析出来的用户角色id
func GetRoleGuids(c *gin.Context) string {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return ""
		} else {
			return cl.RoleGuids
		}
	} else {
		waitUse := claims.(*configs.CustomClaims)
		return waitUse.RoleGuids
	}
}

// GetUserName 从Gin的Context中获取从jwt解析出来的用户名
func GetUserName(c *gin.Context) string {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return ""
		} else {
			return cl.Username
		}
	} else {
		waitUse := claims.(*configs.CustomClaims)
		return waitUse.Username
	}
}

func LoginToken(user domains.Login) (token string, claims configs.CustomClaims, err error) {
	j := NewJWT()
	claims = j.CreateClaims(configs.BaseClaims{
		CompanyGuid: user.GetCompanyGuid(),
		UserGuid:    user.GetUserGuid(),
		Username:    user.GetUsername(),
		RoleGuids:   user.GetRoleGuids(),
		RoleCodes:   user.GetRoleCodes(),
	})
	if u, ok := user.(*domains.SysUser); ok {
		claims.AuthVersion = u.AuthVersion
	}
	token, err = j.CreateToken(claims)
	return token, claims, err
}
