package pool

import (
	"context"
	"errors"
	"net/http"

	"github.com/evgeniums/evgo/pkg/crud"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

const ErrorCodePoolNotFound = "pool_not_found"
const ErrorCodeServiceNotFound = "service_not_found"
const ErrorCodePoolNameConflict = "pool_name_conflict"
const ErrorCodeServiceNameConflict = "service_name_conflict"
const ErrorCodeServiceRoleConflict = "service_role_conflict"
const ErrorCodePoolServiceBindingsExist = "pool_service_bindings_exist"
const ErrorCodeNoServiceWithRole = "no_service_with_role"
const ErrorCodeInvalidServiceConfiguration = "invalid_service_configuration"
const ErrorCodeServiceInitializationFailed = "service_initialization_failed"
const ErrorCodePoolServiceBoundToPool = "service_bound_to_pool"
const ErrorCodeServiceNotActive = "service_not_active"
const ErrorCodePoolNotActive = "pool_not_active"
const ErrorCodeCreateServiceDatabaseFailed = "create_service_database_failed"

var ErrorDescriptions = map[string]string{
	ErrorCodePoolNotFound:                "Pool not found",
	ErrorCodeServiceNotFound:             "Service not found",
	ErrorCodePoolNameConflict:            "Pool with such name already exists, choose another name",
	ErrorCodeServiceNameConflict:         "Service with such name already exists, choose another name",
	ErrorCodeServiceRoleConflict:         "Pool already has service for that role",
	ErrorCodePoolServiceBindingsExist:    "Can't delete pool with services. First, remove all services from the pool",
	ErrorCodePoolServiceBoundToPool:      "Can't delete service bound to pool. First, remove the services from all pools",
	ErrorCodeNoServiceWithRole:           "Pool does not include service with requested role",
	ErrorCodeInvalidServiceConfiguration: "Invalid configuration of service in the pool",
	ErrorCodeServiceInitializationFailed: "Failed to connect to service",
	ErrorCodeServiceNotActive:            "Service not active. First, activate corresponsing service",
	ErrorCodePoolNotActive:               "Pool not active. First, activate corresponsing pool",
	ErrorCodeCreateServiceDatabaseFailed: "Failed to create database for pool service",
}

var ErrorHttpCodes = map[string]int{
	ErrorCodePoolNotFound:                http.StatusNotFound,
	ErrorCodeServiceNotFound:             http.StatusNotFound,
	ErrorCodeInvalidServiceConfiguration: http.StatusInternalServerError,
	ErrorCodeServiceInitializationFailed: http.StatusInternalServerError,
	ErrorCodeServiceNotActive:            http.StatusInternalServerError,
	ErrorCodePoolNotActive:               http.StatusInternalServerError,
	ErrorCodeCreateServiceDatabaseFailed: http.StatusInternalServerError,
}

type PoolController interface {
	AddPool(sctx context.Context, pool Pool) (Pool, error)
	FindPool(sctx context.Context, id string, idIsName ...bool) (Pool, error)
	UpdatePool(sctx context.Context, id string, fields db.Fields, idIsName ...bool) (Pool, error)
	DeletePool(sctx context.Context, id string, idIsName ...bool) error
	GetPools(sctx context.Context, filter *db.Filter) ([]*PoolBase, int64, error)

	AddService(sctx context.Context, service PoolService) (PoolService, error)
	FindService(sctx context.Context, id string, idIsName ...bool) (PoolService, error)
	UpdateService(sctx context.Context, id string, fields db.Fields, idIsName ...bool) (PoolService, error)
	DeleteService(sctx context.Context, id string, idIsName ...bool) error
	GetServices(sctx context.Context, filter *db.Filter) ([]*PoolServiceBase, int64, error)

	AddServiceToPool(sctx context.Context, poolId string, serviceId string, role string, idIsName ...bool) error
	RemoveServiceFromPool(sctx context.Context, poolId string, role string, idIsName ...bool) error
	RemoveAllServicesFromPool(sctx context.Context, poolId string, idIsName ...bool) error
	RemoveServiceFromAllPools(sctx context.Context, id string, idIsName ...bool) error

	GetPoolBindings(sctx context.Context, id string, idIsName ...bool) ([]*PoolServiceBinding, error)
	GetServiceBindings(sctx context.Context, id string, idIsName ...bool) ([]*PoolServiceBinding, error)
}

func NewPoolController(crud crud.CRUD) *PoolControllerBase {
	c := &PoolControllerBase{}
	c.CRUD = crud
	return c
}

type PoolControllerBase struct {
	CRUD crud.CRUD
}

func fieldName(idIsName ...bool) string {
	fieldName := "id"
	if utils.OptionalArg(false, idIsName...) {
		fieldName = "name"
	}
	return fieldName
}

func (m *PoolControllerBase) OpLog(sctx context.Context, operation string, oplog *OpLogPool) {
	oplog.SetOperation(operation)
	ctx := op_context.OpContext[op_context.Context](sctx)
	ctx.Oplog(oplog)
}

func (m *PoolControllerBase) AddPool(sctx context.Context, pool Pool) (Pool, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolController.AddPool", logger.Fields{"name": pool.Name()})
	defer ctx.TraceOutMethod()

	// check if pool name is unique
	err := m.CheckPoolNameUnique(sctx, pool.Name())
	if err != nil {
		return nil, c.SetError(err)
	}

	// create pool
	pool.InitObject()
	err = m.CRUD.Create(sctx, pool)
	if err != nil {
		return nil, c.SetError(err)
	}

	// save oplog
	m.OpLog(sctx, "add_pool", &OpLogPool{PoolId: pool.GetID(), PoolName: pool.Name()})

	// done
	return pool, nil
}

func (m *PoolControllerBase) FindPool(sctx context.Context, id string, idIsName ...bool) (Pool, error) {
	field := fieldName(idIsName...)
	ctx := op_context.OpContext[op_context.Context](sctx)
	pool, err := crud.FindByField(m.CRUD, sctx, "PoolController.FindPool", field, id, &PoolBase{})
	if err != nil {
		return nil, err
	}
	if pool == nil {
		ctx.SetGenericErrorCode(ErrorCodePoolNotFound)
		return nil, nil
	}
	return pool, nil
}

func (m *PoolControllerBase) CheckPoolNameUnique(sctx context.Context, name interface{}) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolController.CheckPoolNameUnique", logger.Fields{"name": name})
	defer ctx.TraceOutMethod()

	filter := db.NewFilter()
	filter.AddField("name", name)
	exists, err := m.CRUD.Exists(sctx, filter, &PoolBase{})
	if err != nil {
		c.SetMessage("failed to check existence of pool with desired name")
		return c.SetError(err)
	}
	if exists {
		ctx.SetGenericErrorCode(ErrorCodePoolNameConflict)
		return c.SetError(errors.New("pool with desired name exists"))
	}
	return nil
}

func (m *PoolControllerBase) UpdatePool(sctx context.Context, id string, fields db.Fields, idIsName ...bool) (Pool, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolController.UpdatePool", logger.Fields{"pool": id, "use_name": utils.OptionalArg(false, idIsName...)})
	defer ctx.TraceOutMethod()

	// check if name is unique
	if name, found := fields["name"]; found {
		err := m.CheckPoolNameUnique(sctx, name)
		if err != nil {
			return nil, c.SetError(err)
		}
	}

	// update
	field := fieldName(idIsName...)
	obj, err := crud.FindUpdate(m.CRUD, sctx, "PoolController.FindUpdatePool", field, id, fields, &PoolBase{}, logger.Fields{field: id})
	if err != nil {
		return nil, err
	}
	if obj == nil {
		ctx.SetGenericErrorCode(ErrorCodePoolNotFound)
		return nil, c.SetError(errors.New("pool not found"))
	}

	// write oplog
	hasActivate, activateOp, hasOtherFields := CheckIfActivate(fields)
	if hasActivate {
		m.OpLog(sctx, utils.ConcatStrings(activateOp, "_pool"), &OpLogPool{ServiceId: obj.GetID(), ServiceName: obj.Name()})
	}
	if hasOtherFields {
		m.OpLog(sctx, "update_pool", &OpLogPool{ServiceId: obj.GetID(), ServiceName: obj.Name()})
	}

	// find updated pool
	p, err := m.FindPool(sctx, obj.GetID())
	if err != nil {
		return nil, err
	}
	if p == nil {
		ctx.SetGenericErrorCode(ErrorCodePoolNotFound)
		return nil, c.SetError(errors.New("pool not found"))
	}

	// done
	return p, nil
}

func (m *PoolControllerBase) DeletePool(sctx context.Context, id string, idIsName ...bool) error {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolController.DeletePool")
	defer ctx.TraceOutMethod()

	poolId, err := m.PoolId(c, sctx, id, idIsName...)
	if err != nil {
		return err
	}
	if poolId == "" {
		ctx.ClearError()
		return nil
	}

	filter := db.NewFilter()
	filter.AddField("pool_id", poolId)
	exists, err := m.CRUD.Exists(sctx, filter, &PoolServiceAssociationBase{})
	if err != nil {
		c.SetMessage("failed to check associations")
		return c.SetError(err)
	}
	if exists {
		ctx.SetGenericErrorCode(ErrorCodePoolServiceBindingsExist)
		return c.SetError(errors.New("can not delete pool with services, remove all service bindings first"))
	}

	err = crud.Delete(m.CRUD, sctx, "PoolController.DeletePool", "id", poolId, &PoolBase{}, logger.Fields{"id": id})
	if err != nil {
		return err
	}

	o := &OpLogPool{PoolId: poolId}
	if utils.OptionalArg(false, idIsName...) {
		o.PoolName = id
	}
	m.OpLog(sctx, "delete_pool", o)
	return nil
}

func (m *PoolControllerBase) AddService(sctx context.Context, service PoolService) (PoolService, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolController.AddService", logger.Fields{"name": service.Name()})
	defer ctx.TraceOutMethod()

	err := m.CheckServiceNameUnique(sctx, service.Name())
	if err != nil {
		return nil, c.SetError(err)
	}

	service.InitObject()
	err = m.CRUD.Create(sctx, service)
	if err != nil {
		return nil, err
	}

	m.OpLog(sctx, "add_service", &OpLogPool{ServiceId: service.GetID(), ServiceName: service.Name()})

	return service, nil
}

func (m *PoolControllerBase) FindService(sctx context.Context, id string, idIsName ...bool) (PoolService, error) {
	field := fieldName(idIsName...)
	ctx := op_context.OpContext[op_context.Context](sctx)
	service, err := crud.FindByField(m.CRUD, sctx, "PoolController.FindService", field, id, &PoolServiceBase{})
	if err != nil {
		return nil, err
	}
	if service == nil {
		ctx.SetGenericErrorCode(ErrorCodeServiceNotFound)
		return nil, err
	}
	return service, nil
}

func (m *PoolControllerBase) CheckServiceNameUnique(sctx context.Context, name interface{}) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolController.CheckServiceNameUnique", logger.Fields{"name": name})
	defer ctx.TraceOutMethod()

	filter := db.NewFilter()
	filter.AddField("name", name)
	exists, err := m.CRUD.Exists(sctx, filter, &PoolServiceBase{})
	if err != nil {
		return c.SetError(err)
	}
	if exists {
		ctx.SetGenericErrorCode(ErrorCodeServiceNameConflict)
		return c.SetError(errors.New("service with such name exists"))
	}

	return nil
}

func CheckIfActivate(fields db.Fields) (bool, string, bool) {
	if db.IsFieldSet(fields, "active") {
		activate, ok := fields["active"].(bool)
		if ok {
			op := "activate"
			if !activate {
				op = "deactivate"
			}
			return true, op, len(fields) > 1
		}
	}
	return false, "", true
}

func (m *PoolControllerBase) UpdateService(sctx context.Context, id string, fields db.Fields, idIsName ...bool) (PoolService, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolController.UpdateService", logger.Fields{"service": id, "use_name": utils.OptionalArg(false, idIsName...)})
	defer ctx.TraceOutMethod()

	// check if name is unique
	if name, found := fields["name"]; found {
		err := m.CheckServiceNameUnique(sctx, name)
		if err != nil {
			return nil, c.SetError(err)
		}
	}

	// update
	idField := fieldName(idIsName...)
	obj, err := crud.FindUpdate(m.CRUD, sctx, "PoolController.FindUpdateService", idField, id, fields, &PoolServiceBase{}, logger.Fields{idField: id})
	if err != nil {
		return nil, c.SetError(err)
	}
	if obj == nil {
		ctx.SetGenericErrorCode(ErrorCodeServiceNotFound)
		return nil, c.SetError(errors.New("service not found"))
	}

	// write oplog
	hasActivate, activateOp, hasOtherFields := CheckIfActivate(fields)
	if hasActivate {
		m.OpLog(sctx, utils.ConcatStrings(activateOp, "_service"), &OpLogPool{ServiceId: obj.GetID(), ServiceName: obj.Name()})
	}
	if hasOtherFields {
		m.OpLog(sctx, "update_service", &OpLogPool{ServiceId: obj.GetID(), ServiceName: obj.Name()})
	}

	// find updated service
	s, err := m.FindService(sctx, obj.GetID())
	if err != nil {
		return nil, err
	}
	if s == nil {
		ctx.SetGenericErrorCode(ErrorCodeServiceNotFound)
		return nil, c.SetError(errors.New("service not found"))
	}

	// done
	return s, nil
}

func (m *PoolControllerBase) DeleteService(sctx context.Context, id string, idIsName ...bool) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolController.DeleteService")
	defer ctx.TraceOutMethod()

	serviceId, err := m.ServiceId(c, sctx, id, idIsName...)
	if err != nil {
		return err
	}
	if serviceId == "" {
		ctx.ClearError()
		return nil
	}

	filter := db.NewFilter()
	filter.AddField("service_id", serviceId)
	exists, err := m.CRUD.Exists(sctx, filter, &PoolServiceAssociationBase{})
	if err != nil {
		c.SetMessage("failed to check associations")
		return c.SetError(err)
	}
	if exists {
		ctx.SetGenericErrorCode(ErrorCodePoolServiceBoundToPool)
		return c.SetError(errors.New("can not delete service bound to pools, remove all service bindings first"))
	}

	err = crud.Delete(m.CRUD, sctx, "PoolController.DeleteService", "id", serviceId, &PoolServiceBase{}, logger.Fields{"id": id})
	if err != nil {
		return err
	}

	o := &OpLogPool{ServiceId: serviceId}
	if utils.OptionalArg(false, idIsName...) {
		o.ServiceName = id
	}
	m.OpLog(sctx, "delete_service", o)
	return nil
}

func (p *PoolControllerBase) GetPools(sctx context.Context, filter *db.Filter) ([]*PoolBase, int64, error) {
	var pools []*PoolBase
	count, err := crud.List(p.CRUD, sctx, "", filter, &pools)
	if err != nil {
		return nil, 0, err
	}
	return pools, count, nil
}

func (p *PoolControllerBase) GetServices(sctx context.Context, filter *db.Filter) ([]*PoolServiceBase, int64, error) {
	var services []*PoolServiceBase
	count, err := crud.List(p.CRUD, sctx, "PoolController.GetServices", filter, &services)
	if err != nil {
		return nil, 0, err
	}
	return services, count, nil
}

func (m *PoolControllerBase) AddServiceToPool(sctx context.Context, poolId string, serviceId string, role string, idIsName ...bool) error {

	field := fieldName(idIsName...)

	ctx := op_context.OpContext[op_context.Context](sctx)
	ctx.AddLoggerFields(logger.Fields{utils.ConcatStrings("pool_", field): poolId, utils.ConcatStrings("service_", field): serviceId, "role": role})

	c := ctx.TraceInMethod("PoolController.AddServiceToPool")
	defer ctx.TraceOutMethod()

	// find pool
	pool, err := m.FindPool(sctx, poolId, idIsName...)
	if err != nil {
		c.SetMessage("failed to find pool")
		return c.SetError(err)
	}
	if pool == nil {
		return c.SetError(errors.New("pool not found"))
	}

	// find service
	service, err := m.FindService(sctx, serviceId, idIsName...)
	if err != nil {
		c.SetMessage("failed to find service")
		return c.SetError(err)
	}
	if service == nil {
		return c.SetError(errors.New("unknown service"))
	}

	// check if name is unique
	filter := db.NewFilter()
	filter.AddField("pool_id", pool.GetID())
	filter.AddField("role", role)
	exists, err := m.CRUD.Exists(sctx, filter, &PoolServiceAssociationBase{})
	if err != nil {
		c.SetMessage("failed to check existence of pool service binding")
		return c.SetError(err)
	}
	if exists {
		ctx.SetGenericErrorCode(ErrorCodeServiceRoleConflict)
		return c.SetError(errors.New("pool already has service with such role"))
	}

	// create association
	association := &PoolServiceAssociationBase{}
	association.InitObject()
	association.POOL_ID = pool.GetID()
	association.SERVICE_ID = service.GetID()
	association.ROLE = role
	err = m.CRUD.Create(sctx, association)
	if err != nil {
		c.SetMessage("failed to save association in database")
		return c.SetError(err)
	}

	// add oplog
	m.OpLog(sctx, "add_service_to_pool", &OpLogPool{ServiceId: service.GetID(), ServiceName: service.Name(),
		PoolId: pool.GetID(), PoolName: pool.Name(),
		Role: role,
	})

	// done
	return nil
}

func (m *PoolControllerBase) PoolId(c op_context.CallContext, sctx context.Context, id string, idIsName ...bool) (string, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	if !utils.OptionalArg(false, idIsName...) {
		ctx.SetLoggerField("pool_id", id)
		return id, nil
	}

	ctx.SetLoggerField("pool_name", id)
	pool, err := m.FindPool(sctx, id, true)
	if err != nil {
		c.SetMessage("failed to find pool")
		return "", c.SetError(err)
	}
	if pool == nil {
		return "", nil
	}
	pId := pool.GetID()
	ctx.SetLoggerField("pool_id", pId)
	return pId, nil
}

func (m *PoolControllerBase) ServiceId(c op_context.CallContext, sctx context.Context, id string, idIsName ...bool) (string, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	if !utils.OptionalArg(false, idIsName...) {
		ctx.SetLoggerField("service_id", id)
		return id, nil
	}

	ctx.SetLoggerField("service_name", id)
	service, err := m.FindService(sctx, id, true)
	if err != nil {
		c.SetMessage("failed to find service")
		return "", c.SetError(err)
	}
	if service == nil {
		return "", nil
	}
	pId := service.GetID()
	ctx.SetLoggerField("service_id", pId)
	return pId, nil
}

func (m *PoolControllerBase) RemoveServiceFromPool(sctx context.Context, id string, role string, idIsName ...bool) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolController.RemoveServiceFromPool", logger.Fields{"role": role})
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	poolId, err := m.PoolId(c, sctx, id, idIsName...)
	if err != nil {
		return err
	}
	if poolId == "" {
		ctx.ClearError()
		return nil
	}

	association := &PoolServiceAssociationBase{}
	fields := db.Fields{"pool_id": poolId, "role": role}
	found, err := m.CRUD.Read(sctx, fields, association)
	if err != nil {
		c.SetMessage("failed to find association")
		return err
	}
	if !found {
		return nil
	}

	err = m.CRUD.Delete(sctx, association)
	if err != nil {
		c.SetMessage("failed to delete association")
		return err
	}

	o := &OpLogPool{PoolId: poolId, Role: role}
	if utils.OptionalArg(false, idIsName...) {
		o.PoolName = id
	}
	m.OpLog(sctx, "remove_service_from_pool", o)
	return nil
}

func (m *PoolControllerBase) RemoveAllServicesFromPool(sctx context.Context, id string, idIsName ...bool) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolController.RemoveAllServicesFromPool")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	poolId, err := m.PoolId(c, sctx, id, idIsName...)
	if err != nil {
		return err
	}
	if poolId == "" {
		ctx.ClearError()
		return nil
	}

	fields := db.Fields{"pool_id": poolId}
	err = m.CRUD.DeleteByFields(sctx, fields, &PoolServiceAssociationBase{})
	if err != nil {
		return err
	}

	o := &OpLogPool{PoolId: poolId}
	if utils.OptionalArg(false, idIsName...) {
		o.PoolName = id
	}
	m.OpLog(sctx, "remove_all_services_from_pool", o)
	return nil
}

func (m *PoolControllerBase) RemoveServiceFromAllPools(sctx context.Context, id string, idIsName ...bool) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolController.RemoveServiceFromAllPools")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	serviceId, err := m.ServiceId(c, sctx, id, idIsName...)
	if err != nil {
		return err
	}
	if serviceId == "" {
		ctx.ClearError()
		return nil
	}

	fields := db.Fields{"service_id": serviceId}
	err = m.CRUD.DeleteByFields(sctx, fields, &PoolServiceAssociationBase{})
	if err != nil {
		return err
	}

	o := &OpLogPool{ServiceId: serviceId}
	if utils.OptionalArg(false, idIsName...) {
		o.ServiceName = id
	}
	m.OpLog(sctx, "remove_service_from_all_pools", o)
	return nil
}

func (p *PoolControllerBase) GetPoolBindings(sctx context.Context, id string, idIsName ...bool) ([]*PoolServiceBinding, error) {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	var services []*PoolServiceBinding
	c := ctx.TraceInMethod("PoolController.GetPoolBindings")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()
	ctx.SetLoggerField("pool_id", id)

	// adjust pool ID
	poolId, err := p.PoolId(c, sctx, id, idIsName...)
	if err != nil {
		return nil, err
	}
	if poolId == "" {
		c.Logger().Warn("pool not found")
		ctx.ClearError()
		return nil, nil
	}

	// construct join query
	queryBuilder := func() (db.JoinQuery, error) {
		return ctx.Db().Joiner().
			Join(&PoolServiceAssociationBase{}, "pool_id").On(&PoolBase{}, "id").
			Join(&PoolServiceAssociationBase{}, "service_id").On(&PoolServiceBase{}, "id").
			Destination(&PoolServiceBinding{})
	}

	// invoke join
	filter := db.NewFilter()
	filter.AddField("pools.id", poolId)
	_, err = p.CRUD.Join(sctx, db.NewJoin(queryBuilder, "GetPoolBindings"), filter, &services)
	if err != nil {
		return nil, err
	}

	// done
	return services, nil
}

func (p *PoolControllerBase) GetServiceBindings(sctx context.Context, id string, idIsName ...bool) ([]*PoolServiceBinding, error) {

	// setup
	var services []*PoolServiceBinding
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolController.GetServiceBindings")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// adjust service ID
	serviceId, err := p.ServiceId(c, sctx, id, idIsName...)
	if err != nil {
		return nil, err
	}
	if serviceId == "" {
		ctx.ClearError()
		return nil, nil
	}

	// construct join query
	queryBuilder := func() (db.JoinQuery, error) {
		return ctx.Db().Joiner().
			Join(&PoolServiceAssociationBase{}, "service_id").On(&PoolServiceBase{}, "id").
			Join(&PoolServiceAssociationBase{}, "pool_id").On(&PoolBase{}, "id").
			Destination(&PoolServiceBinding{})
	}

	// invoke join
	filter := db.NewFilter()
	filter.AddField("pool_services.id", serviceId)
	_, err = p.CRUD.Join(sctx, db.NewJoin(queryBuilder, "GetServiceBindings"), filter, &services)
	if err != nil {
		return nil, err
	}

	// done
	return services, nil
}

func LoadPool(ctrl PoolController, sctx context.Context, id string, idIsName ...bool) (Pool, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("LoadPool")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	pool, err := ctrl.FindPool(sctx, id, idIsName...)
	if err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, nil
	}

	services, err := ctrl.GetPoolBindings(sctx, pool.GetID())
	if err != nil {
		return nil, err
	}

	pool.SetServices(services)
	return pool, nil
}

func ActivatePool(ctrl PoolController, sctx context.Context, id string, idIsName ...bool) (Pool, error) {
	fields := db.Fields{"active": true}
	return ctrl.UpdatePool(sctx, id, fields, idIsName...)
}

func DeactivatePool(ctrl PoolController, sctx context.Context, id string, idIsName ...bool) (Pool, error) {
	fields := db.Fields{"active": false}
	return ctrl.UpdatePool(sctx, id, fields, idIsName...)
}

func ActivateService(ctrl PoolController, sctx context.Context, id string, idIsName ...bool) (PoolService, error) {
	fields := db.Fields{"active": true}
	return ctrl.UpdateService(sctx, id, fields, idIsName...)
}

func DeactivateService(ctrl PoolController, sctx context.Context, id string, idIsName ...bool) (PoolService, error) {
	fields := db.Fields{"active": false}
	return ctrl.UpdateService(sctx, id, fields, idIsName...)
}
