package apis

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/services"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"gorm.io/gorm"
)

type CompanyApi struct{}

func platformOnly(c *gin.Context) bool {
	a := services.RequestAccess(c)
	if a == nil || !a.PlatformAdmin() {
		response.NoPermission("仅 admin 可管理企业", c)
		return false
	}
	return true
}
func (CompanyApi) Save(c *gin.Context) {
	if !platformOnly(c) {
		return
	}
	var req struct {
		domains.SysCompany
		Enabled       *bool  `json:"enabled"`
		AdminUsername string `json:"adminUsername"`
		AdminPassword string `json:"adminPassword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		accessReply(c, nil, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	if req.Name == "" || req.Code == "" {
		response.FailWithMessage("企业名称和编码不能为空", c)
		return
	}
	id := req.Guid
	err := global.NAV_DB.Transaction(func(tx *gorm.DB) error {
		company := req.SysCompany
		fresh := company.Guid == ""
		if !fresh {
			var old domains.SysCompany
			if err := tx.Where("guid = ?", company.Guid).First(&old).Error; err != nil {
				return err
			}
			company.BaseDataEntity = old.BaseDataEntity
			company.Enabled = old.Enabled
		} else {
			company.BaseDataEntity = domains.BaseDataEntity{Guid: uuid.NewString()}
			company.Enabled = true
			company.Status = 1
		}
		if req.Enabled != nil {
			company.Enabled = *req.Enabled
		}
		enabled := company.Enabled
		if err := tx.Save(&company).Error; err != nil {
			return err
		}
		if err := tx.Model(&company).UpdateColumn("enabled", enabled).Error; err != nil {
			return err
		}
		id = company.Guid
		if err := services.InitializeCompanyRoles(tx, company.Guid); err != nil {
			return err
		}
		if fresh {
			username := strings.ToLower(strings.TrimSpace(req.AdminUsername))
			if username == "" || username == "admin" {
				return errors.New("请设置企业管理员账号，admin 为保留账号")
			}
			var n int64
			if err := tx.Model(&domains.SysUser{}).Where("LOWER(username) = ? OR LOWER(email) = ?", username, username).Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				return errors.New("管理员账号已存在")
			}
			plain, err := utils.AesDecrypt(req.AdminPassword)
			if err != nil || len(plain) < 8 {
				return errors.New("初始密码至少8位")
			}
			hash, err := utils.BcryptHashE(plain)
			if err != nil {
				return err
			}
			u := domains.SysUser{BaseDataEntity: domains.BaseDataEntity{Guid: uuid.NewString()}, CompanyGuid: company.Guid, Username: username, Password: hash, Enable: 1, AuthVersion: 1}
			if err := tx.Create(&u).Error; err != nil {
				return err
			}
			var role domains.SysRole
			if err := tx.Where("company_guid = ? AND code = ?", company.Guid, "COMPANY_ADMIN").First(&role).Error; err != nil {
				return err
			}
			return tx.Create(&domains.SysUserRole{UserGuid: u.Guid, RoleGuid: role.Guid}).Error
		}
		return nil
	})
	if err == nil {
		c.Set("targetCompanyGuid", id)
		refreshCasbinAfterPermissionChange(c)
	}
	if !c.IsAborted() {
		accessReply(c, id, err)
	}
}
func companyQuery(c *gin.Context) *gorm.DB {
	db := global.NAV_DB.Model(&domains.SysCompany{})
	a := services.RequestAccess(c)
	if a == nil {
		return db.Where("1=0")
	}
	if !a.PlatformAdmin() {
		db = db.Where("guid = ?", a.User.CompanyGuid)
	}
	return db
}
func (CompanyApi) List(c *gin.Context) {
	var q domains.PageInfo
	if err := c.ShouldBindQuery(&q); err != nil {
		accessReply(c, nil, err)
		return
	}
	db := companyQuery(c)
	if q.Content != "" {
		db = db.Where("(name LIKE ? OR code LIKE ?)", "%"+q.Content+"%", "%"+q.Content+"%")
	}
	var n int64
	if err := db.Count(&n).Error; err != nil {
		accessReply(c, nil, err)
		return
	}
	companies := []domains.SysCompany{}
	err := db.Scopes(q.Paginate()).Order("id DESC").Find(&companies).Error
	if !services.RequestAccess(c).PlatformAdmin() {
		for i := range companies {
			companies[i].ApiKey = ""
			companies[i].Certificate = ""
		}
	}
	accessReply(c, domains.PageResult{Data: companies, Total: n, Page: q.Page, Size: q.Size}, err)
}
func (CompanyApi) ListAll(c *gin.Context) {
	companies := []domains.SysCompany{}
	err := companyQuery(c).Order("id ASC").Find(&companies).Error
	for i := range companies {
		companies[i].ApiKey = ""
		companies[i].Certificate = ""
	}
	accessReply(c, companies, err)
}
func (CompanyApi) GetByGuid(c *gin.Context) {
	var company domains.SysCompany
	err := companyQuery(c).Where("guid = ?", c.Param("guid")).First(&company).Error
	a := services.RequestAccess(c)
	if !a.PlatformAdmin() {
		company.ApiKey = ""
		company.Certificate = ""
	}
	accessReply(c, company, err)
}
func (CompanyApi) Current(c *gin.Context) {
	a := services.RequestAccess(c)
	if a == nil {
		response.NoAuth("未登录", c)
		return
	}
	response.Ok(a.User.Company, c)
}
func (CompanyApi) UpdateCurrent(c *gin.Context) {
	a := services.RequestAccess(c)
	if a == nil || !a.CompanyAdmin() || a.User.CompanyGuid == "" {
		response.NoPermission("需要企业管理员权限", c)
		return
	}
	var req struct {
		Name     string `json:"name"`
		Logo     string `json:"logo"`
		Phone    string `json:"phone"`
		Email    string `json:"email"`
		Location string `json:"location"`
		Brief    string `json:"brief"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		accessReply(c, nil, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		response.FailWithMessage("企业名称不能为空", c)
		return
	}
	err := global.NAV_DB.Model(&domains.SysCompany{}).Where("guid = ?", a.User.CompanyGuid).Updates(map[string]any{"name": req.Name, "logo": req.Logo, "phone": req.Phone, "email": req.Email, "location": req.Location, "brief": req.Brief}).Error
	accessReply(c, true, err)
}
func (CompanyApi) DeleteByGuid(c *gin.Context) {
	if !platformOnly(c) {
		return
	}
	response.FailWithMessage("企业保留业务数据，请使用停用功能", c)
}
func (CompanyApi) Pass(c *gin.Context) {
	if !platformOnly(c) {
		return
	}
	err := global.NAV_DB.Model(&domains.SysCompany{}).Where("guid = ?", c.Param("guid")).UpdateColumn("status", 1).Error
	accessReply(c, true, err)
}
func (CompanyApi) Refuse(c *gin.Context) {
	if !platformOnly(c) {
		return
	}
	err := global.NAV_DB.Model(&domains.SysCompany{}).Where("guid = ?", c.Param("guid")).UpdateColumn("status", 2).Error
	accessReply(c, true, err)
}
