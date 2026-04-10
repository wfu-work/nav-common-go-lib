package services

import (
	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/gorm"
)

type LoggerService struct {
	CrudService[domains.SysLog]
}

var LoggerServiceApp = new(LoggerService)

func (s *LoggerService) WithDB(db *gorm.DB) *LoggerService {
	cloned := *s
	cloned.CrudService = *s.CrudService.WithDB(db)
	return &cloned
}
