package dbs

import (
	"context"

	"github.com/pkg/errors"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const initOrderRolePermission = initOrderUserRole + 1

type initRolePermission struct{}

// auto run
func init() {
	RegisterInit(initOrderRolePermission, &initRolePermission{})
}

func (i *initRolePermission) InitializerName() string {
	return domains.SysRolePermission{}.TableName()
}

func (i *initRolePermission) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&domains.SysRolePermission{})
}

func (i *initRolePermission) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&domains.SysRolePermission{})
}

func (i *initRolePermission) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("role_guid = ? AND permission_guid = ?", "9a9ed1e6a5a334461b263e5a80207cc6", PermissionGuidAllAPI).
		First(&domains.SysRolePermission{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}

func (i *initRolePermission) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, ErrMissingDBContext
	}
	entities := []domains.SysRolePermission{
		{
			RoleGuid:       "9a9ed1e6a5a334461b263e5a80207cc6",
			PermissionGuid: PermissionGuidAllAPI,
		},
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&entities).Error; err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!", domains.SysRolePermission{}.TableName())
	}
	next := context.WithValue(ctx, i.InitializerName(), entities)
	return next, nil
}
