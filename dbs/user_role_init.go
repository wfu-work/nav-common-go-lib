package dbs

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"navfirst.com/nav-common-go-lib/domains"
)

const initOrderUserRole = initOrderUser + 1

type initUserRole struct{}

// auto run
func init() {
	RegisterInit(initOrderUserRole, &initUserRole{})
}

func (i *initUserRole) InitializerName() string {
	return domains.SysUserRole{}.TableName()
}

func (i *initUserRole) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&domains.SysUserRole{})
}

func (i *initUserRole) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&domains.SysUserRole{})
}

func (i *initUserRole) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("user_guid = ? and role_guid = ?", "9a9ed1e6a5a334461b263e5a80207cc7", "9a9ed1e6a5a334461b263e5a80207cc6").
		First(&domains.SysUserRole{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}

func (i *initUserRole) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, ErrMissingDBContext
	}
	entities := []domains.SysUserRole{
		{RoleGuid: "9a9ed1e6a5a334461b263e5a80207cc6", UserGuid: "9a9ed1e6a5a334461b263e5a80207cc7"},
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!", domains.SysUserRole{}.TableName())
	}
	next := context.WithValue(ctx, i.InitializerName(), entities)
	return next, nil
}
