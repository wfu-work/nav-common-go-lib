package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

var (
	configDefaultsMu sync.RWMutex
	configDefaults   = make(map[string]any)
)

// SetDefaults replaces the defaults shared by the configuration loaders.
func SetDefaults(defaults map[string]any) {
	configDefaultsMu.Lock()
	defer configDefaultsMu.Unlock()

	configDefaults = make(map[string]any, len(defaults))
	for key, value := range defaults {
		configDefaults[key] = value
	}
}

// ApplyDefaults applies the shared defaults without overriding values from a
// configuration file.
func ApplyDefaults(v *viper.Viper) {
	configDefaultsMu.RLock()
	defer configDefaultsMu.RUnlock()

	for key, value := range configDefaults {
		v.SetDefault(key, value)
	}
}

// LoadConfigFile creates an isolated Viper instance for the file used by
// source. The isolated instance prevents file reads and writes from racing
// with source's config watcher.
func LoadConfigFile(source *viper.Viper) (*viper.Viper, string, error) {
	if source == nil {
		return nil, "", fmt.Errorf("viper 配置实例未初始化")
	}
	configFile := source.ConfigFileUsed()
	if configFile == "" {
		return nil, "", fmt.Errorf("当前 Viper 实例没有关联配置文件")
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, "", fmt.Errorf("读取配置文件失败: %w", err)
	}
	ApplyDefaults(v)
	return v, configFile, nil
}

// WriteConfigValues updates the file used by source without mutating source.
func WriteConfigValues(source *viper.Viper, values map[string]any) error {
	v, configFile, err := LoadConfigFile(source)
	if err != nil {
		return err
	}
	for key, value := range values {
		v.Set(key, value)
	}
	var next Server
	if err := v.Unmarshal(&next); err != nil {
		return err
	}
	if err := next.Validate(); err != nil {
		return err
	}
	return SaveConfigFile(v, configFile)
}

// SaveConfigFile replaces the file atomically so watchers never see partial YAML.
func SaveConfigFile(v *viper.Viper, path string) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".nav-config-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	mode := os.FileMode(0600)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	if err := file.Chmod(mode); err != nil {
		return err
	}
	if err := v.WriteConfigTo(file); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
