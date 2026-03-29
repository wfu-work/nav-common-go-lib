package services

import (
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
)

type UserRoleService struct {
}

var UserRoleServiceApp = new(UserRoleService)

func (s UserRoleService) GetByUserGuid(userGuid string) (*domains.SysRole, error) {
	var result domains.SysUserRole
	err := global.NAV_DB.Where("user_guid = ?", userGuid).First(&result).Error
	if err != nil {
		return nil, err
	}
	role, err := RoleServiceApp.GetByGuid(result.RoleGuid)
	return role, err
}
