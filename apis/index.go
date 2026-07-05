package apis

import "github.com/wfu-work/nav-common-go-lib/services"

var SysApiGroupApp = new(SysApiGroup)

type SysApiGroup struct {
	LogApi
	OsApi
	ConfigApi
	FileApi
	LoginApi
	SecretApi
	LoginLogApi
	UserApi
	CompanyApi
	RoleApi
	PermissionApi
	RegisterApi
}

var (
	loggerService     = services.SysServiceGroupApp.LoggerService
	DBService         = services.SysServiceGroupApp.DBService
	osService         = services.SysServiceGroupApp.OsService
	configService     = services.SysServiceGroupApp.ConfigService
	fileService       = services.SysServiceGroupApp.FileService
	loginService      = services.SysServiceGroupApp.LoginService
	loginLogService   = services.SysServiceGroupApp.LoginLogService
	userService       = services.SysServiceGroupApp.UserService
	userRoleService   = services.SysServiceGroupApp.UserRoleService
	companyService    = services.SysServiceGroupApp.CompanyService
	roleService       = services.SysServiceGroupApp.RoleService
	permissionService = services.SysServiceGroupApp.PermissionService
	registerService   = services.SysServiceGroupApp.RegisterService
)
