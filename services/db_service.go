package services

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/wfu-work/nav-common-go-lib/dbs"
	"github.com/wfu-work/nav-common-go-lib/global"
	"gorm.io/gorm"
)

type DBService struct{}

var (
	ErrMissingDBContext        = errors.New("missing db in context")
	ErrMissingDependentContext = errors.New("missing dependent value in context")
	ErrDBTypeMismatch          = errors.New("db type mismatch")
)

// InitDataBase 创建数据库并初始化 总入口
func (dBService *DBService) InitDataBase(conf dbs.InitDB) (err error) {
	ctx := context.TODO()
	ctx = context.WithValue(ctx, "adminPassword", conf.AdminPassword)
	if len(dbs.Initializers) == 0 {
		return nil
	}
	sort.Sort(&dbs.Initializers) // 保证有依赖的 initializer 排在后面执行
	// Note: 若 initializer 只有单一依赖，可以写为 B=A+1, C=A+1; 由于 BC 之间没有依赖关系，所以谁先谁后并不影响初始化
	// 若存在多个依赖，可以写为 C=A+B, D=A+B+C, E=A+1;
	// C必然>A|B，因此在AB之后执行，D必然>A|B|C，因此在ABC后执行，而E只依赖A，顺序与CD无关，因此E与CD哪个先执行并不影响
	var initHandler dbs.TypedDBInitHandler
	switch strings.ToLower(conf.DBType) {
	case "sqlite":
		initHandler = dbs.NewSqliteInitHandler()
		ctx = context.WithValue(ctx, "dbtype", "sqlite")
	case "mysql":
		initHandler = dbs.NewMysqlInitHandler()
		ctx = context.WithValue(ctx, "dbtype", "mysql")
	case "pgsql":
		initHandler = dbs.NewPgsqlInitHandler()
		ctx = context.WithValue(ctx, "dbtype", "pgsql")
	default:
		initHandler = dbs.NewMysqlInitHandler()
		ctx = context.WithValue(ctx, "dbtype", "mysql")
	}
	db := global.NAV_DB
	if db == nil {
		ctx, err = initHandler.EnsureDB(ctx, &conf)
		if err != nil {
			return err
		}
		var ok bool
		db, ok = ctx.Value("db").(*gorm.DB)
		if !ok || db == nil {
			return ErrMissingDBContext
		}
		global.NAV_DB = db
	} else {
		ctx = context.WithValue(ctx, "db", db)
	}

	if err = initHandler.InitTables(ctx, dbs.Initializers); err != nil {
		return err
	}
	if err = initHandler.InitData(ctx, dbs.Initializers); err != nil {
		return err
	}

	return nil
}
