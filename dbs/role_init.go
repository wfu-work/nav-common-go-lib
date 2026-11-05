package dbs

import (
	"context"
	"errors"

	"fmt"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/gorm"
)

const initOrderRole = 10 + 1

type initRole struct{}

// auto run
func init() {
	RegisterInit(initOrderRole, &initRole{})
}

func (i *initRole) InitializerName() string {
	return domains.SysRole{}.TableName()
}

func (i *initRole) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&domains.SysRole{})
}

func (i *initRole) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&domains.SysRole{})
}

func (i *initRole) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("guid = ?", "9a9ed1e6a5a334461b263e5a80207cc6").
		First(&domains.SysRole{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}

func (i *initRole) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, ErrMissingDBContext
	}
	entities := []domains.SysRole{
		{BaseDataEntity: domains.BaseDataEntity{Guid: "9a9ed1e6a5a334461b263e5a80207cc6"}, Name: "超级管理员", Code: "SUPER_ADMIN", Sort: 1, Remark: "超级管理员"},
		{BaseDataEntity: domains.BaseDataEntity{Guid: "2a9ed1e6a5a334461b263e5a80207cc1"}, Name: "企业管理员", Code: "COMPANY_ADMIN", Sort: 2, Remark: "企业管理员"},
		{BaseDataEntity: domains.BaseDataEntity{Guid: "3a9ed1e6a5a334461b263e5a80207cc3"}, Name: "普通用户", Code: "USER", Sort: 3, Remark: "普通用户"},
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, fmt.Errorf("%s表数据初始化失败: %w", domains.SysRole{}.TableName(), err)
	}
	next := context.WithValue(ctx, i.InitializerName(), entities)
	return next, nil
}
