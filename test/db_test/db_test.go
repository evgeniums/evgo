package db_test

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/crud"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/db/db_gorm"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _, testBasePath, _, _ = runtime.Caller(0)
var testDir = filepath.Dir(testBasePath)

func TestInitDb(t *testing.T) {
	app, _ := test_utils.InitAppContext(t, testDir, nil, "maindb.json")
	app.Close()
}

type SampleModel1 struct {
	common.ObjectBase
	Field1 string `gorm:"uniqueIndex"`
	Field2 string `gorm:"index"`
}

type SampleModel2 struct {
	common.ObjectBase
	Field1 string `gorm:"index"`
	Field2 int    `gorm:"index"`
}

func dbModels() []interface{} {
	return append([]interface{}{}, &SampleModel1{}, &SampleModel2{}, &WithAmount{}, &Terminal{})
}

func TestCreateDatabase(t *testing.T) {
	app, _ := test_utils.InitAppContext(t, testDir, dbModels(), "maindb.json")
	defer app.Close()
}

func TestMainDbOperations(t *testing.T) {
	app, appCtx := test_utils.InitAppContext(t, testDir, dbModels(), "maindb.json")
	defer app.Close()

	doc1 := &SampleModel1{}
	doc1.InitObject()
	doc1.Field1 = "value1"
	doc1.Field2 = "value2"
	require.NoError(t, app.Db().Create(appCtx, doc1), "failed to create doc1 in database")

	docDb1 := &SampleModel1{}
	found, err := app.Db().FindByFields(appCtx, db.Fields{"field1": "value1"}, docDb1)
	require.NoError(t, err, "failed to find doc1 in database")
	assert.Equal(t, found, true)
	test_utils.ObjectEqual(t, doc1, docDb1)

	docDb1NotFound := &SampleModel1{}
	found, err = app.Db().FindByFields(appCtx, db.Fields{"field1": "value11"}, docDb1NotFound)
	require.NoError(t, err, "failed to find docDb1NotFound in database")
	assert.Equal(t, found, false)

	filter := &db.Filter{}
	filter.Fields = db.Fields{"field2": "value2"}
	filter.SortField = "field1"
	filter.SortDirection = db.SORT_ASC
	filter.Count = true
	docsDb1 := make([]*SampleModel1, 0)
	count, err := app.Db().FindWithFilter(appCtx, filter, &docsDb1)
	require.NoError(t, err, "failed to find docs with filter in database")
	assert.Equal(t, int64(1), count)
	require.Len(t, docsDb1, 1)
	test_utils.ObjectEqual(t, doc1, docsDb1[0])

	doc2 := &SampleModel1{}
	doc2.InitObject()
	doc2.Field1 = "value1"
	doc2.Field2 = "value2"
	assert.Error(t, app.Db().Create(appCtx, doc2), "doc with field1=valu1e must be unique in database")
	docsDb2 := make([]*SampleModel1, 0)
	filter.Count = false
	count, err = app.Db().FindWithFilter(appCtx, filter, &docsDb2)
	require.NoError(t, err, "failed to find docs with filter in database")
	assert.Equal(t, int64(1), count)
	require.Len(t, docsDb2, 1)
	test_utils.ObjectEqual(t, doc1, docsDb2[0])

	doc3 := &SampleModel1{}
	doc3.InitObject()
	doc3.Field1 = "value3"
	doc3.Field2 = "value2"
	assert.NoError(t, app.Db().Create(appCtx, doc3), "failed to create doc3 in database")

	docsDb3 := make([]*SampleModel1, 0)
	count, err = app.Db().FindWithFilter(appCtx, filter, &docsDb3)
	require.NoError(t, err, "failed to find docs with filter in database")
	require.Len(t, docsDb3, 2)
	require.Equal(t, int64(2), count)
	test_utils.ObjectEqual(t, doc1, docsDb3[0])
	test_utils.ObjectEqual(t, doc3, docsDb3[1])

	require.NoError(t, app.Db().Update(appCtx, doc3, db.Fields{"field1": "value3"}, db.Fields{"field2": "value33"}), "failed to update doc3 in database")

	docsDb4 := make([]*SampleModel1, 0)
	count, err = app.Db().FindWithFilter(appCtx, filter, &docsDb4)
	require.NoError(t, err, "failed to find docsDb4 with filter in database")
	require.Len(t, docsDb4, 1)
	require.Equal(t, int64(1), count)
	test_utils.ObjectEqual(t, doc1, docsDb4[0])

	docDb33 := &SampleModel1{}
	found, err = app.Db().FindByFields(appCtx, db.Fields{"field2": "value33"}, docDb33)
	require.NoError(t, err, "failed to find docDb33 in database")
	assert.Equal(t, found, true)
	assert.Equal(t, docDb33.Field1, doc3.Field1)
}

type ParentEssentials struct {
	ParentField1 string `json:"parent_field1"`
	ParentField2 string `json:"parent_field2"`
	ParentField3 string `json:"parent_field3"`
}

type Parent struct {
	common.ObjectBase
	common.WithNameBase
	ParentEssentials
}

type ParentDest struct {
	common.IDBase
	common.WithNameBase
	ParentEssentials
}

type ChildEssentials struct {
	ChildField1 string `json:"child_field1"`
	ChildField2 string `json:"child_field2"`
	ChildField3 string `json:"child_field3"`
}

type Child struct {
	common.ObjectBase
	ChildEssentials
	common.WithNameBase
	ParentId string `json:"parent_id"`
}

type ChildWithParentEssentials struct {
	ChildEssentials
	ParentEssentials
}

type ChildWithParent struct {
	ChildWithParentEssentials
	ParentId   string `json:"parents.id" gorm:"->;column:parents_id"`
	ParentName string `json:"parents.name" gorm:"->;column:parents_name"`
	ChildId    string `json:"children.id" gorm:"->;column:children_id"`
	ChildName  string `json:"children.name" gorm:"->;column:children_name"`
}

func TestDottedJsonFields(t *testing.T) {

	obj := &ChildWithParent{}
	obj.ParentId = "p1"
	obj.ChildId = "c1"
	obj.ParentName = "parent name 1"
	obj.ChildName = "child name 1"
	obj.ParentField1 = "parent value 1"
	obj.ChildField1 = "child value 1"

	b, err := json.MarshalIndent(obj, "", "  ")
	assert.NoError(t, err)
	t.Logf("Object: \n %s", string(b))
}

type Terminal struct {
	common.ObjectBase
	common.WithNameBase
}

type WithAmount struct {
	Field1     string  `gorm:"index" json:"field1"`
	Field2     string  `gorm:"index" json:"field2"`
	Field3     string  `gorm:"index" json:"field3"`
	Amount1    int     `gorm:"index" json:"amount1"`
	Amount2    int     `gorm:"index" json:"amount2"`
	Amount3    float64 `gorm:"index" json:"amount3"`
	TerminalId string  `gorm:"index" json:"terminal_id"`
}

type WithAmountItem struct {
	WithAmount   `source:"with_amounts"`
	TerminalName string `source:"terminals.name" json:"terminal_name"`
}

func TestSum(t *testing.T) {

	app, appCtx := test_utils.InitAppContext(t, testDir, dbModels(), "maindb.json")
	defer app.Close()

	add := func(doc *WithAmount) {
		require.NoError(t, app.Db().Create(appCtx, doc), "failed to create doc in database")
	}

	doc := &WithAmount{}
	doc.Field1 = "value1"
	doc.Field2 = "value2.1"
	doc.Field3 = "value3.1"
	doc.Amount1 = 1
	doc.Amount2 = 10
	doc.Amount3 = 100
	add(doc)

	doc.Field3 = "value3.2"
	add(doc)

	doc.Field3 = "value3.3"
	add(doc)

	doc.Field2 = "value2.2"
	add(doc)

	doc.Field3 = "value3.4"
	add(doc)

	var dest1 []WithAmount
	count, err := app.Db().Sum(appCtx, []string{"field1", "field2"}, []string{"amount1", "amount2", "amount3"}, nil, &dest1)
	b, err1 := json.MarshalIndent(dest1, "", "  ")
	assert.NoError(t, err1)
	t.Logf("Result: \n %s", string(b))
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	require.Equal(t, 2, len(dest1))
	assert.Equal(t, "value1", dest1[0].Field1)
	assert.Equal(t, "value2.1", dest1[0].Field2)
	assert.Equal(t, 3, dest1[0].Amount1)
	assert.Equal(t, 30, dest1[0].Amount2)
	assert.InEpsilon(t, 300.00, dest1[0].Amount3, 0.001)
	assert.Equal(t, "value1", dest1[1].Field1)
	assert.Equal(t, "value2.2", dest1[1].Field2)
	assert.Equal(t, 2, dest1[1].Amount1)
	assert.Equal(t, 20, dest1[1].Amount2)
	assert.InEpsilon(t, 200.00, dest1[1].Amount3, 0.001)

	var dest2 []WithAmount
	count, err = app.Db().Sum(appCtx, []string{"field1"}, []string{"amount1", "amount2", "amount3"}, nil, &dest2)
	b, err1 = json.MarshalIndent(dest2, "", "  ")
	assert.NoError(t, err1)
	t.Logf("Result: \n %s", string(b))
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	require.Equal(t, 1, len(dest2))
	assert.Equal(t, "value1", dest2[0].Field1)
	assert.Equal(t, 5, dest2[0].Amount1)
	assert.Equal(t, 50, dest2[0].Amount2)
	assert.InEpsilon(t, 500.00, dest2[0].Amount3, 0.001)

	var dest3 []WithAmount
	filter := db.NewFilter()
	filter.AddFieldNotIn("field3", "value3.2", "value3.4")
	count, err = app.Db().Sum(appCtx, []string{"field1"}, []string{"amount1", "amount2", "amount3"}, filter, &dest3)
	b, err1 = json.MarshalIndent(dest3, "", "  ")
	assert.NoError(t, err1)
	t.Logf("Result: \n %s", string(b))
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	require.Equal(t, 1, len(dest3))
	assert.Equal(t, "value1", dest3[0].Field1)
	assert.Equal(t, 3, dest3[0].Amount1)
	assert.Equal(t, 30, dest3[0].Amount2)
	assert.InEpsilon(t, 300.00, dest3[0].Amount3, 0.001)

	var dest4 []WithAmount
	count, err = app.Db().Sum(appCtx, []string{}, []string{"amount1", "amount2", "amount3"}, nil, &dest4)
	b, err1 = json.MarshalIndent(dest4, "", "  ")
	assert.NoError(t, err1)
	t.Logf("Result: \n %s", string(b))
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	require.Equal(t, 1, len(dest2))
	assert.Equal(t, "", dest4[0].Field1)
	assert.Equal(t, "", dest4[0].Field2)
	assert.Equal(t, "", dest4[0].Field3)
	assert.Equal(t, 5, dest4[0].Amount1)
	assert.Equal(t, 50, dest4[0].Amount2)
	assert.InEpsilon(t, 500.00, dest4[0].Amount3, 0.001)
}

func TestJoinSum(t *testing.T) {

	app, appCtx := test_utils.InitAppContext(t, testDir, dbModels(), "maindb.json")
	defer app.Close()
	crud := &crud.DbCRUD{}

	// create terminals
	terminal1 := &Terminal{}
	terminal1.InitObject()
	terminal1.SetName("terminal1")
	require.NoError(t, app.Db().Create(appCtx, terminal1))

	terminal2 := &Terminal{}
	terminal2.InitObject()
	terminal2.SetName("terminal2")
	require.NoError(t, app.Db().Create(appCtx, terminal2))

	terminal3 := &Terminal{}
	terminal3.InitObject()
	terminal3.SetName("terminal3")
	require.NoError(t, app.Db().Create(appCtx, terminal3))

	// add docs

	add := func(doc *WithAmount) {
		require.NoError(t, app.Db().Create(appCtx, doc), "failed to create doc in database")
	}

	doc := &WithAmount{}
	doc.TerminalId = terminal1.GetID()
	doc.Field1 = "value1"
	doc.Field2 = "value2.1"
	doc.Field3 = "value3.1"
	doc.Amount1 = 1
	doc.Amount2 = 10
	doc.Amount3 = 100
	add(doc)
	doc.Field2 = "value2.2"
	add(doc)
	doc.Field3 = "value3.2"
	add(doc)
	doc.Field3 = "value3.3"
	add(doc)
	doc.Field3 = "value3.4"
	add(doc)

	// construct join query 1
	queryBuilder1 := func() (db.JoinQuery, error) {
		return app.Db().Joiner().
			Join(&WithAmount{}, "terminal_id").On(&Terminal{}, "id").
			Sum([]string{"field1", "field2"}, []string{"amount1", "amount2", "amount3"}).
			Destination(&WithAmountItem{})
	}

	// invoke join 1
	_, sctx1 := test_utils.SimpleOpContext(app, "query1")
	var items1 []WithAmountItem
	count, err := crud.Join(sctx1, db.NewJoin(queryBuilder1, "Query1"), nil, &items1)
	b, err1 := json.MarshalIndent(items1, "", "  ")
	assert.NoError(t, err1)
	t.Logf("Result:\n%s", string(b))
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	require.Equal(t, 2, len(items1))
	assert.Equal(t, "value1", items1[0].Field1)
	assert.Equal(t, "value2.1", items1[0].Field2)
	assert.Equal(t, 1, items1[0].Amount1)
	assert.Equal(t, 10, items1[0].Amount2)
	assert.InEpsilon(t, 100.00, items1[0].Amount3, 0.001)
	assert.Equal(t, "value1", items1[1].Field1)
	assert.Equal(t, "value2.2", items1[1].Field2)
	assert.Equal(t, 4, items1[1].Amount1)
	assert.Equal(t, 40, items1[1].Amount2)
	assert.InEpsilon(t, 400.00, items1[1].Amount3, 0.001)

	// construct join query 12
	queryBuilder2 := func() (db.JoinQuery, error) {
		return app.Db().Joiner().
			Join(&WithAmount{}, "terminal_id").On(&Terminal{}, "id").
			Sum([]string{"terminal_id", "terminal_name"}, []string{"amount1", "amount2", "amount3"}).
			Destination(&WithAmountItem{})
	}

	// invoke join with single terminal
	_, sctx2 := test_utils.SimpleOpContext(app, "query2")
	var items2 []WithAmountItem
	count, err = crud.Join(sctx2, db.NewJoin(queryBuilder2, "Query2"), nil, &items2)
	b, err1 = json.MarshalIndent(items2, "", "  ")
	assert.NoError(t, err1)
	t.Logf("Result:\n%s", string(b))
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	require.Equal(t, 1, len(items2))
	assert.Equal(t, terminal1.GetID(), items2[0].TerminalId)
	assert.Equal(t, terminal1.Name(), items2[0].TerminalName)
	assert.Equal(t, 5, items2[0].Amount1)
	assert.Equal(t, 50, items2[0].Amount2)
	assert.InEpsilon(t, 500.00, items2[0].Amount3, 0.001)

	// multiple terminals
	doc.TerminalId = terminal2.GetID()
	doc.Field3 = "value3.5"
	add(doc)
	doc.TerminalId = terminal2.GetID()
	doc.Field3 = "value3.6"
	add(doc)
	doc.TerminalId = terminal2.GetID()
	doc.Field3 = "value3.7"
	add(doc)
	doc.TerminalId = terminal3.GetID()
	doc.Field1 = "value1.8"
	add(doc)

	_, sctx3 := test_utils.SimpleOpContext(app, "query3")
	var items3 []WithAmountItem
	count, err = crud.Join(sctx3, db.NewJoin(queryBuilder2, "Query2"), nil, &items3)
	b, err1 = json.MarshalIndent(items3, "", "  ")
	assert.NoError(t, err1)
	t.Logf("Result:\n%s", string(b))
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
	require.Equal(t, 3, len(items3))
	assert.Equal(t, terminal1.GetID(), items3[0].TerminalId)
	assert.Equal(t, terminal1.Name(), items3[0].TerminalName)
	assert.Equal(t, 5, items3[0].Amount1)
	assert.Equal(t, 50, items3[0].Amount2)
	assert.InEpsilon(t, 500.00, items3[0].Amount3, 0.001)
	assert.Equal(t, terminal2.GetID(), items3[1].TerminalId)
	assert.Equal(t, terminal2.Name(), items3[1].TerminalName)
	assert.Equal(t, 3, items3[1].Amount1)
	assert.Equal(t, 30, items3[1].Amount2)
	assert.InEpsilon(t, 300.00, items3[1].Amount3, 0.001)
	assert.Equal(t, terminal3.GetID(), items3[2].TerminalId)
	assert.Equal(t, terminal3.Name(), items3[2].TerminalName)
	assert.Equal(t, 1, items3[2].Amount1)
	assert.Equal(t, 10, items3[2].Amount2)
	assert.InEpsilon(t, 100.00, items3[2].Amount3, 0.001)

	// run with filter
	_, sctx4 := test_utils.SimpleOpContext(app, "query4")
	var items4 []WithAmountItem
	filter := db.NewFilter()
	filter.AddFieldIn("terminal_name", "terminal1", "terminal3")
	count, err = crud.Join(sctx4, db.NewJoin(queryBuilder2, "Query2"), filter, &items4)
	b, err1 = json.MarshalIndent(items4, "", "  ")
	assert.NoError(t, err1)
	t.Logf("Result:\n%s", string(b))
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	require.Equal(t, 2, len(items4))
	assert.Equal(t, terminal1.GetID(), items4[0].TerminalId)
	assert.Equal(t, terminal1.Name(), items4[0].TerminalName)
	assert.Equal(t, 5, items4[0].Amount1)
	assert.Equal(t, 50, items4[0].Amount2)
	assert.InEpsilon(t, 500.00, items4[0].Amount3, 0.001)
	assert.Equal(t, terminal3.GetID(), items4[1].TerminalId)
	assert.Equal(t, terminal3.Name(), items4[1].TerminalName)
	assert.Equal(t, 1, items4[1].Amount1)
	assert.Equal(t, 10, items4[1].Amount2)
	assert.InEpsilon(t, 100.00, items4[1].Amount3, 0.001)
}

func TestQueryFilter(t *testing.T) {

	store := db.GlobalModelStore()
	if store == nil {
		store = db_gorm.NewModelStore(true)
	}
	store.RegisterModel(&common.ObjectBase{})
	fmanager := db_gorm.NewFilterManager()

	qStr := `{"limit":2,"intervals":{"created_at":{"from":"2023-06-19"}}}`
	q := &db.Query{}
	err := json.Unmarshal([]byte(qStr), q)
	require.NoError(t, err)

	f, err := fmanager.ParseFilterDirect(q, &common.ObjectBase{}, "")
	require.NoError(t, err)
	require.NotNil(t, f)

	test_utils.DumpObject(t, f, "Filter")
}

func TestInList(t *testing.T) {

	app, _ := test_utils.InitAppContext(t, testDir, dbModels(), "maindb.json")
	defer app.Close()
	crud := &crud.DbCRUD{}
	_, sctx := test_utils.SimpleOpContext(app, "query")

	ids := []string{"a", "b", "c"}
	var result []*SampleModel1
	f := db.NewFilter()
	ifs := utils.ListInterfaces(ids...)
	f.AddFieldIn("id", ifs...)
	_, err := crud.List(sctx, f, &result)
	test_utils.NoError(t, sctx, err)
}
