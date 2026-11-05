package dbs

import (
	"github.com/wfu-work/nav-common-go-lib/configs"
	"github.com/wfu-work/nav-common-go-lib/global"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// GormMysql 初始化Mysql数据库
func GormMysql() (*gorm.DB, error) {
	s := global.Config().Mysql
	return initMysqlDatabase(s)
}

// GormMysqlByConfig 初始化Mysql数据库用过传入配置
func GormMysqlByConfig(s configs.Mysql) (*gorm.DB, error) {
	return initMysqlDatabase(s)
}

// initSqliteDatabase 初始化Mysql数据库辅助函数
func initMysqlDatabase(s configs.Mysql) (*gorm.DB, error) {
	if s.Dbname == "" {
		return nil, nil
	}
	mysqlConfig := mysql.Config{
		DSN:                       s.Dsn(), // DSN data source name
		DefaultStringSize:         255,     // string 类型字段的默认长度
		SkipInitializeWithVersion: false,   // 根据版本自动配置
	}
	// 数据库配置
	general := s.GeneralDB
	if db, err := gorm.Open(mysql.New(mysqlConfig), Gorm.Config(general)); err != nil {
		return nil, err
	} else {
		db.InstanceSet("gorm:table_options", "ENGINE="+s.Engine)
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxIdleConns(s.MaxIdleConns)
		sqlDB.SetMaxOpenConns(s.MaxOpenConns)
		return db, nil
	}
}
