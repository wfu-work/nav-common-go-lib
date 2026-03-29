package services

import (
	"navfirst.com/nav-common-go-lib/domains"
)

type LoggerService struct {
	CrudService[domains.SysLog]
}

var LoggerServiceApp = new(LoggerService)
