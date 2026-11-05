package services

import (
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
