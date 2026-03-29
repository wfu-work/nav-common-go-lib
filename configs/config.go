package configs

type Server struct {
	JWT      JWT            `mapstructure:"jwt" json:"jwt" yaml:"jwt"`
	Zap      Zap            `mapstructure:"zap" json:"zap" yaml:"zap"`
	System   System         `mapstructure:"system" json:"system" yaml:"system"`
	Captcha  Captcha        `mapstructure:"captcha" json:"captcha" yaml:"captcha"`
	Sqlite   Sqlite         `mapstructure:"sqlite" json:"sqlite" yaml:"sqlite"`
	Mysql    Mysql          `mapstructure:"mysql" json:"mysql" yaml:"mysql"`
	Pgsql    Pgsql          `mapstructure:"pgsql" json:"pgsql" yaml:"pgsql"`
	Redis    Redis          `mapstructure:"redis" json:"redis" yaml:"redis"`
	DiskList []DiskList     `mapstructure:"disk-list" json:"disk-list" yaml:"disk-list"`
	Cors     CORS           `mapstructure:"cors" json:"cors" yaml:"cors"`
	Local    Local          `mapstructure:"local" json:"local" yaml:"local"`
	Extras   map[string]any `json:"-" yaml:"-"`
}

func (s *Server) RegisterExtra(key string, value any) {
	if s.Extras == nil {
		s.Extras = make(map[string]any)
	}
	s.Extras[key] = value
}

func GetExtra[T any](s *Server, key string) (T, bool) {
	if val, ok := s.Extras[key]; ok {
		tVal, ok := val.(T)
		return tVal, ok
	}
	var zero T
	return zero, false
}
