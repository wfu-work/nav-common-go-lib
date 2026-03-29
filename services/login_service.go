package services

import (
	"errors"

	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"github.com/wfu-work/nav-common-go-lib/vos"
	"go.uber.org/zap"
)

type LoginService struct{}

var LoginServiceApp = new(LoginService)

func (s *LoginService) LoginIn(v *vos.LoginVo) (*vos.TokenVo, error) {
	user, err := UserServiceApp.GetByUserName(v.Username)
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
	role, err := UserRoleServiceApp.GetByUserGuid(user.Guid)
	if err == nil && role != nil {
		user.RoleList = append(user.RoleList, *role)
	}
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
