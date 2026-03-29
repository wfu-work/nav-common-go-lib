package domains

type SysUserRole struct {
	BaseEntity
	UserGuid string `json:"userGuid" gorm:"size:50;comment:用户GUID"`
	RoleGuid string `json:"roleGuid" gorm:"size:50;comment:角色GUID"`
}

func (SysUserRole) TableName() string {
	return "sys_user_role"
}
