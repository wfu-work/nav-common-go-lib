package dbs

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"navfirst.com/nav-common-go-lib/configs"
	"navfirst.com/nav-common-go-lib/global"
)

// GormPgSql 初始化 Postgresql 数据库
func GormPgSql() (*gorm.DB, error) {
	p := global.NAV_CONFIG.Pgsql
	return initPgSqlDatabase(p)
}

// GormPgSqlByConfig 初始化 Postgresql 数据库 通过指定参数
func GormPgSqlByConfig(p configs.Pgsql) (*gorm.DB, error) {
	return initPgSqlDatabase(p)
}

// initPgSqlDatabase 初始化 Postgresql 数据库的辅助函数
func initPgSqlDatabase(p configs.Pgsql) (*gorm.DB, error) {
	if p.Dbname == "" {
		return nil, nil
	}
	pgsqlConfig := postgres.Config{
		DSN:                  p.Dsn(), // DSN data source name
		PreferSimpleProtocol: false,
	}
	// 数据库配置
	general := p.GeneralDB
	if db, err := gorm.Open(postgres.New(pgsqlConfig), Gorm.Config(general)); err != nil {
		return nil, err
	} else {
		sqlDB, _ := db.DB()
		sqlDB.SetMaxIdleConns(p.MaxIdleConns)
		sqlDB.SetMaxOpenConns(p.MaxOpenConns)
		return db, nil
	}
}
