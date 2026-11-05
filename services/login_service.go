package services

import (
	"errors"
	"gorm.io/gorm"

	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"github.com/wfu-work/nav-common-go-lib/vos"
	"go.uber.org/zap"
)

type LoginService struct{ db *gorm.DB }

func (s *LoginService) WithDB(db *gorm.DB) *LoginService {
	cloned := *s
	cloned.db = db
	return &cloned
}
func (s *LoginService) DB() *gorm.DB {
	if s.db != nil {
		return s.db
	}
	return global.NAV_DB
}

var LoginServiceApp = new(LoginService)

func (s *LoginService) LoginIn(v *vos.LoginVo) (*vos.TokenVo, error) {
	user, err := UserServiceApp.WithDB(s.DB()).GetByLoginAccount(v.Username)
	if err != nil {
		return nil, errors.New("数据库错误")
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}
	password, err := utils.AesDecrypt(v.Password)
	if err != nil {
		return nil, errors.New("解密错误")
	}
	if ok := utils.BcryptCheck(password, user.Password); !ok {
		return nil, errors.New("密码错误")
	}
	a, err := LoadAccess(s.DB(), user.Guid)
	if err != nil {
		return nil, err
	}
	user = &a.User
	user.RoleList = a.Roles

	token, claims, err := utils.LoginToken(user)
	if err != nil {
		global.NAV_LOG.Error("获取token失败", zap.Error(err))
		return nil, err
	}
	tokenVo := &vos.TokenVo{
		Token:        token,
		RefreshToken: token,
		Expired:      claims.RegisteredClaims.ExpiresAt.UnixMilli(),
		TokenType:    "Bearer",
	}
	return tokenVo, nil
}
