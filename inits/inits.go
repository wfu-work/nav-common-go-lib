package inits

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/wfu-work/nav-common-go-lib/scheduleds"
)

type SysInit struct {
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

func (s *SysInit) InitE() error {
	// 初始化系统
	if err := s.InitializeSystem(); err != nil {
		return err
	}
	// 运行服务器
	return s.RunServer()
}
