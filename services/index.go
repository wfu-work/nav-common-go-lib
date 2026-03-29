package services

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"navfirst.com/nav-common-go-lib/domains"
	"navfirst.com/nav-common-go-lib/global"
	"navfirst.com/nav-common-go-lib/utils"
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
	CompanyService
	RegisterService
}

type HasBaseData interface {
	GetBaseData() domains.BaseDataEntity
}

type CrudService[T HasBaseData] struct {
}

func (s *CrudService[T]) Create(entity T) error {
	if entity.GetBaseData().Guid == "" {
		return global.NAV_DB.Create(&entity).Error
	}
	updateColumns, err := s.mutableColumns()
	if err != nil {
		return err
	}
	return global.NAV_DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "guid"}},
		DoUpdates: clause.AssignmentColumns(updateColumns),
	}).Create(&entity).Error
}

func (s *CrudService[T]) Updates(entity T) error {
	return global.NAV_DB.Model(&entity).Where("guid = ?", entity.GetBaseData().Guid).Updates(&entity).Error
}

func (s *CrudService[T]) Update(entity T, field string, value interface{}) error {
	return global.NAV_DB.Model(&entity).Where("guid = ?", entity.GetBaseData().Guid).Update(field, value).Error
}

func (s *CrudService[T]) DeleteByGuid(guid string) error {
	return global.NAV_DB.Where("guid = ?", guid).Delete(new(T)).Error
}

func (s *CrudService[T]) GetByGuid(guid string) (*T, error) {
	var result T
	err := global.NAV_DB.Where("guid = ?", guid).First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, err
}

func (s *CrudService[T]) GetById(id uint) (*T, error) {
	var result T
	err := global.NAV_DB.Where("id = ?", id).First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, err
}

func (s *CrudService[T]) GetByField(field string, value string) (*T, error) {
	column, err := s.resolveColumn(field)
	if err != nil {
		return nil, err
	}
	var result T
	err = global.NAV_DB.Where(column+" = ?", value).First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, err
}

func (s *CrudService[T]) List(pageInfo domains.PageInfo, likeFields string) (list interface{}, total int64, err error) {
	limit := pageInfo.Size
	offset := pageInfo.Size * (pageInfo.Page - 1)
	var result []T
	db := global.NAV_DB
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
	var results []T
	db := global.NAV_DB
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
	err := global.NAV_DB.First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func (s *CrudService[T]) SafeLast(result T) (*T, error) {
	err := global.NAV_DB.Last(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func (s *CrudService[T]) schemaColumns() (map[string]string, error) {
	stmt := &gorm.Statement{DB: global.NAV_DB}
	if err := stmt.Parse(new(T)); err != nil {
		return nil, err
	}
	columns := make(map[string]string, len(stmt.Schema.FieldsByDBName)*3)
	for _, field := range stmt.Schema.Fields {
		if field.DBName == "" {
			continue
		}
		columns[field.DBName] = field.DBName
		columns[field.Name] = field.DBName
		columns[strings.ToLower(field.Name)] = field.DBName
		columns[utils.CamelToSnake(field.Name)] = field.DBName
	}
	return columns, nil
}

func (s *CrudService[T]) resolveColumn(name string) (string, error) {
	columns, err := s.schemaColumns()
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("column name is empty")
	}
	if column, ok := columns[name]; ok {
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

func (s *CrudService[T]) mutableColumns() ([]string, error) {
	columns, err := s.schemaColumns()
	if err != nil {
		return nil, err
	}
	immutable := map[string]struct{}{
		"id":           {},
		"guid":         {},
		"create_time":  {},
		"deleted_time": {},
	}
	result := make([]string, 0, len(columns))
	seen := map[string]struct{}{}
	for _, column := range columns {
		if _, skip := immutable[column]; skip {
			continue
		}
		if _, exists := seen[column]; exists {
			continue
		}
		seen[column] = struct{}{}
		result = append(result, column)
	}
	return result, nil
}
