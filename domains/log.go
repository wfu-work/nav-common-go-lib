package domains

type SysLog struct {
	BaseDataEntity
	CompanyGuid string `json:"companyGuid" gorm:"size:50;comment:企业guid"`
	UserGuid    string `json:"userGuid" gorm:"size:50;comment:用户GUID"`
	Username    string `json:"username" gorm:"comment:用户名"`
	Operation   string `json:"operation" gorm:"comment:用户操作"`
	Method      string `json:"method" gorm:"comment:请求方法"`
	Params      string `json:"params" gorm:"type:text;comment:请求参数"`
	IP          string `json:"ip" gorm:"comment:IP地址"`
	Time        int64  `json:"time" gorm:"comment:执行时长(毫秒)"`
	Address     string `json:"address" gorm:"comment:地址"`
	UserAgent   string `json:"userAgent" gorm:"comment:浏览器"`
	Exception   string `json:"exception" gorm:"comment:异常信息"`
}

func (SysLog) TableName() string {
	return "sys_log"
}

func (s SysLog) GetBaseData() BaseDataEntity {
	return s.BaseDataEntity
}
