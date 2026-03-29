package dbs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrMissingDBContext        = errors.New("missing db in context")
	ErrMissingDependentContext = errors.New("missing dependent value in context")
	ErrDBTypeMismatch          = errors.New("db type mismatch")
)

const (
	InitSuccess     = "\n[%v] --> 初始数据成功!\n"
	InitDataExist   = "\n[%v] --> %v 的初始数据已存在!\n"
	InitDataFailed  = "\n[%v] --> %v 初始数据失败! \nerr: %+v\n"
	InitDataSuccess = "\n[%v] --> %v 初始数据成功!\n"
)

// SubInitializer 提供 source/*/init() 使用的接口，每个 initializer 完成一个初始化过程
type SubInitializer interface {
	InitializerName() string // 不一定代表单独一个表，所以改成了更宽泛的语义
	MigrateTable(ctx context.Context) (next context.Context, err error)
	InitializeData(ctx context.Context) (next context.Context, err error)
	TableCreated(ctx context.Context) bool
	DataInserted(ctx context.Context) bool
}

// TypedDBInitHandler 执行传入的 initializer
type TypedDBInitHandler interface {
	EnsureDB(ctx context.Context, conf *InitDB) (context.Context, error) // 建库，失败属于 fatal error，因此让它 panic
	WriteConfig(ctx context.Context) error                               // 回写配置
	InitTables(ctx context.Context, inits InitSlice) error               // 建表 handler
	InitData(ctx context.Context, inits InitSlice) error                 // 建数据 handler
}

// OrderedInitializer 组合一个顺序字段，以供排序
type OrderedInitializer struct {
	order int
	SubInitializer
}

// InitSlice 供 initializer 排序依赖时使用
type InitSlice []*OrderedInitializer

var (
	Initializers InitSlice
	Cache        map[string]*OrderedInitializer
)

// RegisterInit 注册要执行的初始化过程，会在 InitDB() 时调用
func RegisterInit(order int, i SubInitializer) {
	if Initializers == nil {
		Initializers = InitSlice{}
	}
	if Cache == nil {
		Cache = map[string]*OrderedInitializer{}
	}
	name := i.InitializerName()
	if _, existed := Cache[name]; existed {
		panic(fmt.Sprintf("Name conflict on %s", name))
	}
	ni := OrderedInitializer{order, i}
	Initializers = append(Initializers, &ni)
	Cache[name] = &ni
}

// checkDatabaseExists 检查数据库是否存在的函数
func checkDatabaseExists(dsn string, driver string, dbName string) (bool, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return false, err
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return false, err
	}
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1);"
	err = db.QueryRow(query, dbName).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// createDatabase 创建数据库（ EnsureDB() 中调用 ）
func createDatabase(dsn string, driver string, createSql string) error {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return err
	}
	defer func(db *sql.DB) {
		err = db.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(db)
	if err = db.Ping(); err != nil {
		return err
	}
	_, err = db.Exec(createSql)
	return err
}

// createTables 创建表（默认 dbInitHandler.initTables 行为）
func createTables(ctx context.Context, inits InitSlice) error {
	next, cancel := context.WithCancel(ctx)
	defer func(c func()) { c() }(cancel)
	for _, init := range inits {
		if init.TableCreated(next) {
			continue
		}
		if n, err := init.MigrateTable(next); err != nil {
			return err
		} else {
			next = n
		}
	}
	return nil
}

// Len /* -- sortable interface -- */
func (a InitSlice) Len() int {
	return len(a)
}

func (a InitSlice) Less(i, j int) bool {
	return a[i].order < a[j].order
}

func (a InitSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
