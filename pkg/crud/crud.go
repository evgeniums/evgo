package crud

import (
	"context"

	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type CRUD interface {
	Create(sctx context.Context, object common.Object) error
	CreateDup(sctx context.Context, object common.Object, ignoreConflict ...bool) (bool, error)

	Read(sctx context.Context, fields db.Fields, object interface{}, dest ...interface{}) (bool, error)
	ReadByField(sctx context.Context, fieldName string, fieldValue interface{}, object interface{}, dest ...interface{}) (bool, error)
	ReadForUpdate(sctx context.Context, fields db.Fields, object interface{}) (bool, error)
	ReadForShare(sctx context.Context, fields db.Fields, object interface{}) (bool, error)
	Update(sctx context.Context, object common.Object, fields db.Fields) error
	UpdateMonthObject(sctx context.Context, obj common.ObjectWithMonth, fields db.Fields) error
	UpdateMulti(sctx context.Context, model interface{}, filter db.Fields, fields db.Fields) error
	UpdateWithFilter(sctx context.Context, model interface{}, filter *db.Filter, fields db.Fields) error
	Delete(sctx context.Context, object common.Object) error
	DeleteByFields(sctx context.Context, field db.Fields, object common.Object) error

	List(sctx context.Context, filter *db.Filter, object interface{}, dest ...interface{}) (int64, error)
	Exists(sctx context.Context, filter *db.Filter, object interface{}) (bool, error)

	Join(sctx context.Context, joinConfig *db.JoinQueryConfig, filter *db.Filter, dest interface{}) (int64, error)

	Db(sctx context.Context) db.DBHandlers
}

type WithCRUD interface {
	CRUD() CRUD
}

type DbCRUD struct {
	ForceMainDb bool
	DryRun      bool
}

type WithCRUDBase struct {
	crud CRUD
}

func (w *WithCRUDBase) Construct(cruds ...CRUD) {
	if len(cruds) == 0 {
		w.crud = &DbCRUD{}
	} else {
		w.crud = cruds[0]
	}
}

func (w *WithCRUDBase) CRUD() CRUD {
	return w.crud
}

func (w *WithCRUDBase) SetForceMainDb(enable bool) {
	dbCrud, ok := w.crud.(*DbCRUD)
	if ok {
		dbCrud.ForceMainDb = enable
	}
}

func (w *WithCRUDBase) IsForceMainDb() bool {
	dbCrud, ok := w.crud.(*DbCRUD)
	if ok {
		return dbCrud.ForceMainDb
	}
	return false
}

func (w *WithCRUDBase) SetDryRun(enable bool) {
	dbCrud, ok := w.crud.(*DbCRUD)
	if ok {
		dbCrud.DryRun = enable
	}
}

func (w *WithCRUDBase) IsDryRun() bool {
	dbCrud, ok := w.crud.(*DbCRUD)
	if ok {
		return dbCrud.DryRun
	}
	return false
}

func (d *DbCRUD) Db(sctx context.Context) db.DBHandlers {
	ctx := op_context.OpContext[op_context.Context](sctx)
	return op_context.DB(ctx, d.ForceMainDb)
}

func (d *DbCRUD) Create(sctx context.Context, object common.Object) error {

	if d.DryRun {
		return nil
	}

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.Create")
	defer ctx.TraceOutMethod()

	err := op_context.DB(ctx, d.ForceMainDb).Create(sctx, object)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func (d *DbCRUD) CreateDup(sctx context.Context, object common.Object, ignoreConflict ...bool) (bool, error) {

	if d.DryRun {
		return false, nil
	}

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.CreateDup")
	defer ctx.TraceOutMethod()

	duplicate, err := op_context.DB(ctx, d.ForceMainDb).CreateDup(sctx, object, ignoreConflict...)
	if err != nil {
		return duplicate, c.SetError(err)
	}

	return false, nil
}

func (d *DbCRUD) Read(sctx context.Context, fields db.Fields, object interface{}, dest ...interface{}) (bool, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.Read")
	defer ctx.TraceOutMethod()

	found, err := op_context.DB(ctx, d.ForceMainDb).FindByFields(sctx, fields, object, dest...)
	if err != nil {
		return found, c.SetError(err)
	}

	return found, nil
}

func (d *DbCRUD) ReadForUpdate(sctx context.Context, fields db.Fields, object interface{}) (bool, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.ReadForUpdate")
	defer ctx.TraceOutMethod()

	found, err := op_context.DB(ctx, d.ForceMainDb).FindForUpdate(sctx, fields, object)
	if err != nil {
		return found, c.SetError(err)
	}

	return found, nil
}

func (d *DbCRUD) ReadForShare(sctx context.Context, fields db.Fields, object interface{}) (bool, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.ReadForShare")
	defer ctx.TraceOutMethod()

	found, err := op_context.DB(ctx, d.ForceMainDb).FindForShare(sctx, fields, object)
	if err != nil {
		return found, c.SetError(err)
	}

	return found, nil
}

func (d *DbCRUD) ReadByField(sctx context.Context, fieldName string, fieldValue interface{}, object interface{}, dest ...interface{}) (bool, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.Read", logger.Fields{fieldName: fieldValue})
	defer ctx.TraceOutMethod()

	found, err := op_context.DB(ctx, d.ForceMainDb).FindByField(sctx, fieldName, fieldValue, object, dest...)
	if err != nil {
		return found, c.SetError(err)
	}

	return found, nil
}

func (d *DbCRUD) Update(sctx context.Context, obj common.Object, fields db.Fields) error {

	if d.DryRun {
		return nil
	}

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.Update")
	defer ctx.TraceOutMethod()

	err := db.Update(op_context.DB(ctx, d.ForceMainDb), sctx, obj, fields)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func (d *DbCRUD) UpdateMonthObject(sctx context.Context, obj common.ObjectWithMonth, fields db.Fields) error {

	if d.DryRun {
		return nil
	}

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.UpdateMonthObject")
	defer ctx.TraceOutMethod()

	err := db.UpdateMulti(op_context.DB(ctx, d.ForceMainDb), sctx, obj, db.Fields{"month": obj.GetMonth(), "id": obj.GetID()}, fields)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func (d *DbCRUD) UpdateMulti(sctx context.Context, model interface{}, filter db.Fields, fields db.Fields) error {

	if d.DryRun {
		return nil
	}

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.UpdateMulti")
	defer ctx.TraceOutMethod()

	var err error
	if filter == nil {
		err = db.UpdateAll(op_context.DB(ctx, d.ForceMainDb), sctx, model, fields)
	} else {
		err = db.UpdateMulti(op_context.DB(ctx, d.ForceMainDb), sctx, model, filter, fields)
	}
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func (d *DbCRUD) UpdateWithFilter(sctx context.Context, model interface{}, filter *db.Filter, fields db.Fields) error {

	if d.DryRun {
		return nil
	}

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.UpdateWithFilter")
	defer ctx.TraceOutMethod()

	err := db.UpdateWithFilter(op_context.DB(ctx, d.ForceMainDb), sctx, model, filter, fields)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func (d *DbCRUD) Delete(sctx context.Context, object common.Object) error {

	if d.DryRun {
		return nil
	}

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.Delete")
	defer ctx.TraceOutMethod()

	err := op_context.DB(ctx, d.ForceMainDb).Delete(sctx, object)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func (d *DbCRUD) DeleteByFields(sctx context.Context, fields db.Fields, object common.Object) error {

	if d.DryRun {
		return nil
	}

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.DeleteByFields")
	defer ctx.TraceOutMethod()

	err := op_context.DB(ctx, d.ForceMainDb).DeleteByFields(sctx, fields, object)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func (d *DbCRUD) List(sctx context.Context, filter *db.Filter, objects interface{}, dest ...interface{}) (int64, error) {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.List")
	defer ctx.TraceOutMethod()
	count, err := op_context.DB(ctx, d.ForceMainDb).FindWithFilter(sctx, filter, objects, dest...)
	if err != nil {
		return 0, c.SetError(err)
	}
	return count, nil
}

func (d *DbCRUD) Exists(sctx context.Context, filter *db.Filter, object interface{}) (bool, error) {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.Exists")
	defer ctx.TraceOutMethod()
	exists, err := op_context.DB(ctx, d.ForceMainDb).Exists(sctx, filter, object)
	if err != nil {
		return false, c.SetError(err)
	}
	return exists, nil
}

func (d *DbCRUD) Join(sctx context.Context, joinConfig *db.JoinQueryConfig, filter *db.Filter, dest interface{}) (int64, error) {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("CRUD.Join")
	defer ctx.TraceOutMethod()
	count, err := op_context.DB(ctx, d.ForceMainDb).Join(sctx, joinConfig, filter, dest)
	if err != nil {
		return 0, c.SetError(err)
	}
	return count, nil
}

func List[T common.Object](crud CRUD, sctx context.Context, methodName string, filter *db.Filter, objects *[]T, dest ...interface{}) (int64, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod(methodName)
	defer ctx.TraceOutMethod()

	count, err := crud.List(sctx, filter, objects, dest...)
	if err != nil {
		return 0, c.SetError(err)
	}

	return count, nil
}

func Find[T common.Object](crud CRUD, sctx context.Context, methodName string, fields db.Fields, object T, dest ...interface{}) (T, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod(methodName)
	defer ctx.TraceOutMethod()

	found, err := crud.Read(sctx, fields, object, dest...)
	if err != nil {
		return *new(T), c.SetError(err)
	}
	if !found {
		return *new(T), nil
	}
	return object, nil
}

func FindIdAndTopic[T common.Object](crud CRUD, sctx context.Context, methodName string, id string, topic string, object T, dest ...interface{}) (T, error) {
	return Find(crud, sctx, methodName, db.Fields{"id": id, "topic": topic}, object, dest...)
}

func FindByField[T common.Object](crud CRUD, sctx context.Context, methodName string, fieldName string, fieldValue interface{}, object T, dest ...interface{}) (T, error) {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod(methodName)
	defer ctx.TraceOutMethod()

	found, err := crud.ReadByField(sctx, fieldName, fieldValue, object, dest...)
	if err != nil {
		return *new(T), c.SetError(err)
	}
	if !found {
		return *new(T), nil
	}
	return object, nil
}

func Create(crud CRUD, sctx context.Context, methodName string, obj common.Object, loggerFields ...logger.Fields) error {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod(methodName, loggerFields...)
	defer ctx.TraceOutMethod()
	err := crud.Create(sctx, obj)
	if err != nil {
		return c.SetError(err)
	}
	return nil
}

func Update(crud CRUD, sctx context.Context, methodName string, obj common.Object, fields db.Fields, loggerFields ...logger.Fields) error {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod(methodName, loggerFields...)
	defer ctx.TraceOutMethod()
	err := crud.Update(sctx, obj, fields)
	if err != nil {
		return c.SetError(err)
	}
	return nil
}

func FindUpdate[T common.Object](crud CRUD, sctx context.Context, methodName string, fieldName string, fieldValue interface{}, fields db.Fields, object T, dest ...interface{}) (T, error) {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod(methodName)
	defer ctx.TraceOutMethod()
	var obj T
	var err error

	obj, err = FindByField(crud, sctx, "Find", fieldName, fieldValue, object)
	if err != nil {
		return *new(T), c.SetError(err)
	}
	if utils.IsNil(obj) {
		return obj, nil
	}

	err = Update(crud, sctx, "Update", obj, fields)
	if err != nil {
		return *new(T), c.SetError(err)
	}

	return obj, nil
}

func Delete(crud CRUD, sctx context.Context, methodName string, fieldName string, fieldValue interface{}, object common.Object, loggerFields ...logger.Fields) error {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod(methodName)
	defer ctx.TraceOutMethod()

	obj, err := FindByField(crud, sctx, "Find", fieldName, fieldValue, object)
	if err != nil {
		return c.SetError(err)
	}
	if obj == nil {
		return nil
	}

	err = crud.Delete(sctx, obj)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func DeleteByFields(crud CRUD, sctx context.Context, methodName string, fields db.Fields, object common.Object, loggerFields ...logger.Fields) error {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod(methodName)
	defer ctx.TraceOutMethod()

	err := crud.DeleteByFields(sctx, fields, object)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func Exists(crud CRUD, sctx context.Context, methodName string, filter *db.Filter, object interface{}) (bool, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod(methodName)
	defer ctx.TraceOutMethod()

	exists, err := crud.Exists(sctx, filter, object)
	if err != nil {
		return false, c.SetError(err)
	}

	return exists, nil
}

func FindOne[T common.Object](crud CRUD, sctx context.Context, filter *db.Filter, model T) (T, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("crud.FindOne")
	defer ctx.TraceOutMethod()

	var objects []T

	count, err := crud.List(sctx, filter, &objects)
	if err != nil {
		return *new(T), c.SetError(err)
	}

	if count == 0 {
		return *new(T), nil
	}

	return objects[0], nil
}
