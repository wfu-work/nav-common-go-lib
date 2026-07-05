package dbs

import (
	"context"

	"github.com/pkg/errors"
	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"gorm.io/gorm"
)

const initOrderUser = initOrderPermission + 1

type initUser struct{}

// auto run
func init() {
	RegisterInit(initOrderUser, &initUser{})
}

func (i *initUser) InitializerName() string {
	return domains.SysUser{}.TableName()
}

func (i *initUser) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&domains.SysUser{})
}

func (i *initUser) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&domains.SysUser{})
}

func (i *initUser) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("guid = ?", "9a9ed1e6a5a334461b263e5a80207cc7").
		First(&domains.SysUser{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}

func (i *initUser) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, ErrMissingDBContext
	}
	adminPassword := utils.BcryptHash("navfirst.com")
	entities := []domains.SysUser{
		{BaseDataEntity: domains.BaseDataEntity{Guid: "9a9ed1e6a5a334461b263e5a80207cc7"}, Username: "admin", Password: adminPassword, Phone: "15207129575", Email: "support@shoumtech.com", Enable: 1},
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!", domains.SysUser{}.TableName())
	}
	next := context.WithValue(ctx, i.InitializerName(), entities)
	return next, nil
}
