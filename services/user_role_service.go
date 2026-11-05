package services

import (
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"gorm.io/gorm"
)

type UserRoleService struct {
	db *gorm.DB
}

var UserRoleServiceApp = new(UserRoleService)

func (s *UserRoleService) WithDB(db *gorm.DB) *UserRoleService {
	cloned := *s
	cloned.db = db
	return &cloned
}

func (s *UserRoleService) DB() *gorm.DB {
	if s != nil && s.db != nil {
		return s.db
	}
	return global.NAV_DB
}

func (s UserRoleService) GetByUserGuid(userGuid string) (*domains.SysRole, error) {
	var result domains.SysUserRole
	err := s.DB().Where("user_guid = ?", userGuid).First(&result).Error
	if err != nil {
		return nil, err
	}
	role, err := RoleServiceApp.WithDB(s.DB()).GetByGuid(result.RoleGuid)
	return role, err
}

// ListByUserGuid returns every active role belonging to the user's company.
func (s UserRoleService) ListByUserGuid(userGuid string) ([]domains.SysRole, error) {
	var roles []domains.SysRole
	err := s.DB().Model(&domains.SysRole{}).Joins("JOIN sys_user_role ur ON ur.role_guid=sys_role.guid JOIN sys_user u ON u.guid=ur.user_guid").Where("u.guid = ? AND sys_role.company_guid=u.company_guid AND sys_role.enabled = ? AND u.deleted_time IS NULL", userGuid, true).Find(&roles).Error
	return roles, err
}
