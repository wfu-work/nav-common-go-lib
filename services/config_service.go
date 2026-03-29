package services

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"github.com/wfu-work/nav-common-go-lib/global"
)

type ConfigService struct {
}

var ConfigServiceApp = new(ConfigService)

func (s ConfigService) SaveConfigToFile() error {
	settings := global.NAV_VIPER.AllSettings()
	file, err := os.Create("config.yaml")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := viper.ConfigFileUsed()
	if encoder == "" {
		encoder = "config.yaml"
	}
	v := viper.New()
	for k, val := range settings {
		v.Set(k, convertMapNumbers(val))
	}
	v.SetConfigType("yaml")
	return v.WriteConfigAs(encoder)
}

// 递归处理 map[string]interface{} 中的所有字符串，能转成 int 就转成 int
func convertMapNumbers(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		for key, val := range v {
			v[key] = convertMapNumbers(val)
		}
		return v
	case []interface{}:
		for i, val := range v {
			v[i] = convertMapNumbers(val)
		}
		return v
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
		if isSafeFloatString(v) {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
		return v
	default:
		return v
	}
}

// 判断是否是精度在 float64 可接受范围的安全小数
func isSafeFloatString(s string) bool {
	if strings.Contains(s, ".") {
		if len(s) <= 18 {
			return true
		}
	}
	return false
}

func writeMapAsYaml(f *os.File, prefix string, data map[string]interface{}) error {
	for k, v := range data {
		keyLine := fmt.Sprintf("%s%s:", prefix, k)
		switch val := v.(type) {
		case map[string]interface{}:
			if _, err := fmt.Fprintln(f, keyLine); err != nil {
				return err
			}
			if err := writeMapAsYaml(f, prefix+"  ", val); err != nil {
				return err
			}
		case []interface{}:
			if _, err := fmt.Fprintf(f, "%s # unsupported array\n", keyLine); err != nil {
				return err
			}
		default:
			strVal := fmt.Sprint(val)
			if _, err := fmt.Fprintf(f, "%s %s\n", keyLine, strVal); err != nil {
				return err
			}
		}
	}
	return nil
}
