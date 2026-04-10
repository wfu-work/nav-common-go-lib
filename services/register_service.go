package services

import (
	"errors"

	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"github.com/wfu-work/nav-common-go-lib/vos"
	"gorm.io/gorm"
)

type RegisterService struct {
	db *gorm.DB
}

var RegisterServiceApp = new(RegisterService)

func (s *RegisterService) WithDB(db *gorm.DB) *RegisterService {
	cloned := *s
	cloned.db = db
	return &cloned
}

func (s *RegisterService) DB() *gorm.DB {
	if s != nil && s.db != nil {
		return s.db
	}
	return global.NAV_DB
}

func (s *RegisterService) RegisterUser(v *vos.RegisterVo) (bool, error) {
	user, err := UserServiceApp.WithDB(s.DB()).GetByUserName(v.Username)
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
	err = s.DB().Create(sysUser).Error
	return err == nil, err
}
