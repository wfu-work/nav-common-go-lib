package dbs

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/wfu-work/nav-common-go-lib/configs"
	"github.com/wfu-work/nav-common-go-lib/global"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// GormSqlite 初始化Sqlite数据库
func GormSqlite() (*gorm.DB, error) {
	s := global.Config().Sqlite
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
	dsn := s.Dsn()
	if dsn != ":memory:" && !strings.HasPrefix(dsn, "file:") {
		if err := os.MkdirAll(filepath.Dir(dsn), 0755); err != nil {
			return nil, err
		}
		absolute, err := filepath.Abs(dsn)
		if err != nil {
			return nil, err
		}
		dsn = (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}).String()
	}
	if dsn == ":memory:" {
		dsn = "file:nav-memory-" + uuid.NewString() + "?mode=memory&cache=shared"
	}
	base, query, _ := strings.Cut(dsn, "?")
	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, err
	}
	values.Set("_busy_timeout", "5000")
	values.Set("_journal_mode", "WAL")
	dsn = base + "?" + values.Encode()
	db, err := gorm.Open(sqlite.Open(dsn), Gorm.Config(s.GeneralDB))
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	// SQLite permits one writer. A single connection is the safe default.
	maxOpen := s.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 1
	}
	maxIdle := s.MaxIdleConns
	if maxIdle <= 0 || maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	pool.SetMaxOpenConns(maxOpen)
	pool.SetMaxIdleConns(maxIdle)
	var timeout int
	if err := db.Raw("PRAGMA busy_timeout").Scan(&timeout).Error; err != nil || timeout != 5000 {
		pool.Close()
		return nil, fmt.Errorf("sqlite busy_timeout initialization failed: %v", err)
	}
	return db, nil
}
