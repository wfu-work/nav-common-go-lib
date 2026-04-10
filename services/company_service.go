package services

import (
	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/gorm"
)

type CompanyService struct {
	CrudService[domains.SysCompany]
}

var CompanyServiceApp = new(CompanyService)

func (s *CompanyService) WithDB(db *gorm.DB) *CompanyService {
	cloned := *s
	cloned.CrudService = *s.CrudService.WithDB(db)
	return &cloned
}
