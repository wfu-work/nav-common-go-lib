package middlewares

import (
	"bytes"
	"context"
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/wfu-work/nav-common-go-lib/configs"
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
var configFlag string

// Viper 配置
func Viper() (*viper.Viper, error) { return ViperWithContext(context.Background()) }

func ViperWithContext(ctx context.Context) (*viper.Viper, error) {
	v, _, err := LoadConfig(ctx)
	return v, err
}

// LoadConfig provides an explicit stop-and-wait function for the file watcher.
func LoadConfig(parent context.Context) (*viper.Viper, func(), error) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	v, done, err := watchViper(ctx)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	return v, func() { cancel(); <-done }, nil
}

func watchViper(ctx context.Context) (*viper.Viper, <-chan struct{}, error) {
	configBytes, err := configYaml.ReadFile("config-common.yaml")
	if err != nil {
		return nil, nil, fmt.Errorf("读取基础配置失败: %w", err)
	}
	base := viper.New()
	base.SetConfigType("yaml")
	if err = base.ReadConfig(bytes.NewReader(configBytes)); err != nil {
		return nil, nil, fmt.Errorf("读取基础配置失败: %w", err)
	}

	config := GetConfigPath()
	v := viper.New()
	v.SetConfigFile(config)
	v.SetConfigType("yaml")
	err = v.ReadInConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("读取业务配置失败: %w", err)
	}

	defaults := make(map[string]any, len(base.AllKeys()))
	for _, key := range base.AllKeys() {
		defaults[key] = base.Get(key)
	}
	configs.SetDefaults(defaults)
	configs.ApplyDefaults(v)

	if err = publishServerConfig(v, false); err != nil {
		return nil, nil, fmt.Errorf("解析配置失败: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}
	configPath, err := filepath.Abs(config)
	if err != nil {
		watcher.Close()
		return nil, nil, err
	}
	if err := watcher.Add(filepath.Dir(configPath)); err != nil {
		watcher.Close()
		return nil, nil, err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer watcher.Close()
		var timer *time.Timer
		var pending <-chan time.Time
		defer func() {
			if timer != nil {
				timer.Stop()
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				path, _ := filepath.Abs(event.Name)
				if path == configPath && event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
					if timer != nil {
						timer.Stop()
					}
					timer = time.NewTimer(100 * time.Millisecond)
					pending = timer.C
				}
			case <-pending:
				pending = nil
				if ctx.Err() != nil {
					return
				}
				isolated, _, err := configs.LoadConfigFile(v)
				if err == nil {
					err = publishServerConfig(isolated, true)
				}
				if err != nil {
					fmt.Printf("reload config failed: %v\n", err)
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return v, done, nil
}

func reloadServerConfig(v *viper.Viper) error { return publishServerConfig(v, true) }

func publishServerConfig(v *viper.Viper, reload bool) error {
	var next configs.Server
	if err := v.Unmarshal(&next); err != nil {
		return err
	}
	previous := global.Config()
	next.Extras = previous.Extras
	if reload {
		if err := next.ValidateReload(*previous); err != nil {
			return err
		}
	} else if err := next.Validate(); err != nil {
		return err
	}
	if !reload {
		global.Lock.Lock()
		global.NAV_CONFIG = next.Clone()
		global.Lock.Unlock()
	}
	global.SetConfig(next)
	return nil
}

// GetConfigPath 获取配置文件路径, 优先级: 命令行 > 环境变量 > 默认值
func GetConfigPath() (config string) {
	initFlags.Do(func() {
		if flag.Lookup("c") == nil {
			flag.StringVar(&configFlag, "c", "", "choose config file.")
		}
		if !flag.Parsed() {
			flag.Parse()
		}
		configFlag = flag.Lookup("c").Value.String()
	})
	config = configFlag
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
