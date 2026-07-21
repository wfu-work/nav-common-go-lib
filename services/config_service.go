package services

import (
	"github.com/wfu-work/nav-common-go-lib/configs"
	"github.com/wfu-work/nav-common-go-lib/global"
)

type ConfigService struct {
}

var ConfigServiceApp = new(ConfigService)

func (s ConfigService) SaveConfigToFile() error {
	global.Lock.Lock()
	defer global.Lock.Unlock()

	v, configFile, err := configs.LoadConfigFile(global.NAV_VIPER)
	if err != nil {
		return err
	}
	return v.WriteConfigAs(configFile)
}

func (s ConfigService) SaveConfigValue(key string, value any) error {
	global.Lock.Lock()
	defer global.Lock.Unlock()

	return configs.WriteConfigValues(global.NAV_VIPER, map[string]any{key: value})
}

func (s ConfigService) GetConfigValue(key string) (any, error) {
	global.Lock.RLock()
	defer global.Lock.RUnlock()

	v, _, err := configs.LoadConfigFile(global.NAV_VIPER)
	if err != nil {
		return nil, err
	}
	return v.Get(key), nil
}
