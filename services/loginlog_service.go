package services

import (
	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/gorm"
)

type LoginLogService struct {
	CrudService[domains.LoginLog]
}

var LoginLogServiceApp = new(LoginLogService)

func (s *LoginLogService) WithDB(db *gorm.DB) *LoginLogService {
	cloned := *s
	cloned.CrudService = *s.CrudService.WithDB(db)
	return &cloned
}
