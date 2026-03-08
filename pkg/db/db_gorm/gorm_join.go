package db_gorm

import (
	"context"
	"fmt"

	"github.com/evgeniums/evgo/pkg/db"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type JoinTable struct {
	db.JoinTableBase
	schema *schema.Schema
}

func (jt *JoinTable) Schema(f *FilterManager) (*schema.Schema, error) {
	if jt.schema == nil {
		var err error
		jt.schema, err = schema.Parse(jt.Model(), f.modelStore.schemaCache, f.modelStore.schemaNamer)
		return jt.schema, err
	}
	return jt.schema, nil
}

type JoinPair struct {
	db.JoinPairBase
	left  *JoinTable
	right *JoinTable
}

type JoinQueryConstructor struct {
	db.JoinQueryBase
	pairs       []*JoinPair
	sumFields   map[string]bool
	groupFields map[string]bool
}

// TODO make join type configurable LEFT/LEFT OUTER/RIGHT/RIGHT OUTER
func constructJoins(g *gorm.DB, f *FilterManager, q *JoinQueryConstructor) (*gorm.DB, error) {
	db := g

	for _, pair := range q.pairs {
		leftSchema, err := pair.left.Schema(f)
		if err != nil {
			return nil, err
		}
		rightSchema, err := pair.right.Schema(f)
		if err != nil {
			return nil, err
		}
		join := fmt.Sprintf("LEFT OUTER JOIN \"%s\" ON \"%s\".\"%s\"=\"%s\".\"%s\"", rightSchema.Table, leftSchema.Table, pair.LeftField(), rightSchema.Table, pair.RightField())
		db = db.Joins(join)
	}

	return db, nil
}

func PrepareJoin(f *FilterManager, g *gorm.DB, constructor *JoinQueryConstructor) (*gorm.DB, error) {

	q := constructor

	mainModel := q.pairs[0].left.Model()
	db := g.Model(mainModel)

	destinationSchema, err := schema.Parse(constructor.Destination(), f.modelStore.schemaCache, f.modelStore.schemaNamer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse destination model: %s", err)
	}

	destinationDescriptor := f.modelStore.FindDescriptor(destinationSchema.Table)
	if destinationDescriptor == nil {
		f.modelStore.RegisterModel(constructor.Destination())
		destinationDescriptor = f.modelStore.FindDescriptor(destinationSchema.Table)
	}

	selects := make([]string, 0, len(destinationDescriptor.FieldsJson))
	if destinationDescriptor.FieldsJson == nil {
		err = f.modelStore.ParseModelFields(destinationDescriptor)
		if err != nil {
			return nil, fmt.Errorf("failed to parse destination model fields: %s", err)
		}
	}

	sums := len(constructor.sumFields) > 0
	groups := len(constructor.groupFields) > 0 || sums

	for _, field := range destinationDescriptor.FieldsJson {
		if !groups {
			fieldSelect := fmt.Sprintf("\"%s\".\"%s\" AS \"%s\"", field.DbTable, field.DbField, field.Schema.DBName)
			selects = append(selects, fieldSelect)
		} else {
			if sums {
				_, isSum := constructor.sumFields[field.Schema.DBName]
				if isSum {
					fieldSelect := fmt.Sprintf("sum(\"%s\".\"%s\") AS \"%s\"", field.DbTable, field.DbField, field.Schema.DBName)
					selects = append(selects, fieldSelect)
				}
			}
			if groups {
				_, isGroup := constructor.groupFields[field.Schema.DBName]
				if isGroup {
					fieldSelect := fmt.Sprintf("\"%s\".\"%s\" AS \"%s\"", field.DbTable, field.DbField, field.Schema.DBName)
					selects = append(selects, fieldSelect)
				}
			}
		}
	}

	db = db.Select(selects)
	db, err = constructJoins(db, f, q)
	if err != nil {
		return nil, fmt.Errorf("failed to construct joins: %s", err)
	}

	for groupField := range constructor.groupFields {
		db = db.Group(groupField)
	}

	return db, nil
}

type JoinQuery struct {
	db              *GormDB
	preparedSession *gorm.DB
}

func (j *JoinQuery) Join(sctx context.Context, filter *Filter, dest interface{}) (int64, error) {
	// TODO process filter fields to match nested field names
	session := j.preparedSession.Session(&gorm.Session{})
	if j.db != nil && j.db.ENABLE_DEBUG {
		session = session.Debug()
	}
	return find(session, filter, j.db.paginator, dest)
}

type Joiner struct {
	db          *GormDB
	session     *gorm.DB
	constructor *JoinQueryConstructor
	pair        *JoinPair
}

func newJoiner(db *GormDB) *Joiner {
	j := &Joiner{}
	j.constructor = newJoinQueryConstuctor()
	j.session = db.db_().Session(&gorm.Session{})
	j.db = db
	return j
}

func newJoinQueryConstuctor() *JoinQueryConstructor {
	q := &JoinQueryConstructor{}
	q.pairs = make([]*JoinPair, 0)
	q.groupFields = make(map[string]bool)
	q.sumFields = make(map[string]bool)
	return q
}

func (j *Joiner) Sum(groupFields []string, sumFields []string) db.JoinEnd {

	if j.constructor == nil {
		j.constructor = newJoinQueryConstuctor()
	}

	for _, field := range sumFields {
		j.constructor.sumFields[field] = true
	}

	for _, field := range groupFields {
		j.constructor.groupFields[field] = true
	}

	return j
}

func (j *Joiner) Destination(destination interface{}) (db.JoinQuery, error) {
	if j.constructor == nil {
		j.constructor = newJoinQueryConstuctor()
	}
	j.constructor.SetDestination(destination)

	preparedSession, err := PrepareJoin(j.db.filterManager, j.session, j.constructor)
	if err != nil {
		return nil, err
	}

	q := &JoinQuery{preparedSession: preparedSession, db: j.db}
	models := make(map[string]interface{})
	for _, pair := range j.constructor.pairs {
		models[pair.left.schema.Table] = pair.left.Model
		models[pair.right.schema.Table] = pair.right.Model
	}

	return q, nil
}

func (j *Joiner) Join(model interface{}, field string) db.JoinBegin {
	if j.constructor == nil {
		j.constructor = newJoinQueryConstuctor()
	}
	j.pair = &JoinPair{}
	j.pair.left = &JoinTable{}
	j.pair.left.JoinTableData.Model = model
	j.pair.JoinPairData.LeftField = field
	return j
}

func (j *Joiner) On(model interface{}, field string) db.JoinEnd {
	if j.constructor == nil || j.pair == nil {
		panic("can not call ON without calling Join first")
	}
	j.pair.right = &JoinTable{}
	j.pair.right.JoinTableData.Model = model
	j.pair.JoinPairData.RightField = field
	j.constructor.pairs = append(j.constructor.pairs, j.pair)
	return j
}

func (g *GormDB) Joiner() db.Joiner {
	return newJoiner(g)
}

func (g *GormDB) Join(sctx context.Context, joinConfig *db.JoinQueryConfig, filter *Filter, dest interface{}) (int64, error) {
	q, err := g.joinQueries.FindOrCreate(joinConfig)
	if err != nil {
		return 0, err
	}
	return q.Join(sctx, filter, dest)
}
