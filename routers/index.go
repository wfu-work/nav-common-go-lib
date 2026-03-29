package routers

import "github.com/wfu-work/nav-common-go-lib/apis"

var SysRouterGroupApp = new(SysRouterGroup)

type SysRouterGroup struct {
	LogRouter
	OsRouter
	ConfigRouter
	LoginRouter
	FileRouter
	SecretRouter
	LoginLogRouter
	UserRouter
	CompanyRouter
	RoleRouter
	RegisterRouter
}

var (
	logApi      = apis.SysApiGroupApp.LogApi
	osApi       = apis.SysApiGroupApp.OsApi
	configApi   = apis.SysApiGroupApp.ConfigApi
	loginApi    = apis.SysApiGroupApp.LoginApi
	loginLogApi = apis.SysApiGroupApp.LoginLogApi
	fileApi     = apis.SysApiGroupApp.FileApi
	secretApi   = apis.SysApiGroupApp.SecretApi
	userApi     = apis.SysApiGroupApp.UserApi
	companyApi  = apis.SysApiGroupApp.CompanyApi
	roleApi     = apis.SysApiGroupApp.RoleApi
	registerApi = apis.SysApiGroupApp.RegisterApi
)
