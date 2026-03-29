package dbs

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"navfirst.com/nav-common-go-lib/configs"
	"navfirst.com/nav-common-go-lib/global"
)

// GormSqlite 初始化Sqlite数据库
func GormSqlite() (*gorm.DB, error) {
	s := global.NAV_CONFIG.Sqlite
	return initSqliteDatabase(s)
}

// GormSqliteByConfig 初始化Sqlite数据库用过传入配置
func GormSqliteByConfig(s configs.Sqlite) (*gorm.DB, error) {
	return initSqliteDatabase(s)
}

// initSqliteDatabase 初始化Sqlite数据库辅助函数
func initSqliteDatabase(s configs.Sqlite) (*gorm.DB, error) {
	if s.Dbname == "" {
		return nil, nil
	}
	if db, err := gorm.Open(sqlite.Open(s.Dsn()), Gorm.Config(s.GeneralDB)); err != nil {
		return nil, err
	} else {
		db.Exec("PRAGMA busy_timeout = 5000;")
		sqlDB, _ := db.DB()
		_, _ = sqlDB.Exec("PRAGMA journal_mode = WAL;")
		sqlDB.SetMaxIdleConns(s.MaxIdleConns)
		sqlDB.SetMaxOpenConns(s.MaxOpenConns)
		return db, nil
	}
}
