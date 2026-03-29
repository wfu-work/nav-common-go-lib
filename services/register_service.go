package services

import (
	"errors"

	"navfirst.com/nav-common-go-lib/domains"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/utils"
	"navfirst.com/nav-common-go-lib/vos"
)

type RegisterService struct{}

var RegisterServiceApp = new(RegisterService)

func (s *RegisterService) RegisterUser(v *vos.RegisterVo) (bool, error) {
	user, err := UserServiceApp.GetByUserName(v.Username)
	if err != nil {
		return false, errors.New("数据库错误")
	}
	if user != nil {
		return false, errors.New("用户不存在")
	}
	password, err := utils.AesDecrypt(v.Password)
	if err != nil {
		return false, errors.New("解密错误")
	}
	bcryptPassword := utils.BcryptHash(password)
	sysUser := &domains.SysUser{Username: v.Username, Password: bcryptPassword, Email: v.Email, Enable: 1}
	err = global.NAV_DB.Create(sysUser).Error
	return err == nil, err
}
