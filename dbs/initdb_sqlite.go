package dbs

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/wfu-work/nav-common-go-lib/configs"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/utils"
)

type SqliteInitHandler struct{}

func NewSqliteInitHandler() *SqliteInitHandler {
	return &SqliteInitHandler{}
}

// WriteConfig mysql回写配置
func (h SqliteInitHandler) WriteConfig(ctx context.Context) error {
	global.Lock.Lock()
	defer global.Lock.Unlock()
	next := global.Config().Clone()

	c, ok := ctx.Value("config").(configs.Sqlite)
	if !ok {
		return errors.New("sqlite config invalid")
	}
	next.System.DbType = "sqlite"
	next.Sqlite = c
	next.JWT.SigningKey = uuid.New().String()
	cs := utils.StructToMap(next)
	global.NAV_ACTIVE_DBNAME = &c.Dbname
	return configs.WriteConfigValues(global.NAV_VIPER, cs)
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

	db, err := initSqliteDatabase(c)
	if err != nil {
		return ctx, err
	}

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
