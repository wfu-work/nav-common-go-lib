package services

import (
	"errors"

	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/gorm"
)

type RoleService struct {
	CrudService[domains.SysRole]
}

var RoleServiceApp = new(RoleService)

func (s *RoleService) WithDB(db *gorm.DB) *RoleService {
	cloned := *s
	cloned.CrudService = *s.CrudService.WithDB(db)
	return &cloned
}

func (s RoleService) CreateRole(d domains.SysRole) error {
	if d.Guid == "" {
		bean, _ := s.GetRoleByName(d.Name)
		if bean != nil {
			return errors.New("角色名称已存在")
		}
		bean, _ = s.GetRoleByCode(d.Code)
		if bean != nil {
			return errors.New("角色标识已存在")
		}
	}
	return s.Create(d)
}

func (s RoleService) GetRoleByName(name string) (*domains.SysRole, error) {
	var result domains.SysRole
	err := s.DB().Where("name = ?", name).First(&result).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &result, err
}

func (s RoleService) GetRoleByCode(code string) (*domains.SysRole, error) {
	var result domains.SysRole
	err := s.DB().Where("code = ?", code).First(&result).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &result, err
}
