package domains

type SysCompany struct {
	BaseDataEntity
	Enabled      bool   `json:"enabled" gorm:"default:true;index"`
	Name         string `json:"name" gorm:"size:50;comment:公司名称"`
	Code         string `json:"code" gorm:"size:50;uniqueIndex;comment:公司编号"`
	Phone        string `json:"phone" gorm:"size:11;comment:公司电话"`
	Email        string `json:"email" gorm:"size:50;comment:公司邮箱"`
	Logo         string `json:"logo" gorm:"comment:图标"`
	Certificate  string `json:"certificate" gorm:"comment:营业执照"`
	Location     string `json:"location" gorm:"comment:地址"`
	ApiKey       string `json:"apiKey" gorm:"comment:API-KEY"`
	MaxDevice    int    `json:"maxDevice" gorm:"comment:最大设备数量"`
	RegisterTime int64  `json:"registerTime" gorm:"comment:注册时间"`
	EndTime      int64  `json:"endTime" gorm:"comment:到期时间"`
	Status       int    `json:"status" gorm:"comment:状态 0：审核中 1：审核通过 2：审核拒绝"`
	Brief        string `json:"brief" gorm:"comment:简介"`
	Remark       string `json:"remark" gorm:"comment:备注"`
}

func (SysCompany) TableName() string {
	return "sys_company"
}

func (s SysCompany) GetBaseData() BaseDataEntity {
	return s.BaseDataEntity
}
