package middlewares

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/wfu-work/nav-common-go-lib/global"
)

const (
	ConfigEnv         = "NAV_CONFIG"
	ConfigCommonFile  = "config-common.yaml"
	ConfigDefaultFile = "config.yaml"
	ConfigTestFile    = "config.test.yaml"
	ConfigDebugFile   = "config.debug.yaml"
	ConfigReleaseFile = "config.release.yaml"
)

//go:embed config-common.yaml
var configYaml embed.FS

var initFlags sync.Once

// Viper 配置
func Viper() (*viper.Viper, error) {
	configBytes, err := configYaml.ReadFile("config-common.yaml")
	if err != nil {
		return nil, fmt.Errorf("读取基础配置失败: %w", err)
	}
	base := viper.New()
	base.SetConfigType("yaml")
	if err = base.ReadConfig(bytes.NewReader(configBytes)); err != nil {
		return nil, fmt.Errorf("读取基础配置失败: %w", err)
	}

	config := GetConfigPath()
	v := viper.New()
	v.SetConfigFile(config)
	v.SetConfigType("yaml")
	err = v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("读取业务配置失败: %w", err)
	}

	for _, key := range base.AllKeys() {
		if !v.IsSet(key) {
			v.Set(key, base.Get(key))
		}
	}

	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("config file changed: %v\n", e)
		if err = v.Unmarshal(&global.NAV_CONFIG); err != nil {
			fmt.Println(err)
			return
		}
	})
	if err = v.Unmarshal(&global.NAV_CONFIG); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	return v, nil
}

// GetConfigPath 获取配置文件路径, 优先级: 命令行 > 环境变量 > 默认值
func GetConfigPath() (config string) {
	initFlags.Do(func() {
		flag.StringVar(&config, "c", "", "choose config file.")
		flag.Parse()
	})
	if config != "" {
		fmt.Printf("您正在使用命令行的 '-c' 参数传递的值, config 的路径为 %s\n", config)
		return
	}
	if env := os.Getenv(ConfigEnv); env != "" {
		config = env
		fmt.Printf("您正在使用 %s 环境变量, config 的路径为 %s\n", ConfigEnv, config)
		return
	}

	switch gin.Mode() {
	case gin.DebugMode:
		config = ConfigDebugFile
	case gin.ReleaseMode:
		config = ConfigReleaseFile
	case gin.TestMode:
		config = ConfigTestFile
	}
	fmt.Printf("您正在使用 gin 的 %s 模式运行, config 的路径为 %s\n", gin.Mode(), config)

	_, err := os.Stat(config)
	if err != nil || os.IsNotExist(err) {
		config = ConfigDefaultFile
		fmt.Printf("配置文件路径不存在, 使用默认配置文件路径: %s\n", config)
	}

	return
}
