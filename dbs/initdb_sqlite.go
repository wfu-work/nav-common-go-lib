package dbs

import (
	"context"
	"errors"
	"fmt"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"navfirst.com/nav-common-go-lib/configs"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/utils"
)

type SqliteInitHandler struct{}

func NewSqliteInitHandler() *SqliteInitHandler {
	return &SqliteInitHandler{}
}

// WriteConfig mysql回写配置
func (h SqliteInitHandler) WriteConfig(ctx context.Context) error {
	c, ok := ctx.Value("config").(configs.Sqlite)
	if !ok {
		return errors.New("sqlite config invalid")
	}
	global.NAV_CONFIG.System.DbType = "sqlite"
	global.NAV_CONFIG.Sqlite = c
	global.NAV_CONFIG.JWT.SigningKey = uuid.New().String()
	cs := utils.StructToMap(global.NAV_CONFIG)
	for k, v := range cs {
		global.NAV_VIPER.Set(k, v)
	}
	global.NAV_ACTIVE_DBNAME = &c.Dbname
	return global.NAV_VIPER.WriteConfig()
}

// EnsureDB 创建数据库并初始化 sqlite
func (h SqliteInitHandler) EnsureDB(ctx context.Context, conf *InitDB) (next context.Context, err error) {
	if s, ok := ctx.Value("dbtype").(string); !ok || s != "sqlite" {
		return ctx, ErrDBTypeMismatch
	}

	c := conf.ToSqliteConfig()
	next = context.WithValue(ctx, "config", c)
	if c.Dbname == "" {
		return ctx, nil
	}

	dsn := conf.SqliteEmptyDsn()

	var db *gorm.DB
	if db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	}); err != nil {
		return ctx, err
	}
	//global.NAV_CONFIG.AutoCode.Root, _ = filepath.Abs("..")
	next = context.WithValue(next, "db", db)
	return next, err
}

func (h SqliteInitHandler) InitTables(ctx context.Context, inits InitSlice) error {
	return createTables(ctx, inits)
}

func (h SqliteInitHandler) InitData(ctx context.Context, inits InitSlice) error {
	next, cancel := context.WithCancel(ctx)
	defer func(c func()) { c() }(cancel)
	for _, init := range inits {
		if init.DataInserted(next) {
			fmt.Printf(InitDataExist, "sqlite", init.InitializerName())
			continue
		}
		if n, err := init.InitializeData(next); err != nil {
			fmt.Printf(InitDataFailed, "sqlite", init.InitializerName(), err)
			return err
		} else {
			next = n
			fmt.Printf(InitDataSuccess, "sqlite", init.InitializerName())
		}
	}
	fmt.Printf(InitSuccess, "sqlite")
	return nil
}
