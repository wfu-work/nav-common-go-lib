package apis

import (
	"errors"

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
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			err = errors.New("账号或企业标识已存在")
		}
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(v, c)
}

// Save 创建或更新企业用户。
//
// @Summary 创建或更新企业用户
// @Description 仅平台管理员或企业管理员可操作。guid 为空表示新建；平台管理员必须指定 companyGuid，企业管理员只能管理所属企业，已有用户不能转移企业。
// @Description username 必填并全局唯一，自动去除首尾空格并转为小写，admin 为保留账号。新建必须提供经 POST /secret/encrypt 加密的 password，明文至少 8 位；更新省略 password 时不改密码。
// @Description enable 为 0 或 1，省略时新建默认 1、更新保留原值。roleGuidList 省略时保留角色，传空数组时清空，仅能分配本企业有效角色，且不能移除最后一位有效企业管理员。
// @Description 更新会提升账号会话版本，使旧令牌失效；内置 admin 不允许通过此接口修改。
// @Tags 用户管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body services.UserSaveRequest true "用户资料、加密密码及可选角色 GUID 列表"
// @Success 200 {object} response.Response{data=string} "保存成功，data 为用户 GUID"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /user [post]
func (UserApi) Save(c *gin.Context) {
	var req services.UserSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		accessReply(c, nil, err)
		return
	}
	id, err := services.SaveManagedUser(services.RequestAccess(c), req)
	accessReply(c, id, err)
}

// DeleteByGuid 删除企业用户。
//
// @Summary 删除企业用户
// @Description 仅平台管理员或企业管理员可操作，企业管理员只能删除所属企业用户。软删除用户并移除角色关系；内置 admin 和企业最后一位有效管理员不能删除。用户名仍保留，不可复用。
// @Tags 用户管理
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /user/{guid} [delete]
func (UserApi) DeleteByGuid(c *gin.Context) {
	accessReply(c, true, services.DeleteManagedUser(services.RequestAccess(c), c.Param("guid")))
}

// GetByGuid 查询用户详情。
//
// @Summary 查询用户详情
// @Description 按企业范围查询用户，并补充 roleGuidList、roleCodeList、roleNameList；密码和盐不会序列化返回。平台管理员可通过 companyGuid 进一步限定企业。
// @Tags 用户管理
// @Security ApiKeyAuth
// @Produce json
// @Param guid path string true "记录 GUID"
// @Param companyGuid query string false "仅平台管理员可选：限定目标企业；企业用户始终限定当前企业"
// @Success 200 {object} response.Response{data=domains.SysUser} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /user/{guid} [get]
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
	users := []domains.SysUser{*u}
	if populateUsersRoles(global.NAV_DB, users) == nil {
		*u = users[0]
	}
}

func populateUsersRoles(db *gorm.DB, users []domains.SysUser) error {
	guids := make([]string, 0, len(users))
	byGuid := make(map[string]*domains.SysUser, len(users))
	for i := range users {
		user := &users[i]
		user.RoleGuidList, user.RoleCodeList = []string{}, []string{}
		user.RoleNameList = []string{}
		if user.PlatformAdmin() {
			user.RoleNameList = []string{"平台超级管理员"}
		}
		guids = append(guids, user.Guid)
		byGuid[user.Guid] = user
	}
	if len(guids) == 0 {
		return nil
	}
	var assigned []struct {
		domains.SysRole
		AssignedUserGuid string
	}
	err := db.Model(&domains.SysRole{}).Select("sys_role.*, ur.user_guid AS assigned_user_guid").
		Joins("JOIN sys_user_role ur ON ur.role_guid = sys_role.guid").
		Joins("JOIN sys_user u ON u.guid = ur.user_guid AND u.company_guid = sys_role.company_guid AND u.deleted_time IS NULL").
		Where("ur.user_guid IN ?", guids).Scan(&assigned).Error
	if err != nil {
		return err
	}
	for _, item := range assigned {
		if user := byGuid[item.AssignedUserGuid]; user != nil {
			user.RoleGuidList = append(user.RoleGuidList, item.Guid)
			user.RoleCodeList = append(user.RoleCodeList, item.Code)
			user.RoleNameList = append(user.RoleNameList, item.Name)
		}
	}
	return nil
}

// List 分页查询用户。
//
// @Summary 分页查询用户
// @Description 企业用户仅查询所属企业，平台管理员可查询全部或指定企业。按创建顺序倒序返回，批量补充用户角色 GUID、编码与名称；不支持自定义排序。
// @Tags 用户管理
// @Security ApiKeyAuth
// @Produce json
// @Param page query int false "页码，从 1 开始" default(1)
// @Param size query int false "每页条数，最多 100 条" default(10)
// @Param content query string false "模糊搜索：用户名 username、昵称 nickName、手机号 phone"
// @Param companyGuid query string false "仅平台管理员可选：限定目标企业；企业用户始终限定当前企业"
// @Success 200 {object} response.Response{data=domains.PageResult{data=[]domains.SysUser}} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /user/list [get]
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
	if err := populateUsersRoles(global.NAV_DB.WithContext(c.Request.Context()), users); err != nil {
		accessReply(c, nil, err)
		return
	}
	for i := range users {
		users[i].IsPlatformAdmin = users[i].PlatformAdmin()
	}
	response.Ok(domains.PageResult{Data: users, Total: n, Page: q.Page, Size: q.Size}, c)
}

// GetUserByToken 获取当前登录用户与实时权限。
//
// @Summary 获取当前登录用户与实时权限
// @Description 返回当前身份校验后的用户资料、企业摘要、角色 GUID/编码、abilityList、isPlatformAdmin、isCompanyAdmin 和 dataScope。仅需有效登录身份，不以 JWT 中的历史角色作为授权依据。
// @Tags 用户管理
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} response.Response{data=domains.SysUser} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Router /user/token [get]
func (UserApi) GetUserByToken(c *gin.Context) {
	a := services.RequestAccess(c)
	if a == nil {
		response.NoAuth("未登录", c)
		return
	}
	response.Ok(a.User, c)
}

// UpdatePassword 修改当前登录用户密码。
//
// @Summary 修改当前登录用户密码
// @Description 仅修改当前用户。oldPassword、newPassword 均为经 POST /secret/encrypt 加密的字符串；校验旧密码，新密码明文至少 8 位。成功后旧会话失效，需要重新登录。
// @Tags 用户管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param data body vos.PasswordVo true "加密后的旧密码 oldPassword 与新密码 newPassword"
// @Success 200 {object} response.Response{data=bool} "成功时 code=200；业务失败时 code=400，原因见 msg"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Router /user/update/password [put]
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

// SetRoles 替换用户角色分配。
//
// @Summary 替换用户角色分配
// @Description 仅平台管理员或企业管理员可操作，企业管理员仅管理所属企业用户。roleGuids 为完整目标列表，传空数组或省略会清空角色。
// @Description 仅允许本企业有效角色，禁止修改内置 admin 的平台身份，且须保留最后一位有效企业管理员；成功后用户旧会话失效。
// @Description 请求示例：{"roleGuids":["角色GUID"]}。
// @Tags 用户管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param guid path string true "记录 GUID"
// @Param data body object{roleGuids=[]string} true "角色分配对象，roleGuids 为完整角色 GUID 列表"
// @Success 200 {object} response.Response{data=string} "保存成功，data 为用户 GUID"
// @Failure 401 {object} response.Response "未登录、令牌过期或会话已失效"
// @Failure 403 {object} response.Response "无接口权限或不满足所需管理员身份"
// @Router /user/{guid}/roles [put]
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
