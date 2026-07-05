package services

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var SysServiceGroupApp = new(SysServiceGroup)

type SysServiceGroup struct {
	LoggerService
	UserService
	RoleService
	DBService
	OsService
	ConfigService
	FileService
	LoginService
	LoginLogService
	UserRoleService
	PermissionService
	CompanyService
	RegisterService
}

type HasBaseData interface {
	GetBaseData() domains.BaseDataEntity
}

type CrudService[T HasBaseData] struct {
	db *gorm.DB
}

type crudSchemaMeta struct {
	columns        map[string]string
	mutableColumns []string
}

var crudSchemaCache sync.Map

func (s *CrudService[T]) WithDB(db *gorm.DB) *CrudService[T] {
	cloned := *s
	cloned.db = db
	return &cloned
}

func (s *CrudService[T]) DB() *gorm.DB {
	if s != nil && s.db != nil {
		return s.db
	}
	return global.NAV_DB
}

func (s *CrudService[T]) Create(entity T) error {
	db := s.DB()
	if db == nil {
		return errors.New("database is not initialized")
	}
	if entity.GetBaseData().Guid == "" {
		return db.Create(&entity).Error
	}
	meta, err := s.schemaMeta()
	if err != nil {
		return err
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "guid"}},
		DoUpdates: clause.AssignmentColumns(meta.mutableColumns),
	}).Create(&entity).Error
}

func (s *CrudService[T]) Updates(entity T) error {
	db := s.DB()
	if db == nil {
		return errors.New("database is not initialized")
	}
	return db.Model(&entity).Where("guid = ?", entity.GetBaseData().Guid).Updates(&entity).Error
}

func (s *CrudService[T]) Update(entity T, field string, value interface{}) error {
	db := s.DB()
	if db == nil {
		return errors.New("database is not initialized")
	}
	column, err := s.resolveColumn(field)
	if err != nil {
		return err
	}
	return db.Model(&entity).Where("guid = ?", entity.GetBaseData().Guid).Update(column, value).Error
}

func (s *CrudService[T]) DeleteByGuid(guid string) error {
	db := s.DB()
	if db == nil {
		return errors.New("database is not initialized")
	}
	return db.Where("guid = ?", guid).Delete(new(T)).Error
}

func (s *CrudService[T]) GetByGuid(guid string) (*T, error) {
	db := s.DB()
	if db == nil {
		return nil, errors.New("database is not initialized")
	}
	var result T
	err := db.Where("guid = ?", guid).First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, err
}

func (s *CrudService[T]) GetById(id uint) (*T, error) {
	db := s.DB()
	if db == nil {
		return nil, errors.New("database is not initialized")
	}
	var result T
	err := db.Where("id = ?", id).First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, err
}

func (s *CrudService[T]) GetByField(field string, value string) (*T, error) {
	db := s.DB()
	if db == nil {
		return nil, errors.New("database is not initialized")
	}
	column, err := s.resolveColumn(field)
	if err != nil {
		return nil, err
	}
	var result T
	err = db.Where(column+" = ?", value).First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, err
}

func (s *CrudService[T]) List(pageInfo domains.PageInfo, likeFields string) (list interface{}, total int64, err error) {
	db := s.DB()
	if db == nil {
		err = errors.New("database is not initialized")
		return
	}
	pageInfo = normalizePageInfo(pageInfo)
	limit := pageInfo.Size
	offset := pageInfo.Size * (pageInfo.Page - 1)
	var result []T
	if pageInfo.Content != "" && likeFields != "" {
		likeFieldSlice, resolveErr := s.resolveColumns(strings.Split(likeFields, ","))
		if resolveErr != nil {
			err = resolveErr
			return
		}
		for index, field := range likeFieldSlice {
			if index == 0 {
				db = db.Where(field+" like ?", "%"+pageInfo.Content+"%")
			} else {
				db = db.Or(field+" like ?", "%"+pageInfo.Content+"%")
			}
		}
	}
	db = db.Model(new(T))
	err = db.Count(&total).Error
	if err != nil {
		return
	}
	order := "id desc"
	if pageInfo.Asc != "" {
		column, resolveErr := s.resolveColumn(pageInfo.Asc)
		if resolveErr != nil {
			err = resolveErr
			return
		}
		order = column + " asc"
	}
	if pageInfo.Desc != "" {
		column, resolveErr := s.resolveColumn(pageInfo.Desc)
		if resolveErr != nil {
			err = resolveErr
			return
		}
		order = column + " desc"
	}
	err = db.Order(order).Limit(limit).Offset(offset).Find(&result).Error
	return result, total, err
}

func (s *CrudService[T]) ListAll(params map[string]string) ([]T, error) {
	normalized := make(map[string]any, len(params))
	for key, value := range params {
		normalized[key] = value
	}
	return s.ListByFields(normalized)
}

func (s *CrudService[T]) ListByFields(params map[string]any) ([]T, error) {
	db := s.DB()
	if db == nil {
		return nil, errors.New("database is not initialized")
	}
	var results []T
	for key, value := range params {
		column, err := s.resolveColumn(key)
		if err != nil {
			return nil, err
		}
		db = db.Where(column+" = ?", value)
	}
	err := db.Find(&results).Error
	return results, err
}

func (s *CrudService[T]) SafeFirst(result T) (*T, error) {
	db := s.DB()
	if db == nil {
		return nil, errors.New("database is not initialized")
	}
	err := db.First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func (s *CrudService[T]) SafeLast(result T) (*T, error) {
	db := s.DB()
	if db == nil {
		return nil, errors.New("database is not initialized")
	}
	err := db.Last(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func (s *CrudService[T]) schemaMeta() (*crudSchemaMeta, error) {
	db := s.DB()
	if db == nil {
		return nil, errors.New("database is not initialized")
	}
	schemaType := reflect.TypeOf((*T)(nil)).Elem()
	if cached, ok := crudSchemaCache.Load(schemaType); ok {
		return cached.(*crudSchemaMeta), nil
	}
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(new(T)); err != nil {
		return nil, err
	}
	columns := make(map[string]string, len(stmt.Schema.FieldsByDBName)*3)
	immutable := map[string]struct{}{
		"id":           {},
		"guid":         {},
		"create_time":  {},
		"deleted_time": {},
		"creater":      {},
	}
	mutableColumns := make([]string, 0, len(stmt.Schema.FieldsByDBName))
	seenMutable := map[string]struct{}{}
	for _, field := range stmt.Schema.Fields {
		if field.DBName == "" {
			continue
		}
		columns[field.DBName] = field.DBName
		columns[field.Name] = field.DBName
		columns[strings.ToLower(field.Name)] = field.DBName
		columns[utils.CamelToSnake(field.Name)] = field.DBName
		if _, skip := immutable[field.DBName]; skip {
			continue
		}
		if _, exists := seenMutable[field.DBName]; exists {
			continue
		}
		seenMutable[field.DBName] = struct{}{}
		mutableColumns = append(mutableColumns, field.DBName)
	}
	meta := &crudSchemaMeta{
		columns:        columns,
		mutableColumns: mutableColumns,
	}
	actual, _ := crudSchemaCache.LoadOrStore(schemaType, meta)
	return actual.(*crudSchemaMeta), nil
}

func (s *CrudService[T]) resolveColumn(name string) (string, error) {
	meta, err := s.schemaMeta()
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("column name is empty")
	}
	if column, ok := meta.columns[name]; ok {
		return column, nil
	}
	return "", fmt.Errorf("unsupported column: %s", name)
}

func (s *CrudService[T]) resolveColumns(names []string) ([]string, error) {
	columns := make([]string, 0, len(names))
	for _, name := range names {
		column, err := s.resolveColumn(name)
		if err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}
	return columns, nil
}

func normalizePageInfo(pageInfo domains.PageInfo) domains.PageInfo {
	if pageInfo.Page <= 0 {
		pageInfo.Page = 1
	}
	switch {
	case pageInfo.Size > 100:
		pageInfo.Size = 100
	case pageInfo.Size <= 0:
		pageInfo.Size = 10
	}
	return pageInfo
}

func WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	db := global.NAV_DB
	if db == nil {
		return errors.New("database is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return db.WithContext(ctx).Transaction(fn)
}
