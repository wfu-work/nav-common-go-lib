package dbs

import (
	"fmt"
	"os"

	"navfirst.com/nav-common-go-lib/configs"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/utils"
)

type InitDB struct {
	AdminPassword string `json:"adminPassword" binding:"required"`
	DBType        string `json:"dbType"`                    // 数据库类型
	Host          string `json:"host"`                      // 服务器地址
	Port          string `json:"port"`                      // 数据库连接端口
	UserName      string `json:"userName"`                  // 数据库用户名
	Password      string `json:"password"`                  // 数据库密码
	DBName        string `json:"dbName" binding:"required"` // 数据库名
	DBPath        string `json:"dbPath"`                    // sqlite数据库文件路径
	Template      string `json:"template"`                  // postgresql指定template
}

func GetInitDB() InitDB {
	switch global.NAV_CONFIG.System.DbType {
	case "sqlite":
		return InitDB{
			DBType: global.NAV_CONFIG.System.DbType,
			DBName: utils.StringOrDefault(global.NAV_CONFIG.Sqlite.Dbname, "nav-app"),
			DBPath: utils.StringOrDefault(global.NAV_CONFIG.Sqlite.Path, "./data/"),
		}
	case "mysql":
		return InitDB{
			DBType:   global.NAV_CONFIG.System.DbType,
			DBName:   utils.StringOrDefault(global.NAV_CONFIG.Mysql.Dbname, "nav-app"),
			DBPath:   utils.StringOrDefault(global.NAV_CONFIG.Mysql.Path, "./data/"),
			Host:     utils.StringOrDefault(global.NAV_CONFIG.Mysql.Host, utils.StringOrDefault(global.NAV_CONFIG.Mysql.Path, "127.0.0.1")),
			Port:     utils.StringOrDefault(global.NAV_CONFIG.Mysql.Port, "3306"),
			UserName: utils.StringOrDefault(global.NAV_CONFIG.Mysql.Username, "root"),
			Password: utils.StringOrDefault(global.NAV_CONFIG.Mysql.Password, "root"),
		}
	default:
		return InitDB{
			DBType:   global.NAV_CONFIG.System.DbType,
			DBName:   utils.StringOrDefault(global.NAV_CONFIG.Pgsql.Dbname, "nav-app"),
			DBPath:   utils.StringOrDefault(global.NAV_CONFIG.Pgsql.Path, "./data/"),
			Host:     utils.StringOrDefault(global.NAV_CONFIG.Pgsql.Host, utils.StringOrDefault(global.NAV_CONFIG.Pgsql.Path, "127.0.0.1")),
			Port:     utils.StringOrDefault(global.NAV_CONFIG.Pgsql.Port, "5432"),
			UserName: utils.StringOrDefault(global.NAV_CONFIG.Pgsql.Username, "postgres"),
			Password: utils.StringOrDefault(global.NAV_CONFIG.Pgsql.Password, "postgres"),
		}
	}
}

// SqliteEmptyDsn sqlite 空数据库 建库链接
func (i *InitDB) SqliteEmptyDsn() string {
	separator := string(os.PathSeparator)
	return i.DBPath + separator + i.DBName + ".db"
}

// MysqlEmptyDsn msyql 空数据库 建库链接
func (i *InitDB) MysqlEmptyDsn() string {
	if i.Host == "" {
		i.Host = "127.0.0.1"
	}
	if i.Port == "" {
		i.Port = "3306"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/", i.UserName, i.Password, i.Host, i.Port)
}

// PgsqlEmptyDsn pgsql 空数据库 建库链接
func (i *InitDB) PgsqlEmptyDsn() string {
	if i.Host == "" {
		i.Host = "127.0.0.1"
	}
	if i.Port == "" {
		i.Port = "5432"
	}
	return "host=" + i.Host + " user=" + i.UserName + " password=" + i.Password + " port=" + i.Port + " dbname=" + "postgres" + " " + "sslmode=disable TimeZone=Asia/Shanghai"
}

// ToSqliteConfig 转换 config.Sqlite
func (i *InitDB) ToSqliteConfig() configs.Sqlite {
	return configs.Sqlite{
		GeneralDB: configs.GeneralDB{
			Path:         i.DBPath,
			Port:         i.Port,
			Dbname:       i.DBName,
			Username:     i.UserName,
			Password:     i.Password,
			MaxIdleConns: 10,
			MaxOpenConns: 100,
			LogMode:      "error",
			Config:       "",
		},
	}
}

// ToMysqlConfig 转换 config.Mysql
func (i *InitDB) ToMysqlConfig() configs.Mysql {
	return configs.Mysql{
		GeneralDB: configs.GeneralDB{
			Host:         i.Host,
			Path:         i.Host,
			Port:         i.Port,
			Dbname:       i.DBName,
			Username:     i.UserName,
			Password:     i.Password,
			MaxIdleConns: 10,
			MaxOpenConns: 100,
			LogMode:      "error",
			Config:       "charset=utf8mb4&parseTime=True&loc=Local",
		},
	}
}

// ToPgsqlConfig 转换 config.Pgsql
func (i *InitDB) ToPgsqlConfig() configs.Pgsql {
	return configs.Pgsql{
		GeneralDB: configs.GeneralDB{
			Host:         i.Host,
			Path:         i.Host,
			Port:         i.Port,
			Dbname:       i.DBName,
			Username:     i.UserName,
			Password:     i.Password,
			MaxIdleConns: 10,
			MaxOpenConns: 100,
			LogMode:      "error",
			Config:       "sslmode=disable TimeZone=Asia/Shanghai",
		},
	}
}
