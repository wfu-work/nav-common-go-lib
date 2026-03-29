package vos

type LoginVo struct {
	Username string `json:"username" description:"用户名"`
	Password string `json:"password" description:"密码"`
}

type TokenVo struct {
	Token        string `json:"token" description:"访问令牌"`
	RefreshToken string `json:"refreshToken" description:"刷新令牌"`
	TokenType    string `json:"tokenType" description:"令牌类型"`
	Expired      int64  `json:"expired" description:"过期时间"`
}

type SysCaptcha struct {
	CaptchaId     string `json:"captchaId"`
	CaptchaImg    string `json:"captchaImg"`
	CaptchaLength int    `json:"captchaLength"`
	OpenCaptcha   bool   `json:"openCaptcha"`
}

type PasswordVo struct {
	NewPassword string `json:"newPassword" description:"新密码"`
	OldPassword string `json:"oldPassword" description:"旧密码"`
}

type RegisterVo struct {
	Username string `json:"username" description:"用户名"`
	Password string `json:"password" description:"密码"`
	Email    string `json:"email" description:"邮箱"`
}
