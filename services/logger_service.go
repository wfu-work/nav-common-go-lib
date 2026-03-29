package services

import (
	"github.com/wfu-work/nav-common-go-lib/domains"
)

type LoggerService struct {
	CrudService[domains.SysLog]
}

var LoggerServiceApp = new(LoggerService)
