package inits

import (
	"context"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"log"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/wfu-work/nav-common-go-lib/scheduleds"
)

type SysInit struct {
	auditStop   func()
	watcherStop func()
	background  context.Context
	cancel      context.CancelFunc
	cacheStop   func()
	loggerClose func() error
	timers      scheduleds.Timer
	db          *gorm.DB
	redis       redis.UniversalClient
	closeOnce   sync.Once
	closeErr    error
	initialized bool

	onWebInit    OnWebFunc
	onScheInit   OnScheFunc
	onOtherInit  OnOtherFunc
	onCacheInit  OnCacheFunc
	onRouterInit OnRouterFunc
	onTableInit  OnTableFunc
	onShutInit   OnShutFunc
	onClearInit  OnClearFunc
}

type OnWebFunc func(router *gin.Engine)
type OnScheFunc func(timer scheduleds.Timer, options []cron.Option)
type OnRouterFunc func(publicGroup *gin.RouterGroup, privateGroup *gin.RouterGroup)
type OnOtherFunc func()
type OnCacheFunc func()
type OnTableFunc func()
type OnClearFunc func() []scheduleds.ClearDB
type OnShutFunc func()

func (s *SysInit) OnWebInit(f OnWebFunc) {
	s.onWebInit = f
}

func (s *SysInit) OnScheInit(f OnScheFunc) {
	s.onScheInit = f
}

func (s *SysInit) OnRouterInit(f OnRouterFunc) {
	s.onRouterInit = f
}

func (s *SysInit) OnTableInit(f OnTableFunc) {
	s.onTableInit = f
}

func (s *SysInit) OnCacheInit(f OnCacheFunc) { s.onCacheInit = f }

func (s *SysInit) OnOtherInit(f OnOtherFunc) {
	s.onOtherInit = f
}

func (s *SysInit) OnShutInit(f OnShutFunc) {
	s.onShutInit = f
}

func (s *SysInit) OnClearInit(f OnClearFunc) {
	s.onClearInit = f
}

func (s *SysInit) Init() {
	if err := s.InitE(); err != nil {
		log.Printf("system init failed: %v\n", err)
	}
}

func (s *SysInit) InitE() error { return s.InitContext(context.Background()) }

func (s *SysInit) InitContext(ctx context.Context) error {
	if err := s.initializeSystem(ctx); err != nil {
		return err
	}
	return s.RunServer()
}
