package services

import (
	"navfirst.com/nav-common-go-lib/domains"
)

type CompanyService struct {
	CrudService[domains.SysCompany]
}

var CompanyServiceApp = new(CompanyService)
