package domains

type LoginLog struct {
	BaseDataEntity
	UserGuid  string `json:"userGuid" gorm:"size:50;comment:用户GUID"`
	Username  string `json:"username" gorm:"comment:用户名"`
	Device    string `json:"device" gorm:"comment:设备"`
	Ip        string `json:"ip" gorm:"comment:IP"`
	Address   string `json:"address" gorm:"comment:地址"`
	UserAgent string `json:"userAgent" gorm:"comment:浏览器"`
	LoginTime int64  `json:"loginTime" gorm:"comment:登录时间"`
	Type      int    `json:"type" gorm:"comment:类型 0-登录 1-登出"`
	Platform  string `json:"platform" gorm:"comment:平台"`
}

func (LoginLog) TableName() string {
	return "sys_login_log"
}
