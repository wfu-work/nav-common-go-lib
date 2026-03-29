package inits

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"navfirst.com/nav-common-go-lib/apis"
	"navfirst.com/nav-common-go-lib/dbs"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/middlewares"
	"navfirst.com/nav-common-go-lib/scheduleds"
	"navfirst.com/nav-common-go-lib/utils"
	"navfirst.com/nav-common-go-lib/utils/ip2geo"
)

func (s *SysInit) InitializeSystem() error {
	var err error
	global.NAV_VIPER, err = middlewares.Viper()
	if err != nil {
		return err
	}
	global.NAV_LOG = middlewares.Zap()
	zap.ReplaceGlobals(global.NAV_LOG)
	if global.NAV_CONFIG.JWT.SigningKey == "" {
		return errors.New("jwt.signing-key 未配置")
	}
	dbPath := utils.StringOrDefault(global.NAV_CONFIG.Sqlite.Path, "./data/")
	if b, _ := utils.PathExists(dbPath); !b {
		_ = utils.CreateDir(dbPath)
	}
	if err = dbs.InitRedis(); err != nil {
		return err
	}
	global.NAV_DB, err = dbs.InitGorm()
	if err != nil {
		return err
	}
	if global.NAV_DB != nil {
		if err = dbs.RegisterTables(); err != nil {
			return err
		}
		if s.onTableInit != nil {
			s.onTableInit()
		}
		if err = apis.DBService.InitDataBase(dbs.GetInitDB()); err != nil {
			return err
		}
	}
	s.Shed()
	if err = s.OtherInit(); err != nil {
		return err
	}
	return nil
}

func (s *SysInit) OtherInit() error {
	_, err := utils.ParseDuration(global.NAV_CONFIG.JWT.BufferTime)
	if err != nil {
		return err
	}
	file, err := os.Open("go.mod")
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Scan()
	}

	utils.InitCache()
	ip2geo.InitIpGeoLocation()

	if s.onOtherInit != nil {
		s.onOtherInit()
	}
	return nil
}

// Shed 定时任务
func (s *SysInit) Shed() {
	timers := scheduleds.NewTimerTask()
	go func() {
		var option []cron.Option
		option = append(option, cron.WithSeconds())
		_, _ = timers.AddTaskByFunc("ClearDB", "@daily", func() {
			err := scheduleds.ClearTable(s.onClearInit())
			if err != nil {
				fmt.Println("timer error:", err)
			}
		}, "每天定时清理数据库内容", option...)
		if s.onScheInit != nil {
			s.onScheInit(timers, option)
		}
	}()
}
