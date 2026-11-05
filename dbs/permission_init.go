package dbs

import (
	"context"
	"errors"

	"fmt"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	initOrderPermission = initOrderRole + 1

	PermissionGuidAllAPI = "sys-permission-all-api"
)

type initPermission struct{}

// auto run
func init() {
	RegisterInit(initOrderPermission, &initPermission{})
}

func (i *initPermission) InitializerName() string {
	return domains.SysPermission{}.TableName()
}

func (i *initPermission) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&domains.SysPermission{})
}

func (i *initPermission) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&domains.SysPermission{})
}

func (i *initPermission) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("guid = ?", PermissionGuidAllAPI).
		First(&domains.SysPermission{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}

func (i *initPermission) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, ErrMissingDBContext
	}
	entities := []domains.SysPermission{
		{
			BaseDataEntity: domains.BaseDataEntity{Guid: PermissionGuidAllAPI},
			Name:           "全部API权限",
			Code:           "system.api.all",
			Type:           domains.PermissionTypeAPI,
			Path:           "/*",
			Method:         "*",
			Enabled:        true,
			Sort:           1,
			Remark:         "超级管理员默认全量API权限",
		},
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&entities).Error; err != nil {
		return ctx, fmt.Errorf("%s表数据初始化失败: %w", domains.SysPermission{}.TableName(), err)
	}
	next := context.WithValue(ctx, i.InitializerName(), entities)
	return next, nil
}
