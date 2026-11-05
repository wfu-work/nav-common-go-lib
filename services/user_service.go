package services

import (
	"errors"
	"strings"

	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"github.com/wfu-work/nav-common-go-lib/vos"
	"gorm.io/gorm"
)

type UserService struct {
	CrudService[domains.SysUser]
}

var UserServiceApp = new(UserService)

func (s *UserService) WithDB(db *gorm.DB) *UserService {
	cloned := *s
	cloned.CrudService = *s.CrudService.WithDB(db)
	return &cloned
}

func (s *UserService) GetByUserName(username string) (*domains.SysUser, error) {
	var result = &domains.SysUser{}
	err := s.DB().Where("username = ?", username).First(result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
	}
	return result, err
}

func (s *UserService) GetByLoginAccount(account string) (*domains.SysUser, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return nil, nil
	}
	normalized := strings.ToLower(account)

	user, err := s.getFirstUserByCondition("LOWER(username) = ?", normalized)
	if err != nil || user != nil {
		return user, err
	}
	return s.getFirstUserByCondition("LOWER(email) = ?", normalized)
}

func (s *UserService) getFirstUserByCondition(query any, args ...any) (*domains.SysUser, error) {
	var result = &domains.SysUser{}
	err := s.DB().Where(query, args...).First(result).Error
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
	if len(newPassword) < 8 {
		return false, errors.New("密码至少8位")
	}
	newPasswordHash := utils.BcryptHash(newPassword)
	err = s.Update(*user, "password", newPasswordHash)
	if err != nil {
		return false, err
	}
	return true, nil
}
