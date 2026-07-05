package domains

const (
	PermissionTypeAPI    = "api"
	PermissionTypeMenu   = "menu"
	PermissionTypeButton = "button"
)

type SysPermission struct {
	BaseDataEntity
	ParentGuid string `json:"parentGuid" gorm:"column:parent_guid;size:50;index;comment:父级权限GUID"`
	Name       string `json:"name" gorm:"size:100;comment:权限名称"`
	Code       string `json:"code" gorm:"size:100;uniqueIndex;comment:权限编码"`
	Type       string `json:"type" gorm:"size:30;index;comment:权限类型"`
	Path       string `json:"path" gorm:"size:255;index;comment:API路径或前端路由"`
	Method     string `json:"method" gorm:"size:20;index;comment:HTTP方法"`
	Enabled    bool   `json:"enabled" gorm:"default:true;index;comment:是否启用"`
	Sort       int    `json:"sort" gorm:"comment:排序"`
	Remark     string `json:"remark" gorm:"size:255;comment:备注"`
}

func (SysPermission) TableName() string {
	return "sys_permission"
}

func (s SysPermission) GetBaseData() BaseDataEntity {
	return s.BaseDataEntity
}
