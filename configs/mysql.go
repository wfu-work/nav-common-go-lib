package configs

type Mysql struct {
	GeneralDB `yaml:",inline" mapstructure:",squash"`
}

func (s *Mysql) Dsn() string {
	host := s.Host
	if host == "" {
		host = s.Path
	}
	return s.Username + ":" + s.Password + "@tcp(" + host + ":" + s.Port + ")/" + s.Dbname + "?" + s.Config
}
