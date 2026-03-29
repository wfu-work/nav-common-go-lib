package scheduleds

import (
	"errors"
	"fmt"
	"time"

	"github.com/wfu-work/nav-common-go-lib/global"
)

type ClearDB struct {
	TableName    string
	CompareField string
	Interval     string
}

// ClearTable
// @description: 清理数据库表数据
// @param: db(数据库对象) *gorm.DB, tableName(表名) string, compareField(比较字段) string, interval(间隔) string
// @return: error
func ClearTable(dbs []ClearDB) error {
	var ClearTableDetail []ClearDB

	ClearTableDetail = append(ClearTableDetail, ClearDB{
		TableName:    "sys_log",
		CompareField: "create_time",
		Interval:     "720h",
	})
	ClearTableDetail = append(ClearTableDetail, ClearDB{
		TableName:    "sys_login_log",
		CompareField: "create_time",
		Interval:     "720h",
	})
	if dbs != nil && len(dbs) == 0 {
		ClearTableDetail = append(ClearTableDetail, dbs...)
	}

	if global.NAV_DB == nil {
		return errors.New("db Cannot be empty")
	}

	for _, detail := range ClearTableDetail {
		duration, err := time.ParseDuration(detail.Interval)
		if err != nil {
			return err
		}
		if duration < 0 {
			return errors.New("parse duration < 0")
		}
		err = global.NAV_DB.Debug().Exec(fmt.Sprintf("DELETE FROM %s WHERE %s < ?", detail.TableName, detail.CompareField), time.Now().Add(-duration).UnixMilli()).Error
		if err != nil {
			return err
		}
	}
	return nil
}
