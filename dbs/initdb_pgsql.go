package dbs

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"navfirst.com/nav-common-go-lib/configs"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/utils"
)

type PgsqlInitHandler struct{}

func NewPgsqlInitHandler() *PgsqlInitHandler {
	return &PgsqlInitHandler{}
}

// WriteConfig pgsql 回写配置
func (h PgsqlInitHandler) WriteConfig(ctx context.Context) error {
	c, ok := ctx.Value("config").(configs.Pgsql)
	if !ok {
		return errors.New("postgresql config invalid")
	}
	global.NAV_CONFIG.System.DbType = "pgsql"
	global.NAV_CONFIG.Pgsql = c
	global.NAV_CONFIG.JWT.SigningKey = uuid.New().String()
	cs := utils.StructToMap(global.NAV_CONFIG)
	for k, v := range cs {
		global.NAV_VIPER.Set(k, v)
	}
	global.NAV_ACTIVE_DBNAME = &c.Dbname
	return global.NAV_VIPER.WriteConfig()
}

// EnsureDB 创建数据库并初始化 pg
func (h PgsqlInitHandler) EnsureDB(ctx context.Context, conf *InitDB) (next context.Context, err error) {
	if s, ok := ctx.Value("dbtype").(string); !ok || s != "pgsql" {
		return ctx, ErrDBTypeMismatch
	}

	c := conf.ToPgsqlConfig()
	next = context.WithValue(ctx, "config", c)
	if c.Dbname == "" {
		return ctx, nil
	}

	dsn := conf.PgsqlEmptyDsn()
	var createSql string
	if conf.Template != "" {
		createSql = fmt.Sprintf("CREATE DATABASE %s WITH TEMPLATE %s;", c.Dbname, conf.Template)
	} else {
		createSql = fmt.Sprintf("CREATE DATABASE %s;", c.Dbname)
	}
	if exit, _ := checkDatabaseExists(dsn, "pgx", c.Dbname); !exit {
		if err = createDatabase(dsn, "pgx", createSql); err != nil {
			return nil, err
		}
	}

	var db *gorm.DB
	if db, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  c.Dsn(), // DSN data source name
		PreferSimpleProtocol: false,
	}), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true}); err != nil {
		return ctx, err
	}
	//global.NAV_CONFIG.AutoCode.Root, _ = filepath.Abs("..")
	next = context.WithValue(next, "db", db)
	return next, err
}

func (h PgsqlInitHandler) InitTables(ctx context.Context, inits InitSlice) error {
	return createTables(ctx, inits)
}

func (h PgsqlInitHandler) InitData(ctx context.Context, inits InitSlice) error {
	next, cancel := context.WithCancel(ctx)
	defer func(c func()) { c() }(cancel)
	for _, init := range inits {
		if init.DataInserted(next) {
			fmt.Printf(InitDataExist, "pgsql", init.InitializerName())
			continue
		}
		if n, err := init.InitializeData(next); err != nil {
			fmt.Printf(InitDataFailed, "pgsql", init.InitializerName(), err)
			return err
		} else {
			next = n
			fmt.Printf(InitDataSuccess, "pgsql", init.InitializerName())
		}
	}
	fmt.Printf(InitSuccess, "pgsql")
	return nil
}
