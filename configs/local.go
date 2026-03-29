package configs

type Local struct {
	ProxyOssPath string `mapstructure:"proxy-oss-path" json:"proxy-oss-path" yaml:"proxy-oss-path"` // 本地文件访问路径
	OssPath      string `mapstructure:"oss-path" json:"oss-path" yaml:"oss-path"`                   // 本地文件存储路径
	CachePath    string `mapstructure:"cache-path" json:"cache-path" yaml:"cache-path"`             // 缓存文件存储路径
	Ip2geoPath   string `mapstructure:"ip2geo-path" json:"ip2geo-path" yaml:"ip2geo-path"`          // ip2geo文件存储路径
}
