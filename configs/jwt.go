package configs

import (
	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	CookieSecure   bool   `mapstructure:"cookie-secure" json:"cookie-secure" yaml:"cookie-secure"`
	CookieSameSite string `mapstructure:"cookie-same-site" json:"cookie-same-site" yaml:"cookie-same-site"`
	SigningKey     string `mapstructure:"signing-key" json:"signing-key" yaml:"signing-key"`    // jwt签名
	ExpiresTime    string `mapstructure:"expires-time" json:"expires-time" yaml:"expires-time"` // 过期时间
	BufferTime     string `mapstructure:"buffer-time" json:"buffer-time" yaml:"buffer-time"`    // 缓冲时间
	Issuer         string `mapstructure:"issuer" json:"issuer" yaml:"issuer"`                   // 签发者
}

// CustomClaims structure
type CustomClaims struct {
	BaseClaims
	BufferTime int64
	jwt.RegisteredClaims
}

type BaseClaims struct {
	AuthVersion uint
	UserGuid    string
	Username    string
	CompanyGuid string
	RoleGuids   string
	RoleCodes   string
}
