package services

import (
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

type OsService struct {
}

var OsServiceApp = new(OsService)

func (s OsService) GetServerInfo() (server *utils.Server, err error) {
	var result utils.Server
	result.Os = utils.InitOS()
	if result.Cpu, err = utils.InitCPU(); err != nil {
		global.NAV_LOG.Error("func utils.InitCPU() Failed", zap.String("err", err.Error()))
		return &result, err
	}
	if result.Ram, err = utils.InitRAM(); err != nil {
		global.NAV_LOG.Error("func utils.InitRAM() Failed", zap.String("err", err.Error()))
		return &result, err
	}
	if result.Disk, err = utils.InitDisk(); err != nil {
		global.NAV_LOG.Error("func utils.InitDisk() Failed", zap.String("err", err.Error()))
		return &result, err
	}
	return &result, nil

}
