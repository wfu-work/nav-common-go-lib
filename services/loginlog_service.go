package services

import (
	"github.com/wfu-work/nav-common-go-lib/domains"
)

type LoginLogService struct {
	CrudService[domains.LoginLog]
}

var LoginLogServiceApp = new(LoginLogService)
