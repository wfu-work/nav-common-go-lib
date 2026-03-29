package services

import (
	"errors"

	"gorm.io/gorm"
	"navfirst.com/nav-common-go-lib/domains"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/utils"
	"navfirst.com/nav-common-go-lib/vos"
)

type UserService struct {
	CrudService[domains.SysUser]
}

var UserServiceApp = new(UserService)

func (s *UserService) GetByUserName(username string) (*domains.SysUser, error) {
	var result = &domains.SysUser{}
	err := global.NAV_DB.Where("username = ?", username).First(result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
	}
	return result, err
}

func (s *UserService) UpdatePassword(userGuid string, passwordVo *vos.PasswordVo) (bool, error) {
	user, err := s.GetByGuid(userGuid)
	if err != nil || user == nil {
		return false, errors.New("用户不存在")
	}
	password, err := utils.AesDecrypt(passwordVo.OldPassword)
	if err != nil {
		return false, errors.New("解密错误")
	}
	if ok := utils.BcryptCheck(password, user.Password); !ok {
		return false, errors.New("原始密码错误")
	}
	newPassword, err := utils.AesDecrypt(passwordVo.NewPassword)
	if err != nil {
		return false, errors.New("解密错误")
	}
	newPasswordHash := utils.BcryptHash(newPassword)
	err = s.Update(*user, "password", newPasswordHash)
	if err != nil {
		return false, err
	}
	return true, nil
}
