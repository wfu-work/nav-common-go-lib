package domains

import (
	"strings"
)

type SysUser struct {
	BaseDataEntity
	CompanyGuid  string    `json:"companyGuid" gorm:"size:50;comment:企业guid"`
	Username     string    `json:"username" gorm:"index;size:50;comment:用户名"`
	Password     string    `json:"-"  gorm:"comment:用户密码"`
	Salt         string    `json:"-"  gorm:"comment:盐"`
	NickName     string    `json:"nickName" gorm:"size:50;comment:用户昵称"`
	Avatar       string    `json:"avatar" gorm:"comment:用户头像"`
	Phone        string    `json:"phone"  gorm:"comment:手机号"`
	Email        string    `json:"email"  gorm:"comment:邮箱"`
	Gender       int       `json:"gender" gorm:"default:0;comment:性别（0.未知，1.男，2.女）"`
	Enable       int       `json:"enable" gorm:"default:1;comment:用户是否被冻结 1正常 0冻结"`
	RoleList     []SysRole `json:"-" gorm:"-"`
	RoleGuidList []string  `json:"roleGuidList" gorm:"-"`
	RoleCodeList []string  `json:"roleCodeList" gorm:"-"`
	Abilities    []string  `json:"abilities" gorm:"-"`
}

type Login interface {
	GetCompanyGuid() string
	GetUsername() string
	GetUserGuid() string
	GetUserInfo() any
	GetRoleGuids() string
	GetRoleCodes() string
}

var _ Login = new(SysUser)

func (SysUser) TableName() string {
	return "sys_user"
}

func (s SysUser) GetBaseData() BaseDataEntity {
	return s.BaseDataEntity
}

func (s *SysUser) GetUsername() string {
	return s.Username
}

func (s *SysUser) GetNickname() string {
	return s.NickName
}

func (s *SysUser) GetUserGuid() string {
	return s.Guid
}

func (s *SysUser) GetUserInfo() any {
	return *s
}

func (s *SysUser) GetCompanyGuid() string {
	return s.CompanyGuid
}

func (s *SysUser) GetRoleGuids() string {
	return joinField(s.RoleList, func(u SysRole) string { return u.Guid }, ",")
}

func (s *SysUser) GetRoleCodes() string {
	return joinField(s.RoleList, func(u SysRole) string { return u.Code }, ",")
}

func joinField[T any](items []T, get func(T) string, sep string) string {
	var out []string
	for _, item := range items {
		out = append(out, get(item))
	}
	return strings.Join(out, sep)
}
