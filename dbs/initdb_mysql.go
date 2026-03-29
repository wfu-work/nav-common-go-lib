package dbs

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"navfirst.com/nav-common-go-lib/configs"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/utils"
)

type MysqlInitHandler struct{}

func NewMysqlInitHandler() *MysqlInitHandler {
	return &MysqlInitHandler{}
}

// WriteConfig mysql回写配置
func (h MysqlInitHandler) WriteConfig(ctx context.Context) error {
	c, ok := ctx.Value("config").(configs.Mysql)
	if !ok {
		return errors.New("mysql config invalid")
	}
	global.NAV_CONFIG.System.DbType = "mysql"
	global.NAV_CONFIG.Mysql = c
	global.NAV_CONFIG.JWT.SigningKey = uuid.New().String()
	cs := utils.StructToMap(global.NAV_CONFIG)
	for k, v := range cs {
		global.NAV_VIPER.Set(k, v)
	}
	global.NAV_ACTIVE_DBNAME = &c.Dbname
	return global.NAV_VIPER.WriteConfig()
}

// EnsureDB 创建数据库并初始化 mysql
func (h MysqlInitHandler) EnsureDB(ctx context.Context, conf *InitDB) (next context.Context, err error) {
	if s, ok := ctx.Value("dbtype").(string); !ok || s != "mysql" {
		return ctx, ErrDBTypeMismatch
	}

	c := conf.ToMysqlConfig()
	next = context.WithValue(ctx, "config", c)
	if c.Dbname == "" {
		return ctx, nil
	}

	dsn := conf.MysqlEmptyDsn()
	createSql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 DEFAULT COLLATE utf8mb4_general_ci;", c.Dbname)
	if exit, _ := checkDatabaseExists(dsn, "mysql", c.Dbname); !exit {
		if err = createDatabase(dsn, "mysql", createSql); err != nil {
			return nil, err
		}
	}

	var db *gorm.DB
	if db, err = gorm.Open(mysql.New(mysql.Config{
		DSN:                       c.Dsn(), // DSN data source name
		DefaultStringSize:         191,     // string 类型字段的默认长度
		SkipInitializeWithVersion: true,    // 根据版本自动配置
	}), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true}); err != nil {
		return ctx, err
	}
	//global.NAV_CONFIG.AutoCode.Root, _ = filepath.Abs("..")
	next = context.WithValue(next, "db", db)
	return next, err
}

func (h MysqlInitHandler) InitTables(ctx context.Context, inits InitSlice) error {
	return createTables(ctx, inits)
}

func (h MysqlInitHandler) InitData(ctx context.Context, inits InitSlice) error {
	next, cancel := context.WithCancel(ctx)
	defer func(c func()) { c() }(cancel)
	for _, init := range inits {
		if init.DataInserted(next) {
			fmt.Printf(InitDataExist, "mysql", init.InitializerName())
			continue
		}
		if n, err := init.InitializeData(next); err != nil {
			fmt.Printf(InitDataFailed, "mysql", init.InitializerName(), err)
			return err
		} else {
			next = n
			fmt.Printf(InitDataSuccess, "mysql", init.InitializerName())
		}
	}
	fmt.Printf(InitSuccess, "mysql")
	return nil
}
