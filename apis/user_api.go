package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	"github.com/wfu-work/nav-common-go-lib/services"
	"github.com/wfu-work/nav-common-go-lib/vos"
	"gorm.io/gorm"
)

type UserApi struct{}

func accessReply(c *gin.Context, v any, err error) {
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(v, c)
}
func (UserApi) Save(c *gin.Context) {
	var req services.UserSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		accessReply(c, nil, err)
		return
	}
	id, err := services.SaveManagedUser(services.RequestAccess(c), req)
	accessReply(c, id, err)
}
func (UserApi) DeleteByGuid(c *gin.Context) {
	accessReply(c, true, services.DeleteManagedUser(services.RequestAccess(c), c.Param("guid")))
}
func (UserApi) GetByGuid(c *gin.Context) {
	var u domains.SysUser
	db := services.AccessDB(c, &u)
	if err := db.Where("guid = ?", c.Param("guid")).First(&u).Error; err != nil {
		accessReply(c, nil, err)
		return
	}
	populateUserRoles(&u)
	response.Ok(u, c)
}
func populateUserRoles(u *domains.SysUser) {
	var roles []domains.SysRole
	global.NAV_DB.Model(&domains.SysRole{}).Joins("JOIN sys_user_role ur ON ur.role_guid=sys_role.guid").Where("ur.user_guid = ? AND sys_role.company_guid = ?", u.Guid, u.CompanyGuid).Find(&roles)
	u.RoleGuidList = []string{}
	u.RoleCodeList = []string{}
	for _, r := range roles {
		u.RoleGuidList = append(u.RoleGuidList, r.Guid)
		u.RoleCodeList = append(u.RoleCodeList, r.Code)
	}
}
func (UserApi) List(c *gin.Context) {
	var q domains.PageInfo
	if err := c.ShouldBindQuery(&q); err != nil {
		accessReply(c, nil, err)
		return
	}
	db := services.AccessDB(c, &domains.SysUser{})
	if q.Content != "" {
		db = db.Where("(username LIKE ? OR nick_name LIKE ? OR phone LIKE ?)", "%"+q.Content+"%", "%"+q.Content+"%", "%"+q.Content+"%")
	}
	var n int64
	if err := db.Count(&n).Error; err != nil {
		accessReply(c, nil, err)
		return
	}
	var users []domains.SysUser
	if err := db.Scopes(q.Paginate()).Order("id DESC").Find(&users).Error; err != nil {
		accessReply(c, nil, err)
		return
	}
	for i := range users {
		populateUserRoles(&users[i])
		users[i].IsPlatformAdmin = users[i].PlatformAdmin()
	}
	response.Ok(domains.PageResult{Data: users, Total: n, Page: q.Page, Size: q.Size}, c)
}
func (UserApi) GetUserByToken(c *gin.Context) {
	a := services.RequestAccess(c)
	if a == nil {
		response.NoAuth("未登录", c)
		return
	}
	response.Ok(a.User, c)
}
func (UserApi) UpdatePassword(c *gin.Context) {
	a := services.RequestAccess(c)
	if a == nil {
		response.NoAuth("未登录", c)
		return
	}
	var req vos.PasswordVo
	if err := c.ShouldBindJSON(&req); err != nil {
		accessReply(c, nil, err)
		return
	}
	ok, err := services.UserServiceApp.UpdatePassword(a.User.Guid, &req)
	if err == nil {
		err = global.NAV_DB.Model(&domains.SysUser{}).Where("guid = ?", a.User.Guid).UpdateColumn("auth_version", gorm.Expr("auth_version + 1")).Error
	}
	accessReply(c, ok, err)
}
func (UserApi) SetRoles(c *gin.Context) {
	var req struct {
		RoleGuids []string `json:"roleGuids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		accessReply(c, nil, err)
		return
	}
	a := services.RequestAccess(c)
	var u domains.SysUser
	if err := services.AccessDB(c, &u).Where("guid = ?", c.Param("guid")).First(&u).Error; err != nil {
		accessReply(c, nil, err)
		return
	}
	enabled := u.Enable
	id, err := services.SaveManagedUser(a, services.UserSaveRequest{Guid: u.Guid, CompanyGuid: u.CompanyGuid, Username: u.Username, NickName: u.NickName, Email: u.Email, Phone: u.Phone, Enable: &enabled, RoleGuidList: &req.RoleGuids})
	accessReply(c, id, err)
}
