package db_gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func ConnectDB(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	return db, err
}

func FindByField(db *gorm.DB, fieldName string, fieldValue interface{}, doc interface{}, dest ...interface{}) (bool, error) {
	dst := utils.OptionalArg(doc, dest...)
	result := db.Model(doc).First(dst, fmt.Sprintf("\"%v\" = ?", fieldName), fieldValue)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}

	return true, nil
}

func FindByFields(db *gorm.DB, fields db.Fields, doc interface{}, dest ...interface{}) (bool, error) {
	dst := utils.OptionalArg(doc, dest...)
	result := db.Model(doc).Where(fields).First(dst)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}

	return true, nil
}

func FindForUpdate(db *gorm.DB, fields db.Fields, doc interface{}) (bool, error) {
	result := db.Model(doc).Clauses(clause.Locking{Strength: "UPDATE"}).Where(fields).First(doc)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

func FindForShare(db *gorm.DB, fields db.Fields, doc interface{}) (bool, error) {
	result := db.Model(doc).Clauses(clause.Locking{Strength: "SHARE"}).Where(fields).First(doc)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

func RowsByFields(db *gorm.DB, fields db.Fields, doc interface{}) (*sql.Rows, error) {
	return db.Model(doc).Where(fields).Rows()
}

func AllRows(db *gorm.DB, doc interface{}) (*sql.Rows, error) {
	return db.Model(doc).Rows()
}

func FindAll(db *gorm.DB, docs interface{}, dest ...interface{}) error {
	dst := utils.OptionalArg(docs, dest...)
	result := db.Model(docs).Find(dst)
	return result.Error
}

type Interval = db.Interval
type Filter = db.Filter

func compareOp(isOpen bool, comparator string) string {
	if isOpen {
		return comparator
	}
	return utils.ConcatStrings(comparator, "=")
}

func prepareInterval(db *gorm.DB, name string, interval *Interval) *gorm.DB {
	h := db

	if interval.From != nil && interval.To != nil {
		if interval.From == interval.To {
			h = h.Where(fmt.Sprintf("\"%v\" = ?", name), interval.From)
		} else {
			h = h.Where(fmt.Sprintf("\"%v\" %s ? AND \"%v\" %s ? ", name, compareOp(interval.FromOpen, ">"), name, compareOp(interval.ToOpen, "<")), interval.From, interval.To)
		}
	} else if interval.From != nil {
		h = h.Where(fmt.Sprintf("\"%v\" %s ? ", name, compareOp(interval.FromOpen, ">")), interval.From)
	} else if interval.To != nil {
		h = h.Where(fmt.Sprintf("\"%v\" %s ? ", name, compareOp(interval.ToOpen, "<")), interval.To)
	}
	return h
}

func prepareFilter(db *gorm.DB, filter *Filter) *gorm.DB {
	h := db

	if filter.Fields != nil {
		h = h.Where(filter.Fields)
	}

	for _, f := range filter.PresetFields {
		h = h.Where(f)
	}

	for field, values := range filter.FieldsIn {
		h = h.Where(fmt.Sprintf("\"%v\" IN ? ", field), values)
	}

	for field, values := range filter.FieldsNotIn {
		h = h.Where(fmt.Sprintf("\"%v\" NOT IN ? ", field), values)
	}

	for name, interval := range filter.Intervals {
		h = prepareInterval(h, name, interval)
	}

	for _, between := range filter.BetweenFields {
		h = h.Where(fmt.Sprintf("? %s \"%v\" AND ? %s \"%v\"", compareOp(between.FromOpen, ">"), between.FromField, compareOp(between.ToOpen, ">"), between.ToField), between.Value, between.Value)
	}

	for _, orFields := range filter.OrFields {
		m := map[string]interface{}{}
		for i, field := range orFields.Fields {
			m[field] = orFields.Value
			if i == 0 {
				h = h.Where(m)
			} else {
				h = h.Or(m)
			}
			delete(m, field)
		}
	}

	return h
}

func SetFilter(g *gorm.DB, filter *Filter, paginator *Paginator, docs interface{}, paginate ...bool) *gorm.DB {

	if filter == nil {
		return g
	}

	h := g
	if docs != nil {
		h = g.Model(docs)
	}

	h = prepareFilter(h, filter)

	if filter.SortField != "" && (filter.SortDirection == db.SORT_ASC || filter.SortDirection == db.SORT_DESC) {
		parts := strings.Split(filter.SortField, ".")
		if len(parts) == 2 {
			h = h.Order(fmt.Sprintf("\"%s\".\"%s\" %s", parts[0], parts[1], filter.SortDirection))
		} else {
			h = h.Order(fmt.Sprintf("\"%s\" %s", filter.SortField, filter.SortDirection))
		}
	}

	if paginator != nil {
		h = paginator.Paginate(h, filter, paginate...)
	}

	return h
}

type Paginator struct {
	MaxLimit int
}

func (p *Paginator) Paginate(g *gorm.DB, filter *Filter, paginate ...bool) *gorm.DB {
	h := g
	if utils.OptionalArg(true, paginate...) {
		if filter.Offset > 0 {
			h = h.Offset(filter.Offset)
		}

		if filter.Limit > 0 {
			limit := filter.Limit
			if !filter.NoLimit && limit > p.MaxLimit && p.MaxLimit > 0 {
				limit = p.MaxLimit
			}
			h = h.Limit(limit)
		} else if !filter.NoLimit && p.MaxLimit > 0 {
			h = h.Limit(p.MaxLimit)
		}
	}
	return h
}

func find(g *gorm.DB, filter *Filter, paginator *Paginator, dest interface{}) (int64, error) {

	var count int64

	h := g
	if filter != nil {
		h = SetFilter(g, filter, paginator, nil, !filter.Count)
		if filter.Count {
			counter := h.Session(&gorm.Session{})
			result := counter.Count(&count)
			if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return 0, result.Error
			}
			h = paginator.Paginate(h, filter)
		}
	}

	result := h.Find(dest)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return 0, result.Error
	}
	if result.RowsAffected > count {
		count = result.RowsAffected
	}

	return count, nil
}

func FindWithFilter(db *gorm.DB, filter *Filter, paginator *Paginator, docs interface{}, dest ...interface{}) (int64, error) {
	g := db.Model(docs)
	dst := utils.OptionalArg(docs, dest...)
	return find(g, filter, paginator, dst)
}

func RowsWithFilter(db *gorm.DB, filter *Filter, paginator *Paginator, docs interface{}) (*sql.Rows, error) {

	h := db.Model(docs)
	h = SetFilter(h, filter, paginator, docs)

	rows, err := h.Rows()
	if err != nil {
		return nil, err
	}
	return rows, err
}

func CountWithFilter(db *gorm.DB, filter *Filter, doc interface{}) int64 {

	m := db.Model(doc)
	h := prepareFilter(m, filter)

	var count int64
	h.Count(&count)
	return count
}

func SumWithFilter(db *gorm.DB, filter *Filter, fields map[string]string, doc interface{}, result interface{}) error {

	sums := ""
	for key, name := range fields {
		if sums != "" {
			sums += ", "
		}
		sums += fmt.Sprintf("sum(%v) as %v", key, name)
	}

	m := db.Model(doc).Select(sums)
	h := prepareFilter(m, filter)

	r := h.Take(result)
	return r.Error
}

func FindAllByFields(db *gorm.DB, fields db.Fields, docs interface{}) error {
	result := db.Where(fields).Find(docs)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func FindNotIn(db *gorm.DB, fields db.Fields, docs interface{}) error {
	result := db.Not(fields).Find(docs)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func FindSelectNotIn(db *gorm.DB, fields db.Fields, docModel interface{}, docs interface{}) error {
	result := db.Model(docModel).Not(fields).Find(docs)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func Delete(db *gorm.DB, doc interface{}) error {
	result := db.Delete(doc)
	return result.Error
}

func DeleteById(db *gorm.DB, id interface{}, doc interface{}) error {
	result := db.Where("id = ?", id).Delete(doc)
	return result.Error
}

func DeleteByField(db *gorm.DB, field string, value interface{}, doc interface{}) error {
	result := db.Where(fmt.Sprintf("\"%v\" = ?", field), value).Delete(doc)
	return result.Error
}

func Create(db *gorm.DB, doc interface{}) *gorm.DB {
	return db.Create(doc)
}

func UpdateFields(db *gorm.DB, fields db.Fields, doc interface{}) error {
	result := db.Model(doc).Updates(fields)
	return result.Error
}

func UpdateWithFilter(db *gorm.DB, filter *db.Filter, doc interface{}, fields db.Fields) error {
	h := SetFilter(db, filter, nil, doc)
	result := h.Updates(fields)
	return result.Error
}

func UpdateField(db *gorm.DB, field string, doc interface{}) error {
	result := db.Model(doc).Select(field).Updates(doc)
	return result.Error
}

type TransactionHandler func(tx *gorm.DB) error

func Transaction(db *gorm.DB, handler TransactionHandler) error {
	return db.Transaction(handler)
}

var ObjectTypeName = utils.ObjectTypeName

func UpdateFieldMulti(db *gorm.DB, fields db.Fields, doc interface{}, field string, value interface{}) error {
	result := db.Model(doc).Where(fields).Update(field, value)
	return result.Error
}

func UpdateFielsdMulti(db *gorm.DB, filter db.Fields, doc interface{}, newFields db.Fields) error {
	result := db.Model(doc).Where(filter).Updates(newFields)
	return result.Error
}

func UpdateFieldsAll(db *gorm.DB, doc interface{}, newFields db.Fields) error {
	result := db.Model(doc).Where("1 = 1").Updates(newFields)
	return result.Error
}

func Exists(g *gorm.DB, filter *Filter, obj interface{}) (bool, error) {

	h := g
	if filter != nil {
		h = prepareFilter(g.Model(obj), filter)
	}
	h.Limit(1)

	result := h.First(obj)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

func DeleteAllByFields(db *gorm.DB, fields db.Fields, docs interface{}) error {
	result := db.Where(fields).Delete(docs)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func FindAllInterval(db *gorm.DB, name string, interval *Interval, docs interface{}) error {
	h := prepareInterval(db, name, interval)
	result := h.Find(docs)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func DeleteAll(db *gorm.DB, docs interface{}) error {
	result := db.Where("1 = 1").Delete(docs)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func Sum(g *gorm.DB, paginator *Paginator, groupFields []string, sumFields []string, filter *Filter, model interface{}, dest ...interface{}) (int64, error) {

	h := g.Model(model)

	// construct select fields
	var selectFieldsB strings.Builder
	for _, sumField := range sumFields {
		if selectFieldsB.Len() != 0 {
			utils.BuildString(&selectFieldsB, ",")
		}
		utils.BuildString(&selectFieldsB, `sum("`, sumField, `") AS "`, sumField, `"`)
	}
	for _, groupField := range groupFields {
		if selectFieldsB.Len() != 0 {
			utils.BuildString(&selectFieldsB, ",")
		}
		utils.BuildString(&selectFieldsB, `"`, groupField, `"`)
	}
	selectFields := selectFieldsB.String()
	h = h.Select(selectFields)

	// construct group by
	for _, groupField := range groupFields {
		h = h.Group(groupField)
	}

	// find
	dst := utils.OptionalArg(model, dest...)
	return find(h, filter, paginator, dst)
}
