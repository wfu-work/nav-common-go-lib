package inits

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/wfu-work/nav-common-go-lib/apis"
	"github.com/wfu-work/nav-common-go-lib/dbs"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
	"github.com/wfu-work/nav-common-go-lib/scheduleds"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"github.com/wfu-work/nav-common-go-lib/utils/ip2geo"
	"go.uber.org/zap"
)

func (s *SysInit) InitializeSystem() error { return s.initializeSystem(context.Background()) }

func (s *SysInit) initializeSystem(parent context.Context) (err error) {
	if s.initialized {
		return errors.New("system is already initialized; create a new SysInit")
	}
	s.initialized = true
	if parent == nil {
		parent = context.Background()
	}
	s.background, s.cancel = context.WithCancel(parent)
	defer func() {
		if err != nil {
			err = errors.Join(err, s.Close())
		}
	}()
	global.NAV_VIPER, s.watcherStop, err = middlewares.LoadConfig(s.background)
	if err != nil {
		return err
	}
	global.NAV_LOG, s.loggerClose, err = middlewares.ZapWithConfig(global.Config().Zap)
	if err != nil {
		return err
	}
	zap.ReplaceGlobals(global.NAV_LOG)
	if err = dbs.InitRedis(); err != nil {
		return err
	}
	s.redis = global.NAV_REDIS
	global.NAV_DB, err = dbs.InitGorm()
	if err != nil {
		return err
	}
	s.db = global.NAV_DB
	if s.db != nil {
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
	s.auditStop = middlewares.StartAuditWriter(s.db, global.NAV_LOG, 1024)
	if err = s.OtherInit(); err != nil {
		return err
	}
	return s.startScheduled()
}

func (s *SysInit) OtherInit() error {
	if s.background == nil {
		s.background = context.Background()
	}
	s.cacheStop = utils.InitCacheContext(s.background)
	if s.onCacheInit != nil {
		s.onCacheInit()
	}
	if err := ip2geo.InitIpGeoLocation(); err != nil {
		return err
	}
	if s.onOtherInit != nil {
		s.onOtherInit()
	}
	return nil
}

// Shed is retained for hosts that register tasks explicitly.
func (s *SysInit) Shed() {
	if err := s.startScheduled(); err != nil {
		zap.L().Error("initialize scheduled tasks", zap.Error(err))
	}
}

func (s *SysInit) startScheduled() error {
	if s.timers != nil {
		return nil
	}
	s.timers = scheduleds.NewTimerTask()
	options := []cron.Option{cron.WithSeconds(), cron.WithChain(cron.Recover(cron.DefaultLogger), cron.SkipIfStillRunning(cron.DefaultLogger))}
	_, err := s.timers.AddTaskByFunc("ClearDB", "@daily", func() {
		var tables []scheduleds.ClearDB
		if s.onClearInit != nil {
			tables = s.onClearInit()
		}
		if err := scheduleds.ClearTable(tables); err != nil {
			zap.L().Error("clear database", zap.Error(err))
		}
	}, "每天定时清理数据库内容", options...)
	if err != nil {
		return err
	}
	if s.onScheInit != nil {
		s.onScheInit(s.timers, options)
	}
	return nil
}

// Close releases application-owned resources after HTTP requests have drained.
// It is safe to call repeatedly, including after partial initialization failure.
func (s *SysInit) Close() error {
	s.closeOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		if s.watcherStop != nil {
			s.watcherStop()
		}
		if s.timers != nil {
			if timer, ok := s.timers.(interface{ CloseContext(context.Context) error }); ok {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				s.closeErr = errors.Join(s.closeErr, timer.CloseContext(ctx))
				cancel()
			} else {
				s.timers.Close()
			}
		}
		if s.cacheStop != nil {
			s.cacheStop()
			s.closeErr = errors.Join(s.closeErr, utils.SaveRtLogging())
		}
		if s.auditStop != nil {
			s.auditStop()
		}
		if s.onShutInit != nil {
			s.onShutInit()
		}
		ip2geo.CloseIpGeoLocation()
		if s.redis != nil {
			s.closeErr = errors.Join(s.closeErr, s.redis.Close())
		}
		if s.db != nil {
			pool, err := s.db.DB()
			if err == nil {
				err = pool.Close()
			}
			s.closeErr = errors.Join(s.closeErr, err)
		}
		if s.loggerClose != nil {
			s.closeErr = errors.Join(s.closeErr, s.loggerClose())
		}
	})
	if s.closeErr != nil {
		return fmt.Errorf("close system resources: %w", s.closeErr)
	}
	return nil
}
