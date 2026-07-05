package domains

type SysRolePermission struct {
	BaseEntity
	RoleGuid       string `json:"roleGuid" gorm:"size:50;uniqueIndex:idx_role_permission;index;comment:角色GUID"`
	PermissionGuid string `json:"permissionGuid" gorm:"size:50;uniqueIndex:idx_role_permission;index;comment:权限GUID"`
}

func (SysRolePermission) TableName() string {
	return "sys_role_permission"
}
