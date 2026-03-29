package services

import (
	"github.com/wfu-work/nav-common-go-lib/domains"
)

type CompanyService struct {
	CrudService[domains.SysCompany]
}

var CompanyServiceApp = new(CompanyService)
