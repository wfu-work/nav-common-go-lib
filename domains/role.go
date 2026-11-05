package domains

type SysRole struct {
	BaseDataEntity
	CompanyGuid            string `json:"companyGuid" gorm:"size:50;uniqueIndex:idx_company_role_code;index"`
	Enabled                bool   `json:"enabled" gorm:"default:true"`
	PermissionsInitialized bool   `json:"-"`
	BuiltIn                bool   `json:"builtIn"`
	Name                   string `json:"name" gorm:"size:50;comment:角色名"`
	Code                   string `json:"code" gorm:"size:50;uniqueIndex:idx_company_role_code;comment:标识"`
	Sort                   int    `json:"sort" gorm:"comment:排序"`
	Remark                 string `json:"remark" gorm:"comment:备注"`
}

func (SysRole) TableName() string {
	return "sys_role"
}

func (s SysRole) GetBaseData() BaseDataEntity {
	return s.BaseDataEntity
}
