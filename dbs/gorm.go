package dbs

import (
	"errors"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"navfirst.com/nav-common-go-lib/configs"
	"navfirst.com/nav-common-go-lib/domains"
	"navfirst.com/nav-common-go-lib/global"
)

var Gorm = new(_gorm)

type _gorm struct{}

func InitGorm() (*gorm.DB, error) {
	switch global.NAV_CONFIG.System.DbType {
	case "mysql":
		global.NAV_ACTIVE_DBNAME = &global.NAV_CONFIG.Mysql.Dbname
		return GormMysql()
	case "sqlite":
		global.NAV_ACTIVE_DBNAME = &global.NAV_CONFIG.Sqlite.Dbname
		return GormSqlite()
	case "pgsql":
		global.NAV_ACTIVE_DBNAME = &global.NAV_CONFIG.Pgsql.Dbname
		return GormPgSql()
	default:
		global.NAV_ACTIVE_DBNAME = &global.NAV_CONFIG.Mysql.Dbname
		return GormMysql()
	}
}

func RegisterTables() error {
	db := global.NAV_DB
	if db == nil {
		return errors.New("database is not initialized")
	}
	err := db.AutoMigrate(
		domains.SysRole{},
		domains.SysLog{},
		domains.SysUser{},
		domains.LoginLog{},
		domains.SysUserRole{},
		domains.SysCompany{},
	)
	if err != nil {
		global.NAV_LOG.Error("register table failed", zap.Error(err))
		return err
	}
	log.Println("register table success")
	return nil
}

func (g *_gorm) Config(general configs.GeneralDB) *gorm.Config {
	return &gorm.Config{
		Logger: logger.New(NewWriter(general), logger.Config{
			SlowThreshold: 200 * time.Millisecond,
			LogLevel:      general.LogLevel(),
			Colorful:      true,
		}),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   general.Prefix,
			SingularTable: general.Singular,
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	}
}

func (g *_gorm) ConfigSqlite(prefix string, singular bool) *gorm.Config {
	var general = global.NAV_CONFIG.Sqlite.GeneralDB
	return &gorm.Config{
		Logger: logger.New(NewWriter(general), logger.Config{
			SlowThreshold: 200 * time.Millisecond,
			LogLevel:      general.LogLevel(),
			Colorful:      true,
		}),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   prefix,
			SingularTable: singular,
		},
		NowFunc: func() time.Time {
			loc, _ := time.LoadLocation("Asia/Shanghai")
			return time.Now().In(loc)
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	}
}

type Writer struct {
	config configs.GeneralDB
	writer logger.Writer
}

func NewWriter(config configs.GeneralDB) *Writer {
	return &Writer{config: config}
}

func (c *Writer) Printf(message string, data ...any) {
	fmt.Printf(message, data...)
	if c.config.LogZap {
		switch c.config.LogLevel() {
		case logger.Silent:
			global.NAV_LOG.Debug(fmt.Sprintf(message, data...))
		case logger.Error:
			global.NAV_LOG.Error(fmt.Sprintf(message, data...))
		case logger.Warn:
			global.NAV_LOG.Warn(fmt.Sprintf(message, data...))
		case logger.Info:
			global.NAV_LOG.Info(fmt.Sprintf(message, data...))
		default:
			global.NAV_LOG.Info(fmt.Sprintf(message, data...))
		}
		return
	}
}
