package pool_api_test

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/evgeniums/evgo/pkg/admin"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/app_with_pools"
	"github.com/evgeniums/evgo/pkg/pool/pool_api/pool_client"
	"github.com/evgeniums/evgo/pkg/pool/pool_api/pool_service"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/test/api_test"
	"github.com/evgeniums/evgo/test/pool_api_test/pool_test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _, testBasePath, _, _ = runtime.Caller(0)
var testDir = filepath.Dir(testBasePath)

func dbModels() []interface{} {
	return utils.ConcatSlices([]interface{}{}, admin.DbModels(), pool.DbModels())
}

func initTest(t *testing.T) *pool_test_utils.PoolTestContext {

	var appWithPools *app_with_pools.AppWithPoolsBase

	buildApp := func(t *testing.T, buildConfig *app_context.BuildConfig) app_context.Context {
		appWithPools = app_with_pools.New(buildConfig)
		return appWithPools
	}
	initApp := func(t *testing.T, app app_context.Context, configFile string, args []string, configType ...string) error {
		a, ok := app.(*app_with_pools.AppWithPoolsBase)
		require.True(t, ok)
		opCtx, sctx, err := a.InitWithArgs(configFile, args, configType...)
		if opCtx != nil {
			opCtx.Close(sctx)
		}
		return err
	}
	test_utils.SetAppHandlers(buildApp, initApp)

	ctx := &pool_test_utils.PoolTestContext{}
	ctx.TestContext = api_test.InitTest(t, "pool", testDir, dbModels())
	ctx.RemotePoolController = pool_client.NewPoolClient(ctx.RestApiClient)
	ctx.LocalPoolController = appWithPools.Pools().PoolController()

	poolService := pool_service.NewPoolService(ctx.LocalPoolController)
	api_server.AddServiceToServer(ctx.Server.ApiServer(), poolService)

	return ctx
}

func TestInit(t *testing.T) {
	ctx := initTest(t)
	ctx.Close()
}

func addPool(t *testing.T, ctx *pool_test_utils.PoolTestContext, poolName ...string) pool.Pool {
	return pool_test_utils.AddPool(t, ctx, poolName...)
}

func TestAddDeletePool(t *testing.T) {
	ctx := initTest(t)
	defer ctx.Close()

	p1 := addPool(t, ctx)
	p2 := addPool(t, ctx, "pool2")

	filter := db.NewFilter()
	filter.SortField = "name"
	filter.SortDirection = db.SORT_ASC
	pools, _, err := ctx.RemotePoolController.GetPools(ctx.ClientOpCtx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, len(pools))
	assert.Equal(t, p1.Name(), pools[0].Name())
	assert.Equal(t, p2.Name(), pools[1].Name())

	err = ctx.RemotePoolController.DeletePool(ctx.ClientOpCtx, p1.Name(), true)
	require.NoError(t, err)
	pools, _, err = ctx.RemotePoolController.GetPools(ctx.ClientOpCtx, nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(pools))
	assert.Equal(t, p2.Name(), pools[0].Name())

	err = ctx.RemotePoolController.DeletePool(ctx.ClientOpCtx, p2.GetID())
	require.NoError(t, err)
	pools, _, err = ctx.RemotePoolController.GetPools(ctx.ClientOpCtx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(pools))
}

func addService(t *testing.T, ctx *pool_test_utils.PoolTestContext, serviceName ...string) pool.PoolService {
	return pool_test_utils.AddService(t, ctx, pool_test_utils.DefaultServiceConfig(serviceName...))
}

func TestAddDeleteService(t *testing.T) {
	ctx := initTest(t)
	defer ctx.Close()

	s1 := addService(t, ctx)
	s2 := addService(t, ctx, "service2")

	filter := db.NewFilter()
	filter.SortField = "name"
	filter.SortDirection = db.SORT_ASC
	services, _, err := ctx.RemotePoolController.GetServices(ctx.ClientOpCtx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, len(services))
	assert.Equal(t, s1.Name(), services[0].Name())
	assert.Equal(t, s2.Name(), services[1].Name())
	assert.Equal(t, s1.GetID(), services[0].GetID())
	assert.Equal(t, s2.GetID(), services[1].GetID())

	err = ctx.RemotePoolController.DeleteService(ctx.ClientOpCtx, s1.Name(), true)
	require.NoError(t, err)
	services, _, err = ctx.RemotePoolController.GetServices(ctx.ClientOpCtx, nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(services))
	assert.Equal(t, s2.Name(), services[0].Name())

	err = ctx.RemotePoolController.DeleteService(ctx.ClientOpCtx, s2.GetID())
	require.NoError(t, err)
	services, _, err = ctx.RemotePoolController.GetServices(ctx.ClientOpCtx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(services))
}

func TestBindings(t *testing.T) {
	ctx := initTest(t)
	defer ctx.Close()

	ctx.AdminOp.Db().EnableDebug(true)

	// setup
	service := addService(t, ctx)
	service2 := addService(t, ctx, "service2")
	p := addPool(t, ctx)
	p2 := addPool(t, ctx, "pool2")
	role := "main_webservice"
	role2 := "pubsub"

	checkList := func(bindings []*pool.PoolServiceBinding) {
		require.Equal(t, 1, len(bindings))
		binding := bindings[0]
		assert.Equal(t, p.Name(), binding.PoolName)
		assert.Equal(t, p.GetID(), binding.PoolId)
		assert.Equal(t, service.Name(), binding.ServiceName)
		assert.Equal(t, service.GetID(), binding.ServiceId)
		assert.Equal(t, role, binding.Role())
		serviceB := service.(*pool.PoolServiceBase)
		assert.Equal(t, serviceB.PoolServiceBaseData, binding.PoolServiceBaseData)
	}

	checkList2 := func(bindings []*pool.PoolServiceBinding) {
		require.Equal(t, 2, len(bindings))
		binding0 := bindings[0]
		assert.Equal(t, p.Name(), binding0.PoolName)
		assert.Equal(t, p.GetID(), binding0.PoolId)
		assert.Equal(t, service.Name(), binding0.ServiceName)
		assert.Equal(t, service.GetID(), binding0.ServiceId)
		assert.Equal(t, role, binding0.Role())
		serviceB := service.(*pool.PoolServiceBase)
		assert.Equal(t, serviceB.PoolServiceBaseData, binding0.PoolServiceBaseData)

		binding1 := bindings[1]
		assert.Equal(t, p.Name(), binding1.PoolName)
		assert.Equal(t, p.GetID(), binding1.PoolId)
		assert.Equal(t, service2.Name(), binding1.ServiceName)
		assert.Equal(t, service2.GetID(), binding1.ServiceId)
		assert.Equal(t, role2, binding1.Role())
		serviceB = service2.(*pool.PoolServiceBase)
		assert.Equal(t, serviceB.PoolServiceBaseData, binding1.PoolServiceBaseData)
	}

	checkList3 := func(bindings []*pool.PoolServiceBinding) {
		require.Equal(t, 2, len(bindings))
		binding0 := bindings[0]
		assert.Equal(t, p.Name(), binding0.PoolName)
		assert.Equal(t, p.GetID(), binding0.PoolId)
		assert.Equal(t, service.Name(), binding0.ServiceName)
		assert.Equal(t, service.GetID(), binding0.ServiceId)
		assert.Equal(t, role, binding0.Role())
		serviceB := service.(*pool.PoolServiceBase)
		assert.Equal(t, serviceB.PoolServiceBaseData, binding0.PoolServiceBaseData)

		binding1 := bindings[1]
		assert.Equal(t, p2.Name(), binding1.PoolName)
		assert.Equal(t, p2.GetID(), binding1.PoolId)
		assert.Equal(t, service.Name(), binding1.ServiceName)
		assert.Equal(t, service.GetID(), binding1.ServiceId)
		assert.Equal(t, role, binding1.Role())
		serviceB = service.(*pool.PoolServiceBase)
		assert.Equal(t, serviceB.PoolServiceBaseData, binding1.PoolServiceBaseData)
	}

	// add service to pool
	err := ctx.RemotePoolController.AddServiceToPool(ctx.ClientOpCtx, p.GetID(), service.GetID(), role)
	require.NoError(t, err)

	// get services added to pool
	bindings, err := ctx.LocalPoolController.GetPoolBindings(ctx.AdminOpCtx, p.GetID())
	require.NoError(t, err)
	checkList(bindings)
	b1, _ := json.MarshalIndent(bindings[0], "", "  ")
	t.Logf("Pool services: \n\n%s\n\n", string(b1))

	// get pools the service is added to
	bindings, err = ctx.LocalPoolController.GetServiceBindings(ctx.AdminOpCtx, service.GetID())
	require.NoError(t, err)
	checkList(bindings)

	// try to add duplicate service to pool
	err = ctx.RemotePoolController.AddServiceToPool(ctx.ClientOpCtx, p.GetID(), service.GetID(), role)
	test_utils.CheckGenericError(t, err, pool.ErrorCodeServiceRoleConflict, "Pool already has service for that role")

	// try to add unknown service to pool
	err = ctx.RemotePoolController.AddServiceToPool(ctx.ClientOpCtx, p.GetID(), "unknown_id", role)
	test_utils.CheckGenericError(t, err, pool.ErrorCodeServiceNotFound, "Service not found")

	// try to add service to unknown pool
	err = ctx.RemotePoolController.AddServiceToPool(ctx.ClientOpCtx, "unknown_id", "unknown_id", role)
	test_utils.CheckGenericError(t, err, pool.ErrorCodePoolNotFound, "Pool not found")

	// try to remove pool with services
	err = ctx.RemotePoolController.DeletePool(ctx.ClientOpCtx, p.GetID())
	test_utils.CheckGenericError(t, err, pool.ErrorCodePoolServiceBindingsExist, "Can't delete pool with services. First, remove all services from the pool")

	// try to remove bound service
	err = ctx.RemotePoolController.DeleteService(ctx.ClientOpCtx, service.GetID())
	test_utils.CheckGenericError(t, err, pool.ErrorCodePoolServiceBoundToPool, "Can't delete service bound to pool. First, remove the services from all pools")

	// try to remove unknown service from pool
	err = ctx.RemotePoolController.RemoveServiceFromPool(ctx.ClientOpCtx, "unknown_id", role)
	assert.NoErrorf(t, err, "unknown services and/or unknown pools must be ignored")
	err = ctx.RemotePoolController.RemoveServiceFromPool(ctx.ClientOpCtx, p.GetID(), "unknown_role")
	assert.NoErrorf(t, err, "unknown services and/or unknown pools must be ignored")

	// remove service from pool
	err = ctx.RemotePoolController.RemoveServiceFromPool(ctx.ClientOpCtx, p.GetID(), role)
	assert.NoError(t, err)
	bindings, err = ctx.LocalPoolController.GetPoolBindings(ctx.AdminOpCtx, p.GetID())
	assert.NoError(t, err)
	assert.Equal(t, 0, len(bindings))

	// add service to pool again
	err = ctx.RemotePoolController.AddServiceToPool(ctx.ClientOpCtx, p.GetID(), service.GetID(), role)
	require.NoError(t, err)
	bindings, err = ctx.LocalPoolController.GetPoolBindings(ctx.AdminOpCtx, p.GetID())
	assert.NoError(t, err)
	checkList(bindings)

	// add second service to pool
	err = ctx.RemotePoolController.AddServiceToPool(ctx.ClientOpCtx, p.GetID(), service2.GetID(), role2)
	require.NoError(t, err)
	bindings, err = ctx.LocalPoolController.GetPoolBindings(ctx.AdminOpCtx, p.GetID())
	assert.NoError(t, err)
	checkList2(bindings)

	// try to remove all services from unknown pool
	err = ctx.RemotePoolController.RemoveAllServicesFromPool(ctx.ClientOpCtx, "unknown_id")
	assert.NoErrorf(t, err, "unknown pools must be ignored")
	bindings, err = ctx.LocalPoolController.GetPoolBindings(ctx.AdminOpCtx, p.GetID())
	assert.NoError(t, err)
	checkList2(bindings)

	// remove all services from pool
	err = ctx.RemotePoolController.RemoveAllServicesFromPool(ctx.ClientOpCtx, p.GetID())
	assert.NoError(t, err)
	bindings, err = ctx.LocalPoolController.GetPoolBindings(ctx.AdminOpCtx, p.GetID())
	assert.NoError(t, err)
	assert.Equal(t, 0, len(bindings))

	// add service to pool again
	err = ctx.RemotePoolController.AddServiceToPool(ctx.ClientOpCtx, p.GetID(), service.GetID(), role)
	require.NoError(t, err)
	bindings, err = ctx.LocalPoolController.GetPoolBindings(ctx.AdminOpCtx, p.GetID())
	assert.NoError(t, err)
	checkList(bindings)

	// add service to second pool
	err = ctx.RemotePoolController.AddServiceToPool(ctx.ClientOpCtx, p2.GetID(), service.GetID(), role)
	require.NoError(t, err)
	bindings, err = ctx.LocalPoolController.GetServiceBindings(ctx.AdminOpCtx, service.GetID())
	assert.NoError(t, err)
	checkList3(bindings)

	// try to remove unknown service from all pools
	err = ctx.RemotePoolController.RemoveServiceFromAllPools(ctx.ClientOpCtx, "unknown_id")
	assert.NoErrorf(t, err, "unknown services must be ignored")
	bindings, err = ctx.LocalPoolController.GetServiceBindings(ctx.AdminOpCtx, service.GetID())
	assert.NoError(t, err)
	checkList3(bindings)

	// remove service from all pools
	err = ctx.RemotePoolController.RemoveServiceFromAllPools(ctx.ClientOpCtx, service.Name(), true)
	assert.NoError(t, err)
	bindings, err = ctx.LocalPoolController.GetServiceBindings(ctx.AdminOpCtx, service.GetID())
	assert.NoError(t, err)
	assert.Equal(t, 0, len(bindings))
}

func TestUpdatePool(t *testing.T) {
	ctx := initTest(t)
	defer ctx.Close()

	p := addPool(t, ctx)

	// update and check pool
	fields := db.Fields{"name": "updated name", "long_name": "updated long_name", "description": "updated description", "active": true}
	updatedP, err := ctx.RemotePoolController.UpdatePool(ctx.ClientOpCtx, p.GetID(), fields)
	require.NoError(t, err)
	assert.Equal(t, "updated name", updatedP.Name())
	assert.Equal(t, "updated long_name", updatedP.LongName())
	assert.Equal(t, "updated description", updatedP.Description())
	assert.True(t, updatedP.IsActive())

	// find and check pool
	remotePool1, err := ctx.RemotePoolController.FindPool(ctx.ClientOpCtx, p.GetID())
	require.NoError(t, err)
	require.NotNil(t, remotePool1)
	assert.Equal(t, "updated name", remotePool1.Name())
	assert.Equal(t, "updated long_name", remotePool1.LongName())
	assert.Equal(t, "updated description", remotePool1.Description())
	assert.True(t, remotePool1.IsActive())

	// unknown pool
	delete(fields, "name")
	_, err = ctx.RemotePoolController.UpdatePool(ctx.ClientOpCtx, "someid", fields)
	require.Error(t, err)
	test_utils.CheckGenericError(t, err, pool.ErrorCodePoolNotFound, "Pool not found")

	// unknown field
	fields = db.Fields{"unknown_field": "try me"}
	_, err = ctx.RemotePoolController.UpdatePool(ctx.ClientOpCtx, p.GetID(), fields)
	require.Error(t, err)
	test_utils.CheckGenericError(t, err, generic_error.ErrorCodeFormat, "Invalid fields for update")

	// duplicate name
	fields = db.Fields{"name": "updated name"}
	_, err = ctx.RemotePoolController.UpdatePool(ctx.ClientOpCtx, p.GetID(), fields)
	require.Error(t, err)
	test_utils.CheckGenericError(t, err, pool.ErrorCodePoolNameConflict, "Pool with such name already exists, choose another name")
}

func TestUpdateService(t *testing.T) {
	ctx := initTest(t)
	defer ctx.Close()

	s := addService(t, ctx)

	// fiil fields
	fields := db.Fields{"name": "updated name", "long_name": "updated long_name", "description": "updated description", "active": false}
	fields["type_name"] = "new type"
	fields["secret1"] = "new secret 1"
	fields["secret2"] = "new secret 2"
	fields["provider"] = "new provider"
	fields["public_host"] = "new public host"
	fields["public_port"] = 1010
	fields["public_url"] = "new public url"
	fields["private_host"] = "new private host"
	fields["private_port"] = 2020
	fields["private_url"] = "new private url"
	fields["user"] = "new user"
	fields["parameter1"] = "new parameter 1"
	fields["parameter2"] = "new parameter 2"
	fields["parameter3"] = "new parameter 3"
	fields["parameter1_name"] = "new name of parameter 1"
	fields["parameter2_name"] = "new name of parameter 2"
	fields["parameter3_name"] = "new name of parameter 3"

	// update and check service
	updatedS, err := ctx.RemotePoolController.UpdateService(ctx.ClientOpCtx, s.GetID(), fields)
	require.NoError(t, err)
	assert.Equal(t, "updated name", updatedS.Name())
	assert.Equal(t, "updated long_name", updatedS.LongName())
	assert.Equal(t, "updated description", updatedS.Description())
	assert.Equal(t, "new type", updatedS.TypeName())
	assert.Equal(t, "new secret 1", updatedS.Secret1())
	assert.Equal(t, "new secret 2", updatedS.Secret2())
	assert.Equal(t, "new provider", updatedS.Provider())
	assert.Equal(t, "new public host", updatedS.PublicHost())
	assert.Equal(t, uint16(1010), updatedS.PublicPort())
	assert.Equal(t, "new public url", updatedS.PublicUrl())
	assert.Equal(t, "new private host", updatedS.PrivateHost())
	assert.Equal(t, uint16(2020), updatedS.PrivatePort())
	assert.Equal(t, "new private url", updatedS.PrivateUrl())
	assert.Equal(t, "new user", updatedS.User())
	assert.Equal(t, "new parameter 1", updatedS.Parameter1())
	assert.Equal(t, "new parameter 2", updatedS.Parameter2())
	assert.Equal(t, "new parameter 3", updatedS.Parameter3())
	assert.Equal(t, "new name of parameter 1", updatedS.Parameter1Name())
	assert.Equal(t, "new name of parameter 2", updatedS.Parameter2Name())
	assert.Equal(t, "new name of parameter 3", updatedS.Parameter3Name())
	assert.False(t, updatedS.IsActive())

	// find and check service
	remoteService1, err := ctx.RemotePoolController.FindService(ctx.ClientOpCtx, s.GetID())
	require.NoError(t, err)
	require.NotNil(t, remoteService1)
	assert.Equal(t, "updated name", remoteService1.Name())
	assert.Equal(t, "updated long_name", remoteService1.LongName())
	assert.Equal(t, "updated description", remoteService1.Description())
	assert.Equal(t, "new type", remoteService1.TypeName())
	assert.Equal(t, "new secret 1", remoteService1.Secret1())
	assert.Equal(t, "new secret 2", remoteService1.Secret2())
	assert.Equal(t, "new provider", remoteService1.Provider())
	assert.Equal(t, "new public host", remoteService1.PublicHost())
	assert.Equal(t, uint16(1010), remoteService1.PublicPort())
	assert.Equal(t, "new public url", remoteService1.PublicUrl())
	assert.Equal(t, "new private host", remoteService1.PrivateHost())
	assert.Equal(t, uint16(2020), remoteService1.PrivatePort())
	assert.Equal(t, "new private url", remoteService1.PrivateUrl())
	assert.Equal(t, "new user", remoteService1.User())
	assert.Equal(t, "new parameter 1", remoteService1.Parameter1())
	assert.Equal(t, "new parameter 2", remoteService1.Parameter2())
	assert.Equal(t, "new parameter 3", remoteService1.Parameter3())
	assert.Equal(t, "new name of parameter 1", remoteService1.Parameter1Name())
	assert.Equal(t, "new name of parameter 2", remoteService1.Parameter2Name())
	assert.Equal(t, "new name of parameter 3", remoteService1.Parameter3Name())
	assert.False(t, remoteService1.IsActive())

	// unknown service
	delete(fields, "name")
	_, err = ctx.RemotePoolController.UpdateService(ctx.ClientOpCtx, "someid", fields)
	require.Error(t, err)
	test_utils.CheckGenericError(t, err, pool.ErrorCodeServiceNotFound, "Service not found")

	// unknown field
	fields = db.Fields{"unknown_field": "try me"}
	_, err = ctx.RemotePoolController.UpdateService(ctx.ClientOpCtx, s.GetID(), fields)
	require.Error(t, err)
	test_utils.CheckGenericError(t, err, generic_error.ErrorCodeFormat, "Invalid fields for update")

	// duplicate name
	fields = db.Fields{"name": "updated name"}
	_, err = ctx.RemotePoolController.UpdateService(ctx.ClientOpCtx, s.GetID(), fields)
	require.Error(t, err)
	test_utils.CheckGenericError(t, err, pool.ErrorCodeServiceNameConflict, "Service with such name already exists, choose another name")
}
