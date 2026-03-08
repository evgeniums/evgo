package tenancy_manager

import (
	"context"

	"github.com/evgeniums/evgo/pkg/cache"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
)

type TenancyBaseData struct {
	multitenancy.TenancyDb

	db.WithDBBase
	Cache          cache.Cache
	Pool           pool.Pool
	Customer       *customer.Customer
	TenancyManager *TenancyManager
}

type TenancyBase struct {
	TenancyBaseData
}

func NewTenancy(manager *TenancyManager) *TenancyBase {
	t := &TenancyBase{}
	t.TenancyManager = manager
	return t
}

func NewContextPathOnly(tenancyPath string, fromCtx ...op_context.Context) *multitenancy.TenancyContextBase {
	c := multitenancy.NewContext(fromCtx...)
	tenancy := &TenancyBase{}
	tenancy.SetPath(tenancyPath)
	c.SetTenancy(tenancy)
	return c
}

func (d *TenancyBase) IsActive() bool {
	return d.TenancyDb.IsActive() && !d.Customer.IsBlocked() && d.TenancyBaseData.Pool.IsActive()
}

func (t *TenancyBase) Pool() pool.Pool {
	return t.TenancyBaseData.Pool
}

func (t *TenancyBase) Cache() cache.Cache {
	return t.TenancyBaseData.Cache
}

func (t *TenancyBase) SetCache(c cache.Cache) {
	t.TenancyBaseData.Cache = c
}

func (t *TenancyBase) Init(sctx context.Context, data *multitenancy.TenancyDb) error {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("TenancyBase.ConnectDatabase", logger.Fields{"customer": t.CUSTOMER_ID, "role": t.ROLE})
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	t.TenancyDb = *data

	// find customer
	t.Customer, err = t.TenancyManager.Customers.Find(sctx, data.CUSTOMER_ID)
	if err != nil {
		c.SetMessage("failed to find customer")
		return err
	}
	if t.Customer == nil {
		c.SetMessage("failed to find customer")
		return err
	}
	c.SetLoggerField("tenancy", multitenancy.TenancyDisplay(t))

	// check if tenancy is active
	if !t.TenancyDb.IsActive() {
		c.Logger().Info("skipping tenancy initialization because it is not active")
		return nil
	}

	// init tenancy resources

	// set tenancy cache
	t.SetCache(ctx.Cache())

	// check if pool exists
	t.TenancyBaseData.Pool, err = t.TenancyManager.Pools.Pool(data.POOL_ID)
	if err != nil {
		ctx.SetGenericErrorCode(pool.ErrorCodePoolNotFound)
		c.SetMessage("unknown pool")
		return err
	}
	if !t.TenancyBaseData.Pool.IsActive() {
		c.Logger().Warn("skipping tenancy initialization pool is not active", logger.Fields{"pool": t.TenancyBaseData.Pool.Name()})
		return nil
	}

	// init database
	err = t.ConnectDatabase(sctx)
	if err != nil {
		return err
	}

	// check tenancy database
	err = multitenancy.CheckTenancyDatabase(sctx, t.Db(), t.GetID())
	if err != nil {
		multitenancy.CloseTenancyDb(t)
		return err
	}

	// done
	return nil
}

func (t *TenancyBase) ConnectDatabase(sctx context.Context, newDb ...bool) error {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("TenancyBase.ConnectDatabase", logger.Fields{"customer": t.CUSTOMER_ID, "role": t.ROLE})
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// connect to database
	dbRole := t.DbRole()
	if dbRole == "" {
		dbRole = multitenancy.TENANCY_DATABASE_ROLE
	}
	db, err := pool.ConnectDatabaseService(sctx, t.Pool(), dbRole, t.DbName(), newDb...)
	if err != nil {
		return err
	}

	// done
	t.WithDBBase.Init(db)
	return nil
}

func (t *TenancyBase) CustomerDisplay() string {
	if t.Customer == nil {
		return ""
	}
	return t.Customer.Display()
}
