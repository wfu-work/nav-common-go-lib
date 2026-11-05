package domains

const BuiltinAdminGUID = "9a9ed1e6a5a334461b263e5a80207cc7"

func (u SysUser) PlatformAdmin() bool {
	return u.Guid == BuiltinAdminGUID && u.Username == "admin" && u.CompanyGuid == ""
}

type CompanySummary struct {
	Guid    string `json:"guid"`
	Name    string `json:"name"`
	Logo    string `json:"logo"`
	Enabled bool   `json:"enabled"`
}
