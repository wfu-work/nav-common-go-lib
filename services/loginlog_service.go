package services

import (
	"navfirst.com/nav-common-go-lib/domains"
)

type LoginLogService struct {
	CrudService[domains.LoginLog]
}

var LoginLogServiceApp = new(LoginLogService)
