package tenancy_manager

import (
	"errors"

	"github.com/evgeniums/evgo/pkg/crud"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
)

type TenancyController struct {
	generic_error.ErrorsExtenderBase
	CRUD    crud.CRUD
	Manager *TenancyManager
}

func NewTenancyController(crud crud.CRUD, manager *TenancyManager) *TenancyController {
	t := &TenancyController{}
	t.CRUD = crud
	t.Manager = manager
	t.Manager.SetController(t)

	t.ErrorsExtenderBase.Init(multitenancy.ErrorDescriptions, multitenancy.ErrorHttpCodes)

	return t
}

func DefaultTenancyController(manager *TenancyManager) *TenancyController {
	return NewTenancyController(&crud.DbCRUD{}, manager)
}

func (t *TenancyController) OpLog(ctx op_context.Context, operation string, oplog *multitenancy.OpLogTenancy) {
	oplog.SetOperation(operation)
	ctx.Oplog(oplog)
}

func (t *TenancyController) PublishOp(tenancy *multitenancy.TenancyItem, op string, poolIds ...string) {
	if len(poolIds) != 0 {
		t.Manager.PoolPubsub.PublishPools(multitenancy.PubsubTopicName,
			&multitenancy.PubsubNotification{Tenancy: tenancy.GetID(), Operation: op},
			poolIds...)
	} else {
		t.Manager.PoolPubsub.PublishPools(multitenancy.PubsubTopicName,
			&multitenancy.PubsubNotification{Tenancy: tenancy.GetID(), Operation: op},
			tenancy.PoolId())
	}
}

func (t *TenancyController) Add(ctx op_context.Context, data *multitenancy.TenancyData) (*multitenancy.TenancyItem, error) {

	// setup
	c := ctx.TraceInMethod("TenancyController.Add", logger.Fields{"customer": data.CUSTOMER_ID, "role": data.ROLE})
	defer ctx.TraceOutMethod()

	// create tenancy
	tenancy, err := t.Manager.CreateTenancy(ctx, data)
	if err != nil {
		c.SetMessage("failed to create tenancy")
		return nil, c.SetError(err)
	}

	// save tenancy in database
	err = t.CRUD.Create(ctx, &tenancy.TenancyDb)
	if err != nil {
		c.SetMessage("failed to save tenancy in database")
		return nil, c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpAdd, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		Role: tenancy.Role(), ShadowPath: tenancy.ShadowPath(), Path: tenancy.Path(), DbName: tenancy.DbName(), Pool: tenancy.PoolName, Customer: tenancy.CustomerDisplay()})

	// publish notification
	t.PublishOp(tenancy, multitenancy.OpAdd)

	// done
	return tenancy, nil
}

func (t *TenancyController) Find(ctx op_context.Context, id string, idIsDisplay ...bool) (*multitenancy.TenancyItem, error) {
	return multitenancy.FindTenancy(t, ctx, id, idIsDisplay...)
}

func (t *TenancyController) List(ctx op_context.Context, filter *db.Filter) ([]*multitenancy.TenancyItem, int64, error) {

	// setup
	c := ctx.TraceInMethod("TenancyController.List")
	defer ctx.TraceOutMethod()

	// construct join query
	queryBuilder := func() (db.JoinQuery, error) {
		return ctx.Db().Joiner().
			Join(&multitenancy.TenancyDb{}, "customer_id").On(&customer.Customer{}, "id").
			Join(&multitenancy.TenancyDb{}, "pool_id").On(&pool.PoolBase{}, "id").
			Destination(&multitenancy.TenancyItem{})
	}

	// invoke join
	var tenancies []*multitenancy.TenancyItem
	count, err := t.CRUD.Join(ctx, db.NewJoin(queryBuilder, "ListTenancies"), filter, &tenancies)
	if err != nil {
		c.SetMessage("failed to join tenancies")
		return nil, 0, c.SetError(err)
	}

	// done
	return tenancies, count, nil
}

func (t *TenancyController) Exists(ctx op_context.Context, fields db.Fields) (bool, error) {
	filter := db.NewFilter()
	filter.AddFields(fields)
	return crud.Exists(t.CRUD, ctx, "TenancyController.Exists", filter, &multitenancy.TenancyDb{})
}

func (t *TenancyController) SetPath(ctx op_context.Context, id string, path string, idIsDisplay ...bool) error {

	// setup
	c := ctx.TraceInMethod("TenancyController.SetPath")
	defer ctx.TraceOutMethod()

	// find tenancy
	tenancy, err := t.Find(ctx, id, idIsDisplay...)
	if err != nil {
		return c.SetError(err)
	}

	// check if tenancy with such path in that pool
	err = t.Manager.CheckDuplicatePath(ctx, c, tenancy.PoolId(), path)
	if err != nil {
		return err
	}

	// update field
	err = t.CRUD.Update(ctx, &tenancy.TenancyDb, db.Fields{"path": path})
	if err != nil {
		c.SetMessage("failed to update tenancy")
		return c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpSetPath, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		Role: tenancy.Role(), ShadowPath: tenancy.ShadowPath(), Path: tenancy.Path(), Customer: tenancy.CustomerDisplay()})

	// publish notification
	t.PublishOp(tenancy, multitenancy.OpSetPath)

	// done
	return nil
}

func (t *TenancyController) SetShadowPath(ctx op_context.Context, id string, path string, idIsDisplay ...bool) error {

	// setup
	c := ctx.TraceInMethod("TenancyController.SetShadowPath")
	defer ctx.TraceOutMethod()

	// find tenancy
	tenancy, err := t.Find(ctx, id, idIsDisplay...)
	if err != nil {
		return c.SetError(err)
	}

	// check if tenancy with such path in that pool
	err = t.Manager.CheckDuplicatePath(ctx, c, tenancy.PoolId(), path)
	if err != nil {
		return err
	}

	// update field
	err = t.CRUD.Update(ctx, &tenancy.TenancyDb, db.Fields{"shadow_path": path})
	if err != nil {
		c.SetMessage("failed to update tenancy")
		return c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpSetShadowPath, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		Role: tenancy.Role(), ShadowPath: tenancy.ShadowPath(), Path: tenancy.Path(), Customer: tenancy.CustomerDisplay()})

	// publish notification
	t.PublishOp(tenancy, multitenancy.OpSetShadowPath)

	// done
	return nil
}

func (t *TenancyController) SetRole(ctx op_context.Context, id string, role string, idIsDisplay ...bool) error {

	// setup
	c := ctx.TraceInMethod("TenancyController.SetRole")
	defer ctx.TraceOutMethod()

	// find tenancy
	tenancy, err := t.Find(ctx, id, idIsDisplay...)
	if err != nil {
		return c.SetError(err)
	}

	// check if tenancy with such role for this customer exists
	err = t.Manager.CheckDuplicateRole(ctx, c, tenancy.CustomerId(), role)
	if err != nil {
		return err
	}

	// update field
	err = t.CRUD.Update(ctx, &tenancy.TenancyDb, db.Fields{"role": role})
	if err != nil {
		c.SetMessage("failed to update tenancy")
		return c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpSetRole, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		Role: tenancy.Role(), Customer: tenancy.CustomerDisplay()})

	// publish notification
	t.PublishOp(tenancy, multitenancy.OpSetRole)

	// done
	return nil
}

func (t *TenancyController) Activate(ctx op_context.Context, id string, idIsDisplay ...bool) error {

	// setup
	c := ctx.TraceInMethod("TenancyController.activate")
	defer ctx.TraceOutMethod()

	// find tenancy
	tenancy, err := t.Find(ctx, id, idIsDisplay...)
	if err != nil {
		return c.SetError(err)
	}

	// update field
	err = t.CRUD.Update(ctx, &tenancy.TenancyDb, db.Fields{"active": true})
	if err != nil {
		c.SetMessage("failed to update tenancy")
		return c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpActivate, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		Role: tenancy.Role(), Customer: tenancy.CustomerDisplay()})

	// publish notification
	t.PublishOp(tenancy, multitenancy.OpActivate)

	// done
	return nil
}

func (t *TenancyController) Deactivate(ctx op_context.Context, id string, idIsDisplay ...bool) error {

	// setup
	c := ctx.TraceInMethod("TenancyController.activate")
	defer ctx.TraceOutMethod()

	// find tenancy
	tenancy, err := t.Find(ctx, id, idIsDisplay...)
	if err != nil {
		return c.SetError(err)
	}

	// update field
	err = t.CRUD.Update(ctx, &tenancy.TenancyDb, db.Fields{"active": false})
	if err != nil {
		c.SetMessage("failed to update tenancy")
		return c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpDeactivate, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		Role: tenancy.Role(), Customer: tenancy.CustomerDisplay()})

	// publish notification
	t.PublishOp(tenancy, multitenancy.OpDeactivate)

	// done
	return nil
}

func (t *TenancyController) SetCustomer(ctx op_context.Context, id string, customer string, idIsDisplay ...bool) error {

	// setup
	c := ctx.TraceInMethod("TenancyController.SetRole")
	defer ctx.TraceOutMethod()

	// find tenancy
	tenancy, err := t.Find(ctx, id, idIsDisplay...)
	if err != nil {
		return c.SetError(err)
	}

	// find customer
	cust, err := t.Manager.FindCustomer(ctx, c, customer)
	if err != nil {
		return c.SetError(err)
	}

	// check if tenancy with such role for this customer exists
	err = t.Manager.CheckDuplicateRole(ctx, c, cust.GetID(), tenancy.Role())
	if err != nil {
		return err
	}

	// update field
	err = t.CRUD.Update(ctx, &tenancy.TenancyDb, db.Fields{"customer_id": cust.GetID()})
	if err != nil {
		c.SetMessage("failed to update tenancy")
		return c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpSetCustomer, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		Role: tenancy.Role(), Customer: cust.Display()})

	// publish notification
	t.PublishOp(tenancy, multitenancy.OpSetCustomer)

	// done
	return nil
}

func (t *TenancyController) ChangePoolOrDb(ctx op_context.Context, id string, poolId string, dbName string, idIsDisplay ...bool) error {

	// setup
	c := ctx.TraceInMethod("TenancyController.ChangePoolOrDb")
	defer ctx.TraceOutMethod()

	// find tenancy
	tenancy, err := t.Find(ctx, id, idIsDisplay...)
	if err != nil {
		return c.SetError(err)
	}
	oldPoolId := tenancy.PoolId()

	// find pool
	pId := poolId
	if pId == "" {
		pId = tenancy.PoolId()
	}
	p, err := t.Manager.FindPool(ctx, c, pId)
	if err != nil {
		return c.SetError(err)
	}

	// check if tenancy with such path in new pool exists
	if p.GetID() != tenancy.PoolId() {
		err = t.Manager.CheckDuplicatePath(ctx, c, p.GetID(), tenancy.Path())
		if err != nil {
			return err
		}
		err = t.Manager.CheckDuplicatePath(ctx, c, p.GetID(), tenancy.ShadowPath())
		if err != nil {
			return err
		}
	}

	dbN := dbName
	if dbN == "" {
		dbN = tenancy.DbName()
	}
	tenancy.DBNAME = dbN
	tenancy.POOL_ID = p.GetID()

	// check database
	if tenancy.IsActive() {
		checkTenancy := NewTenancy(t.Manager)
		err = checkTenancy.Init(ctx, &tenancy.TenancyDb)
		multitenancy.CloseTenancyDb(checkTenancy)
		if err != nil {
			c.SetMessage("failed to init tenancy with new parameters")
			return c.SetError(err)
		}
		if checkTenancy.TenancyBaseData.Pool == nil || !checkTenancy.TenancyBaseData.Pool.IsActive() {
			ctx.SetGenericErrorCode(pool.ErrorCodePoolNotActive)
			err = errors.New("failed to check tenancy database as the pool is not active")
			return c.SetError(err)
		}
	}

	// update fields
	err = t.CRUD.Update(ctx, &tenancy.TenancyDb, db.Fields{"pool_id": p.GetID(), "dbname": dbN})
	if err != nil {
		c.SetMessage("failed to update tenancy")
		return c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpChangePoolOrDb, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		Role: tenancy.Role(), Customer: tenancy.CustomerDisplay(), Pool: p.Name(), DbName: dbN})

	// publish notification
	if oldPoolId != pId {
		t.PublishOp(tenancy, multitenancy.OpDelete, oldPoolId)
	}
	t.PublishOp(tenancy, multitenancy.OpChangePoolOrDb)

	// done
	return nil
}

func (t *TenancyController) SetDbRole(ctx op_context.Context, id string, dbRole string, idIsDisplay ...bool) error {

	// setup
	c := ctx.TraceInMethod("TenancyController.SetDbRole")
	defer ctx.TraceOutMethod()

	// find tenancy
	tenancy, err := t.Find(ctx, id, idIsDisplay...)
	if err != nil {
		return c.SetError(err)
	}

	// check database
	tenancy.DB_ROLE = dbRole
	checkTenancy := NewTenancy(t.Manager)
	err = checkTenancy.Init(ctx, &tenancy.TenancyDb)
	multitenancy.CloseTenancyDb(checkTenancy)
	if err != nil {
		c.SetMessage("failed to init tenancy with new parameters")
		return c.SetError(err)
	}
	if checkTenancy.TenancyBaseData.Pool == nil || !checkTenancy.TenancyBaseData.Pool.IsActive() {
		ctx.SetGenericErrorCode(pool.ErrorCodePoolNotActive)
		err = errors.New("failed to check tenancy database as the pool is not active")
		return c.SetError(err)
	}

	// update fields
	err = t.CRUD.Update(ctx, &tenancy.TenancyDb, db.Fields{"db_role": dbRole})
	if err != nil {
		c.SetMessage("failed to update tenancy")
		return c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpSetDbRole, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		Role: tenancy.Role(), Customer: tenancy.CustomerDisplay(), DbRole: dbRole})

	// publish notification
	t.PublishOp(tenancy, multitenancy.OpSetDbRole)

	// done
	return nil
}

func (t *TenancyController) Delete(ctx op_context.Context, id string, withDatabase bool, idIsDisplay ...bool) error {

	// setup
	c := ctx.TraceInMethod("TenancyController.Delete")
	defer ctx.TraceOutMethod()

	if withDatabase {
		ctx.SetGenericError(generic_error.New(generic_error.ErrorCodeUnsupported, "Database can not be droped through administrator. Use raw database tools for database dropping."))
		return c.SetError(errors.New("database can not be dropped"))
	}

	// find tenancy
	tenancy, err := t.Find(ctx, id, idIsDisplay...)
	if err != nil {
		return c.SetError(err)
	}

	// delete tenancy
	err = t.CRUD.Delete(ctx, &tenancy.TenancyDb)
	if err != nil {
		c.SetMessage("failed to delete tenancy")
		return c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpDelete, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		Role: tenancy.Role(), Customer: tenancy.CustomerDisplay()})

	// publish notification
	t.PublishOp(tenancy, multitenancy.OpDelete)

	// done
	return nil
}

func (t *TenancyController) ListIpAddresses(ctx op_context.Context, filter *db.Filter) ([]*multitenancy.TenancyIpAddressItem, int64, error) {

	// setup
	c := ctx.TraceInMethod("TenancyController.ListIpAddresses")
	defer ctx.TraceOutMethod()

	// construct join query
	queryBuilder := func() (db.JoinQuery, error) {
		return ctx.Db().Joiner().
			Join(&multitenancy.TenancyIpAddress{}, "tenancy_id").On(&multitenancy.TenancyDb{}, "id").
			Join(&multitenancy.TenancyDb{}, "customer_id").On(&customer.Customer{}, "id").
			Join(&multitenancy.TenancyDb{}, "pool_id").On(&pool.PoolBase{}, "id").
			Destination(&multitenancy.TenancyIpAddressItem{})
	}

	// invoke join
	var addresses []*multitenancy.TenancyIpAddressItem
	count, err := t.CRUD.Join(ctx, db.NewJoin(queryBuilder, "ListTenancyIpAddresses"), filter, &addresses)
	if err != nil {
		c.SetMessage("failed to find tenancy")
		return nil, 0, c.SetError(err)
	}

	// done
	return addresses, count, nil
}

func (t *TenancyController) DeleteIpAddress(ctx op_context.Context, id string, ipAddress string, tag string, idIsDisplay ...bool) error {

	// setup
	c := ctx.TraceInMethod("TenancyController.DeleteTenancyIpAddress")
	defer ctx.TraceOutMethod()

	// find
	tenancy, err := multitenancy.FindTenancy(t, ctx, id, idIsDisplay...)
	if err != nil {
		c.SetMessage("failed to find tenancy")
		return c.SetError(err)
	}

	// delete tenancy
	fields := db.Fields{"tenancy_id": tenancy.GetID(), "ip": ipAddress, "tag": tag}
	err = t.CRUD.DeleteByFields(ctx, fields, &multitenancy.TenancyIpAddress{})
	if err != nil {
		c.SetMessage("failed to delete tenancy IP address")
		return c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpDeleteIpAddress, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		Role: tenancy.Role(), Customer: tenancy.CustomerDisplay(), IpAddressTag: tag})

	// publish notification
	t.PublishOp(tenancy, multitenancy.OpDeleteIpAddress)

	// done
	return nil
}

func (t *TenancyController) AddIpAddress(ctx op_context.Context, id string, ipAddress string, tag string, idIsDisplay ...bool) error {

	// setup
	c := ctx.TraceInMethod("TenancyController.AddIpAddress", logger.Fields{"id": id, "ip": ipAddress, "tag": tag})
	defer ctx.TraceOutMethod()

	// find
	tenancy, err := multitenancy.FindTenancy(t, ctx, id, idIsDisplay...)
	if err != nil {
		c.SetMessage("failed to find tenancy")
		return c.SetError(err)
	}

	// delete tenancy
	obj := &multitenancy.TenancyIpAddress{}
	obj.InitObject()
	obj.TenancyId = tenancy.GetID()
	obj.Tag = tag
	obj.Ip = ipAddress
	_, err = t.CRUD.CreateDup(ctx, obj, true)
	if err != nil {
		c.SetMessage("failed to add IP address")
		return c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpAddIpAddress, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		Role: tenancy.Role(), Customer: tenancy.CustomerDisplay(), IpAddressTag: tag, IpAddress: ipAddress})

	// publish notification
	t.PublishOp(tenancy, multitenancy.OpAddIpAddress)

	// done
	return nil
}

func (t *TenancyController) SetPathBlocked(ctx op_context.Context, id string, blocked bool, mode multitenancy.TenancyBlockPathMode, idIsDisplay ...bool) error {

	// setup
	c := ctx.TraceInMethod("TenancyController.SetPathBlocked")
	defer ctx.TraceOutMethod()

	// find tenancy
	tenancy, err := t.Find(ctx, id, idIsDisplay...)
	if err != nil {
		return c.SetError(err)
	}

	// update fields
	switch mode {
	case multitenancy.TenancyBlockPathModeDefault:
		tenancy.BLOCK_PATH = blocked
	case multitenancy.TenancyBlockPathModeShadow:
		tenancy.BLOCK_SHADOW_PATH = blocked
	default:
		tenancy.BLOCK_PATH = blocked
		tenancy.BLOCK_SHADOW_PATH = blocked
	}
	fields := db.Fields{"block_path": tenancy.BLOCK_PATH, "block_shadow_path": tenancy.BLOCK_SHADOW_PATH}
	err = t.CRUD.Update(ctx, &tenancy.TenancyDb, fields)
	if err != nil {
		c.SetMessage("failed to update tenancy")
		return c.SetError(err)
	}

	// save oplog
	t.OpLog(ctx, multitenancy.OpSetPathBlocked, &multitenancy.OpLogTenancy{TenancyId: tenancy.GetID(),
		BlockPath: tenancy.IsBlockedPath(), BlockShadowPath: tenancy.IsBlockedShadowPath(), Customer: tenancy.CustomerDisplay()})

	// publish notification
	t.PublishOp(tenancy, multitenancy.OpSetPathBlocked)

	// done
	return nil
}
