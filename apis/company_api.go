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

// Save 创建或更新企业。
//
// @Summary 创建或更新企业
// @Description 仅平台管理员 admin 可操作。guid 为空时创建企业，否则更新指定企业；name、code 必填，code 会转为大写。
// @Description 新建时还须提供 adminUsername 和经 POST /secret/encrypt 加密的 adminPassword，明文密码至少 8 位；企业、预置角色、初始管理员在同一事务创建。
// @Description 新建默认 enabled=true、status=1；更新时应提交完整企业资料，enabled 省略时保留原值。更新企业不会重设管理员账号或密码。
// @Tags 企业管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body domains.SysCompany{enabled=bool,adminUsername=string,adminPassword=string} true "企业资料；新建时必须同时提交管理员账号和加密密码"
// @Success 200 {object} response.Response{data=string} "保存成功，data 为企业 GUID"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /company [post]
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
		if err := services.InitializeGlobalRoles(tx); err != nil {
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
			if err := tx.Where("company_guid = '' AND code = ?", "COMPANY_ADMIN").First(&role).Error; err != nil {
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

// List 分页查询企业。
//
// @Summary 分页查询企业
// @Description 平台管理员可查询全部企业；企业用户只能查询所属企业，且不返回 apiKey、certificate。按创建顺序倒序返回，分页列表位于 data.data。
// @Tags 企业管理
// @Security ApiKeyAuth
// @Produce json
// @Param page query int false "页码，从 1 开始" default(1)
// @Param size query int false "每页条数，最多 100 条" default(10)
// @Param content query string false "模糊搜索：企业名称 name、编码 code"
// @Success 200 {object} response.Response{data=domains.PageResult{data=[]domains.SysCompany}} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /company/list [get]
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
	var companies []domains.SysCompany
	err := db.Scopes(q.Paginate()).Order("id DESC").Find(&companies).Error
	if !services.RequestAccess(c).PlatformAdmin() {
		for i := range companies {
			companies[i].ApiKey = ""
			companies[i].Certificate = ""
		}
	}
	accessReply(c, domains.PageResult{Data: companies, Total: n, Page: q.Page, Size: q.Size}, err)
}

// ListAll 查询企业选项列表。
//
// @Summary 查询企业选项列表
// @Description 平台管理员返回全部企业，企业用户仅返回所属企业。所有调用者均不返回 apiKey、certificate；按创建顺序升序返回，不分页。
// @Tags 企业管理
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} response.Response{data=[]domains.SysCompany} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /company/list/all [get]
func (CompanyApi) ListAll(c *gin.Context) {
	var companies []domains.SysCompany
	err := companyQuery(c).Order("id ASC").Find(&companies).Error
	for i := range companies {
		companies[i].ApiKey = ""
		companies[i].Certificate = ""
	}
	accessReply(c, companies, err)
}

// GetByGuid 查询企业详情。
//
// @Summary 查询企业详情
// @Description 平台管理员可查询任意企业；企业用户只能查询所属企业，且不返回 apiKey、certificate。
// @Tags 企业管理
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Success 200 {object} response.Response{data=domains.SysCompany} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /company/{guid} [get]
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

// Current 查询当前登录用户的企业摘要。
//
// @Summary 查询当前登录用户的企业摘要
// @Description 返回当前身份中的企业摘要 guid、name、logo、enabled。平台管理员不属于企业，data 为 null。仅需有效登录身份。
// @Tags 企业管理
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} response.Response{data=domains.CompanySummary} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Router /company/current [get]
func (CompanyApi) Current(c *gin.Context) {
	a := services.RequestAccess(c)
	if a == nil {
		response.NoAuth("未登录", c)
		return
	}
	response.Ok(a.User.Company, c)
}

// UpdateCurrent 修改当前企业资料。
//
// @Summary 修改当前企业资料
// @Description 仅所属企业的企业管理员可操作，平台管理员须通过企业管理接口修改。name 不能为空。
// @Description 仅写入 name、logo、phone、email、location、brief；字段省略时按空字符串写入，不修改企业编码、状态、到期时间或密钥。
// @Tags 企业管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body object{name=string,logo=string,phone=string,email=string,location=string,brief=string} true "企业资料：name 必填；其余字段为可编辑资料"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /company/current [put]
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

// DeleteByGuid 企业删除入口（已停用）。
//
// @Summary 企业删除入口（已停用）
// @Description 仅平台管理员可调用；当前始终拒绝删除并提示使用停用功能，保留企业及业务数据。停用请通过 POST /company 将 enabled 设为 false。
// @Tags 企业管理
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Failure 200 {object} response.Response "该接口始终返回业务失败，code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Deprecated
// @Router /company/{guid} [delete]
func (CompanyApi) DeleteByGuid(c *gin.Context) {
	if !platformOnly(c) {
		return
	}
	response.FailWithMessage("企业保留业务数据，请使用停用功能", c)
}

// Pass 审核通过企业。
//
// @Summary 审核通过企业
// @Description 仅平台管理员可操作，将目标企业 status 设为 1；不修改 enabled 或到期时间。
// @Tags 企业管理
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /company/pass/{guid} [put]
func (CompanyApi) Pass(c *gin.Context) {
	if !platformOnly(c) {
		return
	}
	err := global.NAV_DB.Model(&domains.SysCompany{}).Where("guid = ?", c.Param("guid")).UpdateColumn("status", 1).Error
	accessReply(c, true, err)
}

// Refuse 审核拒绝企业。
//
// @Summary 审核拒绝企业
// @Description 仅平台管理员可操作，将目标企业 status 设为 2；不删除企业数据，企业身份校验会拒绝该企业用户继续访问。
// @Tags 企业管理
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /company/refuse/{guid} [put]
func (CompanyApi) Refuse(c *gin.Context) {
	if !platformOnly(c) {
		return
	}
	err := global.NAV_DB.Model(&domains.SysCompany{}).Where("guid = ?", c.Param("guid")).UpdateColumn("status", 2).Error
	accessReply(c, true, err)
}
