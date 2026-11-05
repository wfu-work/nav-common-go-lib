package domains

// SysMenu stores presentation only. Routes and access requirements come
// from the application's registered navigation catalog, independently of roles.
type SysMenu struct {
	Key       string `json:"key" gorm:"column:menu_key;primaryKey;size:64"`
	ParentKey string `json:"parentKey" gorm:"size:64;index"`
	Type      string `json:"type" gorm:"size:10"`
	Name      string `json:"name" gorm:"size:64"`
	Icon      string `json:"icon" gorm:"size:64"`
	TargetKey string `json:"targetKey" gorm:"size:64"`
	Sort      int    `json:"sort"`
	Visible   bool   `json:"visible"`
	BuiltIn   bool   `json:"builtIn"`
}

func (SysMenu) TableName() string { return "sys_menu" }

// SysRoleMenu records explicit navigation grants independently of API dependencies.
type SysRoleMenu struct {
	RoleGuid string `json:"roleGuid" gorm:"primaryKey;size:50"`
	MenuKey  string `json:"menuKey" gorm:"primaryKey;size:64;index"`
}

func (SysRoleMenu) TableName() string { return "sys_role_menu" }
