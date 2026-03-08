package tenancy_api_test

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/evgeniums/evgo/pkg/admin"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/app_with_multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api/tenancy_client"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api/tenancy_service"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_manager"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api/pool_client"
	"github.com/evgeniums/evgo/pkg/pool/pool_api/pool_service"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_providers/pubsub_factory"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/test/api_test"
	"github.com/evgeniums/evgo/test/pool_api_test/pool_test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _, testBasePath, _, _ = runtime.Caller(0)
var testDir = filepath.Dir(testBasePath)

type InTenancySample struct {
	common.IDBase
	Field1 string
	Field2 int
}

type InTenancyItem struct {
	common.IDBase
	Field4 string
	Field5 bool
}

type PartitionedItem struct {
	common.ObjectWithMonthBase
	Field4 string
	Field5 int
}

func tenancyDbModels() *multitenancy.TenancyDbModels {
	models := &multitenancy.TenancyDbModels{}
	models.DbModels = []interface{}{&InTenancySample{}, &InTenancyItem{}}
	models.PartitionedDbModels = []interface{}{&PartitionedItem{}}
	return models
}

type SampleModel1 struct {
	common.ObjectBase
	Field1 string `gorm:"uniqueIndex"`
	Field2 string `gorm:"index"`
}

func dbModels() []interface{} {
	return utils.ConcatSlices([]interface{}{&SampleModel1{}}, admin.DbModels(), pool.DbModels(), customer.DbModels(), multitenancy.DbModels())
}

type TenancyTestContext struct {
	*pool_test_utils.PoolTestContext

	LocalCustomerManager *customer.Manager

	LocalTenancyController  multitenancy.TenancyController
	RemoteTenancyController *tenancy_client.TenancyClient

	AppWithTenancy *app_with_multitenancy.AppWithMultitenancyBase
}

func initContext(t *testing.T, newDb bool, configPrefix ...string) *TenancyTestContext {

	var appWithTenancy *app_with_multitenancy.AppWithMultitenancyBase

	buildApp := func(t *testing.T, buildConfig *app_context.BuildConfig) app_context.Context {
		appWithTenancy = app_with_multitenancy.NewApp(buildConfig, tenancyDbModels())
		return appWithTenancy
	}
	initApp := func(t *testing.T, app app_context.Context, configFile string, args []string, configType ...string) error {
		a, ok := app.(*app_with_multitenancy.AppWithMultitenancyBase)
		require.True(t, ok)
		opCtx, sctx, err := a.InitWithArgs(configFile, args, configType...)
		if opCtx != nil {
			opCtx.Close(sctx)
		}
		return err
	}
	test_utils.SetAppHandlers(buildApp, initApp)

	ctx := &TenancyTestContext{}
	ctx.PoolTestContext = &pool_test_utils.PoolTestContext{}
	ctx.TestContext = api_test.InitTest(t, utils.OptionalArg("tenancy", configPrefix...), testDir, dbModels(), newDb)
	require.NotNil(t, appWithTenancy)
	ctx.LocalPoolController = appWithTenancy.Pools().PoolController()
	ctx.RemotePoolController = pool_client.NewPoolClient(ctx.RestApiClient)
	ctx.LocalCustomerManager = customer.NewManager()
	ctx.LocalCustomerManager.Init(appWithTenancy.Validator())
	ctx.LocalTenancyController = appWithTenancy.Multitenancy().TenancyController()
	ctx.RemoteTenancyController = tenancy_client.NewTenancyClient(ctx.RestApiClient)

	poolService := pool_service.NewPoolService(appWithTenancy.Pools().PoolController())
	api_server.AddServiceToServer(ctx.Server.ApiServer(), poolService)

	tenancyService := tenancy_service.NewTenancyService(ctx.LocalTenancyController)
	api_server.AddServiceToServer(ctx.Server.ApiServer(), tenancyService)

	ctx.AppWithTenancy = appWithTenancy

	return ctx
}

func preparePools(t *testing.T, ctx *TenancyTestContext, names ...string) []pool.Pool {

	pools := make([]pool.Pool, len(names))
	for i, name := range names {
		pool := pool_test_utils.AddPool(t, ctx.PoolTestContext, name)
		pools[i] = pool
	}

	return pools
}

type DbConfig = test_utils.PostgresDbConfig

type TenancyServiceConfig struct {
	Name     string
	Type     string
	Provider string
	DbName   string
	DbConfig
}

func prepareServices(t *testing.T, ctx *TenancyTestContext, configs ...*TenancyServiceConfig) []pool.PoolService {

	services := make([]pool.PoolService, len(configs))
	for i, config := range configs {

		cfg := pool_test_utils.DefaultServiceConfig(config.Name)
		cfg.PROVIDER = config.Provider
		cfg.TYPE_NAME = config.Type
		cfg.DB_NAME = config.DbName
		cfg.PRIVATE_URL = ""
		cfg.PRIVATE_PORT = config.DbPort
		cfg.PRIVATE_HOST = config.DbHost
		cfg.USER = config.DbUser
		cfg.SECRET1 = config.DbPassword

		service := pool_test_utils.AddService(t, ctx.PoolTestContext, cfg)
		services[i] = service
	}

	return services
}

type TenancyPoolConfig struct {
	PoolName      string
	DbService     TenancyServiceConfig
	PubsubService TenancyServiceConfig
}

func preparePoolServices(t *testing.T, ctx *TenancyTestContext, config *TenancyPoolConfig) pool.Pool {

	pools := preparePools(t, ctx, config.PoolName)
	p := pools[0]

	services := prepareServices(t, ctx, &config.DbService, &config.PubsubService)

	require.NoError(t, ctx.RemotePoolController.AddServiceToPool(op_context.MakeOpContext(ctx.ClientOp), p.GetID(), services[0].GetID(), multitenancy.TENANCY_DATABASE_ROLE))
	require.NoError(t, ctx.RemotePoolController.AddServiceToPool(op_context.MakeOpContext(ctx.ClientOp), p.GetID(), services[1].GetID(), pool.TypePubsub))

	return p
}

func preparePoolAndServices(t *testing.T, activatePools bool) (p1 pool.Pool, poolConfig1 *TenancyPoolConfig, p2 pool.Pool, poolConfig2 *TenancyPoolConfig) {

	prepareCtx := initContext(t, true)

	pools1 := prepareCtx.AppWithTenancy.Pools()
	require.NotNil(t, pools1)
	require.Empty(t, pools1.Pools())
	selfPool1, err := pools1.SelfPool()
	assert.Error(t, err)
	assert.Nil(t, selfPool1)

	poolConfig1 = &TenancyPoolConfig{
		PoolName: "pool1",
	}
	poolConfig1.DbService = TenancyServiceConfig{Name: "database_service1", Type: pool.TypeDatabase, Provider: "sqlite"}
	poolConfig1.PubsubService = TenancyServiceConfig{Name: "pubsub_service1", Type: pool.TypePubsub, Provider: pubsub_factory.SingletonInmemProvider, DbName: "0"}
	p1 = preparePoolServices(t, prepareCtx, poolConfig1)

	poolConfig2 = &TenancyPoolConfig{
		PoolName: "pool2",
	}
	poolConfig2.DbService = TenancyServiceConfig{Name: "database_service2", Type: pool.TypeDatabase, Provider: "sqlite"}
	poolConfig2.PubsubService = TenancyServiceConfig{Name: "pubsub_service2", Type: pool.TypePubsub, Provider: pubsub_factory.SingletonInmemProvider, DbName: "1"}
	p2 = preparePoolServices(t, prepareCtx, poolConfig2)

	if activatePools {
		_, err = pool.ActivatePool(prepareCtx.RemotePoolController, op_context.MakeOpContext(prepareCtx.ClientOp), p1.GetID())
		require.NoError(t, err)
		_, err = pool.ActivatePool(prepareCtx.RemotePoolController, op_context.MakeOpContext(prepareCtx.ClientOp), p2.GetID())
		require.NoError(t, err)
	}

	prepareCtx.Close()

	return
}

func PrepareAppWithTenancies(t *testing.T, multiPoolConfig ...string) (multiPoolCtx *TenancyTestContext, singlePoolCtx *TenancyTestContext) {

	preparePoolAndServices(t, true)

	multiPoolCtx = initContext(t, false, multiPoolConfig...)
	singlePoolCtx = initContext(t, false, "tenancy_single")

	customer1, err := multiPoolCtx.LocalCustomerManager.Add(op_context.MakeOpContext(multiPoolCtx.AdminOp), "customer1", "12345678")
	require.NoError(t, err)
	require.NotNil(t, customer1)
	customer2, err := multiPoolCtx.LocalCustomerManager.Add(op_context.MakeOpContext(multiPoolCtx.AdminOp), "customer2", "12345678")
	require.NoError(t, err)
	require.NotNil(t, customer2)

	return
}

func TestInit(t *testing.T) {

	p1, poolConfig1, p2, poolConfig2 := preparePoolAndServices(t, false)

	allPoolsCtx := initContext(t, false)

	pools := allPoolsCtx.AppWithTenancy.Pools()
	require.NotNil(t, pools)
	require.NotEmpty(t, pools.Pools())
	require.Equal(t, 2, len(pools.Pools()))
	selfPool2, err := pools.SelfPool()
	assert.Error(t, err)
	assert.Nil(t, selfPool2)

	pool1, err := pools.PoolByName(poolConfig1.PoolName)
	require.NoError(t, err)
	require.NotEmpty(t, pool1)
	pool1, err = pools.Pool(p1.GetID())
	require.NoError(t, err)
	require.NotEmpty(t, pool1)
	nopool, err := pools.PoolByName("unknown_name")
	assert.Error(t, err)
	assert.Empty(t, nopool)
	dbService1, err := pool1.Service(multitenancy.TENANCY_DATABASE_ROLE)
	require.NoError(t, err)
	require.NotEmpty(t, dbService1)
	assert.Equal(t, "database_service1", dbService1.ServiceName)
	pubsubService1, err := pool1.Service(pool.TypePubsub)
	require.NoError(t, err)
	require.NotEmpty(t, pubsubService1)
	assert.Equal(t, "pubsub_service1", pubsubService1.ServiceName)

	pool2, err := pools.PoolByName(poolConfig2.PoolName)
	require.NoError(t, err)
	require.NotEmpty(t, pool2)
	pool2, err = pools.Pool(p2.GetID())
	require.NoError(t, err)
	require.NotEmpty(t, pool2)
	dbService2, err := pool2.Service(multitenancy.TENANCY_DATABASE_ROLE)
	require.NoError(t, err)
	require.NotEmpty(t, dbService2)
	assert.Equal(t, "database_service2", dbService2.ServiceName)
	pubsubService2, err := pool2.Service(pool.TypePubsub)
	require.NoError(t, err)
	require.NotEmpty(t, pubsubService2)
	assert.Equal(t, "pubsub_service2", pubsubService2.ServiceName)

	allPoolsCtx.Close()

	singlePoolCtx := initContext(t, false, "tenancy_single")

	poolsSingle := singlePoolCtx.AppWithTenancy.Pools()
	require.NotNil(t, poolsSingle)
	require.NotEmpty(t, poolsSingle.Pools())
	require.Equal(t, 1, len(poolsSingle.Pools()))
	selfPool, err := poolsSingle.SelfPool()
	require.NoError(t, err)
	require.NotNil(t, selfPool)
	assert.Equal(t, pool2, selfPool)

	singlePoolCtx.Close()
	pubsub_factory.ResetSingletonInmemPubsub()
}

func TestPrepareAppWithTenancies(t *testing.T) {
	multiPoolCtx, singlePoolCtx := PrepareAppWithTenancies(t)
	multiPoolCtx.Close()
	singlePoolCtx.Close()
	pubsub_factory.ResetSingletonInmemPubsub()
}

func AddTenancies(t *testing.T, ctx *TenancyTestContext) (*multitenancy.TenancyItem, *multitenancy.TenancyItem) {

	tenancyData1 := &multitenancy.TenancyData{}
	tenancyData1.POOL_ID = "pool2"
	tenancyData1.ROLE = "dev"
	tenancyData1.DESCRIPTION = "tenancy for development"
	tenancyData1.CUSTOMER_ID = "customer1"
	addedTenancy1, err := ctx.RemoteTenancyController.Add(op_context.MakeOpContext(ctx.ClientOp), tenancyData1)
	require.NoError(t, err)
	require.NotNil(t, addedTenancy1)

	tenancyData2 := &multitenancy.TenancyData{}
	tenancyData2.POOL_ID = "pool1"
	tenancyData2.ROLE = "stage"
	tenancyData2.DESCRIPTION = "tenancy for stage"
	tenancyData2.CUSTOMER_ID = "customer1"
	addedTenancy2, err := ctx.RemoteTenancyController.Add(op_context.MakeOpContext(ctx.ClientOp), tenancyData2)
	require.NoError(t, err)
	require.NotNil(t, addedTenancy2)

	b1, _ := json.MarshalIndent(addedTenancy1, "", "  ")
	t.Logf("Added tenancy 1: \n\n%s\n\n", string(b1))

	b2, _ := json.MarshalIndent(addedTenancy2, "", "  ")
	t.Logf("Added tenancy 2: \n\n%s\n\n", string(b2))

	return addedTenancy1, addedTenancy2
}

func TestAddTenancy(t *testing.T) {

	// prepare app with multiple pools and single pool
	multiPoolCtx, singlePoolCtx := PrepareAppWithTenancies(t)

	// add first tenancy to the same pool as single pool app, add via mutipool app
	tenancyData1 := &multitenancy.TenancyData{}
	tenancyData1.POOL_ID = "pool2"
	tenancyData1.ROLE = "dev"
	tenancyData1.DESCRIPTION = "tenancy for development"
	tenancyData1.CUSTOMER_ID = "customer1"
	addedTenancy1, err := multiPoolCtx.RemoteTenancyController.Add(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancyData1)
	require.NoError(t, err)
	require.NotNil(t, addedTenancy1)
	b1, _ := json.MarshalIndent(addedTenancy1, "", "  ")
	t.Logf("Added tenancy: \n\n%s\n\n", string(b1))

	// check if tenancy was loaded by single app
	loadedTenancy1, err := singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(addedTenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, loadedTenancy1)
	loadedT1, ok := loadedTenancy1.(*tenancy_manager.TenancyBase)
	require.True(t, ok)
	b2, _ := json.MarshalIndent(loadedT1.TenancyDb, "", "  ")
	t.Logf("Loaded tenancy: \n\n%s\n\n", string(b2))
	assert.Equal(t, addedTenancy1.TenancyDb, loadedT1.TenancyDb)

	// check if database tables were created
	sample1 := &InTenancySample{Field1: "hello world", Field2: 10}
	err = loadedTenancy1.Db().Create(op_context.MakeOpContext(multiPoolCtx.AdminOp), sample1)
	require.NoError(t, err)
	readSample1 := &InTenancySample{}
	found, err := loadedTenancy1.Db().FindByField(op_context.MakeOpContext(multiPoolCtx.AdminOp), "field2", 10, readSample1)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, sample1, readSample1)

	// add second tenancy to pool different from single pool app, add via mutipool app
	tenancyData2 := &multitenancy.TenancyData{}
	tenancyData2.POOL_ID = "pool1"
	tenancyData2.ROLE = "stage"
	tenancyData2.DESCRIPTION = "tenancy for stage"
	tenancyData2.CUSTOMER_ID = "customer1"
	addedTenancy2, err := multiPoolCtx.RemoteTenancyController.Add(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancyData2)
	require.NoError(t, err)
	require.NotNil(t, addedTenancy2)
	b1, _ = json.MarshalIndent(addedTenancy2, "", "  ")
	t.Logf("Added tenancy: \n\n%s\n\n", string(b1))

	// check if the second tenancy was not loaded by single pool app
	loadedTenancy2, err := singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(addedTenancy2.GetID())
	require.Error(t, err)
	assert.Nil(t, loadedTenancy2)
	loadedT2, ok := loadedTenancy1.(*tenancy_manager.TenancyBase)
	require.True(t, ok)
	b2, _ = json.MarshalIndent(loadedT2.TenancyDb, "", "  ")
	t.Logf("Loaded tenancy: \n\n%s\n\n", string(b2))
	assert.Equal(t, addedTenancy1.TenancyDb, loadedT2.TenancyDb)

	// close apps
	multiPoolCtx.Close()
	singlePoolCtx.Close()
	pubsub_factory.ResetSingletonInmemPubsub()

	// re-init single pool app
	singlePoolCtx = initContext(t, false, "tenancy_single")

	// check if the first tenancy was loaded by single app on initialization
	loadedTenancy1, err = singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(addedTenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, loadedTenancy1)
	loadedT1, ok = loadedTenancy1.(*tenancy_manager.TenancyBase)
	require.True(t, ok)
	assert.Equal(t, addedTenancy1.TenancyDb, loadedT1.TenancyDb)
	// check if the second tenancy was not loaded by single app on initialization
	loadedTenancy2, err = singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(addedTenancy2.GetID())
	require.Error(t, err)
	assert.Nil(t, loadedTenancy2)

	// re-init multiple pool app

	// check if the first tenancy was loaded by multipool app on initialization
	multiPoolCtx = initContext(t, false)
	loadedTenancy1, err = multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(addedTenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, loadedTenancy1)
	loadedT1, ok = loadedTenancy1.(*tenancy_manager.TenancyBase)
	require.True(t, ok)
	assert.Equal(t, addedTenancy1.TenancyDb, loadedT1.TenancyDb)
	multiPoolCtx.Close()

	// check if the second tenancy was loaded by multipool app on initialization
	multiPoolCtx = initContext(t, false)
	loadedTenancy2, err = multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(addedTenancy2.GetID())
	require.NoError(t, err)
	require.NotNil(t, loadedTenancy2)
	loadedT2, ok = loadedTenancy2.(*tenancy_manager.TenancyBase)
	require.True(t, ok)
	assert.Equal(t, addedTenancy2.TenancyDb, loadedT2.TenancyDb)

	// close apps
	multiPoolCtx.Close()
	singlePoolCtx.Close()
	pubsub_factory.ResetSingletonInmemPubsub()
}

func TestListTenancies(t *testing.T) {

	// prepare app with multiple pools and single pool
	multiPoolCtx, singlePoolCtx := PrepareAppWithTenancies(t)

	// add tenancies
	tenancy1, tenancy2 := AddTenancies(t, multiPoolCtx)

	// list tenancies
	filter := db.NewFilter()
	filter.SetSorting("pool_name", db.SORT_DESC)
	tenancies, _, err := multiPoolCtx.RemoteTenancyController.List(op_context.MakeOpContext(multiPoolCtx.ClientOp), filter)
	require.NoError(t, err)
	require.Equal(t, 2, len(tenancies))
	b, _ := json.MarshalIndent(tenancies, "", "  ")
	t.Logf("Tenancies: \n\n%s\n\n", string(b))
	assert.Equal(t, tenancy1, tenancies[0])
	assert.Equal(t, tenancy2, tenancies[1])

	// close apps
	multiPoolCtx.Close()
	singlePoolCtx.Close()
	pubsub_factory.ResetSingletonInmemPubsub()
}

func TestTenancySetters(t *testing.T) {

	// prepare app with multiple pools and single pool
	multiPoolCtx, singlePoolCtx := PrepareAppWithTenancies(t)

	// add tenancies
	tenancy1, tenancy2 := AddTenancies(t, multiPoolCtx)

	singleAppTenancy, err := singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, singleAppTenancy)
	assert.True(t, singleAppTenancy.IsActive())
	assert.Equal(t, tenancy1.Path(), singleAppTenancy.Path())
	assert.Equal(t, tenancy1.Role(), singleAppTenancy.Role())
	assert.Equal(t, "customer1", singleAppTenancy.CustomerDisplay())
	assert.Equal(t, tenancy1.PoolId(), singleAppTenancy.PoolId())

	// change path
	newPath := "tenancy1path"
	err = multiPoolCtx.RemoteTenancyController.SetPath(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), newPath)
	require.NoError(t, err)
	singleAppTenancy, err = singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, singleAppTenancy)
	assert.Equal(t, newPath, singleAppTenancy.Path())
	// try duplicate path
	err = multiPoolCtx.RemoteTenancyController.SetPath(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), newPath)
	test_utils.CheckGenericError(t, err, multitenancy.ErrorCodeTenancyConflictPath)
	// try duplicate shadow path
	err = multiPoolCtx.RemoteTenancyController.SetShadowPath(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), newPath)
	test_utils.CheckGenericError(t, err, multitenancy.ErrorCodeTenancyConflictPath)

	// change shadow path
	shadowPath := "tenancy1shadowpath"
	err = multiPoolCtx.RemoteTenancyController.SetShadowPath(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), shadowPath)
	require.NoError(t, err)
	singleAppTenancy, err = singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, singleAppTenancy)
	assert.Equal(t, newPath, singleAppTenancy.Path())
	assert.Equal(t, shadowPath, singleAppTenancy.ShadowPath())
	// try duplicate path
	err = multiPoolCtx.RemoteTenancyController.SetPath(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), shadowPath)
	test_utils.CheckGenericError(t, err, multitenancy.ErrorCodeTenancyConflictPath)
	// try duplicate shadow path
	err = multiPoolCtx.RemoteTenancyController.SetShadowPath(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), shadowPath)
	test_utils.CheckGenericError(t, err, multitenancy.ErrorCodeTenancyConflictPath)

	// change role
	newRole := "prod"
	err = multiPoolCtx.RemoteTenancyController.SetRole(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), newRole)
	require.NoError(t, err)
	singleAppTenancy, err = singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, singleAppTenancy)
	assert.Equal(t, newRole, singleAppTenancy.Role())
	// try duplicate role
	err = multiPoolCtx.RemoteTenancyController.SetRole(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), newRole)
	test_utils.CheckGenericError(t, err, multitenancy.ErrorCodeTenancyConflictRole)

	// change customer
	newCustomer := "customer2"
	err = multiPoolCtx.RemoteTenancyController.SetCustomer(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), newCustomer)
	require.NoError(t, err)
	singleAppTenancy, err = singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, singleAppTenancy)
	assert.Equal(t, newCustomer, singleAppTenancy.CustomerDisplay())
	// try customer with duplicate role/path
	err = multiPoolCtx.RemoteTenancyController.SetCustomer(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), newCustomer)
	test_utils.CheckGenericError(t, err, multitenancy.ErrorCodeTenancyConflictRole)

	// deactivate tenancy
	err = multiPoolCtx.RemoteTenancyController.SetActive(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), false)
	require.NoError(t, err)
	singleAppTenancy, err = singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, singleAppTenancy)
	assert.False(t, singleAppTenancy.IsActive())

	// activate tenancy
	err = multiPoolCtx.RemoteTenancyController.SetActive(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), true)
	require.NoError(t, err)
	singleAppTenancy, err = singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, singleAppTenancy)
	assert.True(t, singleAppTenancy.IsActive())

	// change pool or db
	newDb := tenancy1.DbName()
	err = multiPoolCtx.RemoteTenancyController.ChangePoolOrDb(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), tenancy2.PoolId(), newDb)
	require.NoError(t, err)
	singleAppTenancy, err = singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.Error(t, err)
	require.Nil(t, singleAppTenancy)
	multiAppTenancy, err := multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, multiAppTenancy)
	assert.Equal(t, tenancy2.PoolId(), multiAppTenancy.PoolId())
	// try set db of foreign tenancy
	newDb = tenancy2.DbName()
	err = multiPoolCtx.RemoteTenancyController.ChangePoolOrDb(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), "pool2", newDb)
	test_utils.CheckGenericError(t, err, multitenancy.ErrorCodeForeignDatabase)
	assert.Equal(t, tenancy2.PoolId(), multiAppTenancy.PoolId())

	// close apps
	multiPoolCtx.Close()
	singlePoolCtx.Close()
	pubsub_factory.ResetSingletonInmemPubsub()
}

func TestFindDelete(t *testing.T) {

	// prepare app with multiple pools and single pool
	multiPoolCtx, singlePoolCtx := PrepareAppWithTenancies(t)

	// add tenancies
	tenancy1, tenancy2 := AddTenancies(t, multiPoolCtx)
	singleAppTenancy, err := singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, singleAppTenancy)
	assert.Equal(t, tenancy1.DbName(), singleAppTenancy.DbName())
	multiAppTenancy1, err := multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, multiAppTenancy1)
	assert.Equal(t, tenancy1.DbName(), multiAppTenancy1.DbName())
	multiAppTenancy2, err := multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy2.GetID())
	require.NoError(t, err)
	require.NotNil(t, multiAppTenancy2)
	assert.Equal(t, tenancy2.DbName(), multiAppTenancy2.DbName())

	// find tenancy
	tenancy, err := multiPoolCtx.RemoteTenancyController.Find(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, tenancy)
	assert.Equal(t, tenancy1, tenancy)

	// try to find tenancy with unknown ID
	_, err = multiPoolCtx.RemoteTenancyController.Find(op_context.MakeOpContext(multiPoolCtx.ClientOp), "unknown_id")
	test_utils.CheckGenericError(t, err, multitenancy.ErrorCodeTenancyNotFound)

	// find tenancy by customer/role
	// dev role
	id := "customer1/dev"
	tenancy, err = multiPoolCtx.RemoteTenancyController.Find(op_context.MakeOpContext(multiPoolCtx.ClientOp), id, true)
	require.NoError(t, err)
	require.NotNil(t, tenancy)
	assert.Equal(t, tenancy1, tenancy)
	// stage role
	id = "customer1/stage"
	tenancy, err = multiPoolCtx.RemoteTenancyController.Find(op_context.MakeOpContext(multiPoolCtx.ClientOp), id, true)
	require.NoError(t, err)
	require.NotNil(t, tenancy)
	assert.Equal(t, tenancy2, tenancy)

	// try to find tenancy with unknown role
	id = "customer1/prod"
	_, err = multiPoolCtx.RemoteTenancyController.Find(op_context.MakeOpContext(multiPoolCtx.ClientOp), id, true)
	test_utils.CheckGenericError(t, err, multitenancy.ErrorCodeTenancyNotFound)

	// check if tenancy exists
	multiPoolCtx.ClientOp.Reset()
	fields := db.Fields{}
	fields["customer_id"] = tenancy.CustomerId()
	fields["role"] = "dev"
	exists, err := multiPoolCtx.RemoteTenancyController.Exists(op_context.MakeOpContext(multiPoolCtx.ClientOp), fields)
	require.NoError(t, err)
	assert.True(t, exists)
	fields["role"] = "prod"
	exists, err = multiPoolCtx.RemoteTenancyController.Exists(op_context.MakeOpContext(multiPoolCtx.ClientOp), fields)
	require.NoError(t, err)
	assert.False(t, exists)

	// delete tenancy
	id = "customer1/dev"
	err = multiPoolCtx.RemoteTenancyController.Delete(op_context.MakeOpContext(multiPoolCtx.ClientOp), id, false, true)
	require.NoError(t, err)

	// check if tenancy was deleted from applications
	singleAppTenancy, err = singlePoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.Error(t, err)
	require.Nil(t, singleAppTenancy)
	multiAppTenancy1, err = multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy1.GetID())
	require.Error(t, err)
	require.Nil(t, multiAppTenancy1)
	multiAppTenancy2, err = multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(tenancy2.GetID())
	require.NoError(t, err)
	require.NotNil(t, multiAppTenancy2)
	assert.Equal(t, tenancy2.DbName(), multiAppTenancy2.DbName())

	// try to find deleted tenancy
	_, err = multiPoolCtx.RemoteTenancyController.Find(op_context.MakeOpContext(multiPoolCtx.ClientOp), id, true)
	test_utils.CheckGenericError(t, err, multitenancy.ErrorCodeTenancyNotFound)

	// close apps
	multiPoolCtx.Close()
	singlePoolCtx.Close()
	pubsub_factory.ResetSingletonInmemPubsub()
}

func TestIpAddresses(t *testing.T) {

	// prepare app with multiple pools and single pool
	multiPoolCtx, singlePoolCtx := PrepareAppWithTenancies(t)

	// add tenancies
	tenancy1, tenancy2 := AddTenancies(t, multiPoolCtx)

	// add IP address to tenancy 1
	ip1 := "192.168.0.1"
	tag1 := "test"
	err := multiPoolCtx.RemoteTenancyController.AddIpAddress(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy1.GetID(), ip1, tag1)
	require.NoError(t, err)

	// list adresses
	addresses, count, err := multiPoolCtx.RemoteTenancyController.ListIpAddresses(op_context.MakeOpContext(multiPoolCtx.ClientOp), nil)
	test_utils.DumpObject(t, addresses, "Addresses 1")
	test_utils.NoError(t, op_context.MakeOpContext(multiPoolCtx.ClientOp), err)
	require.Equal(t, int64(1), count)
	assert.Equal(t, tenancy1.GetID(), addresses[0].TenancyId)
	assert.Equal(t, tenancy1.CustomerLogin, addresses[0].CustomerLogin)
	assert.Equal(t, tenancy1.Role(), addresses[0].TenancyRole)
	assert.Equal(t, ip1, addresses[0].Ip)
	assert.Equal(t, tag1, addresses[0].Tag)

	// add IP address to tenancy 2
	ip2 := "192.168.0.2"
	tag2 := "test2"
	err = multiPoolCtx.RemoteTenancyController.AddIpAddress(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy2.GetID(), ip2, tag2)
	require.NoError(t, err)

	// list adresses
	addresses, count, err = multiPoolCtx.RemoteTenancyController.ListIpAddresses(op_context.MakeOpContext(multiPoolCtx.ClientOp), nil)
	test_utils.DumpObject(t, addresses, "Addresses 2")
	test_utils.NoError(t, op_context.MakeOpContext(multiPoolCtx.ClientOp), err)
	require.Equal(t, int64(2), count)
	assert.Equal(t, tenancy1.GetID(), addresses[0].TenancyId)
	assert.Equal(t, tenancy1.CustomerLogin, addresses[0].CustomerLogin)
	assert.Equal(t, tenancy1.Role(), addresses[0].TenancyRole)
	assert.Equal(t, ip1, addresses[0].Ip)
	assert.Equal(t, tag1, addresses[0].Tag)
	assert.Equal(t, tenancy2.GetID(), addresses[1].TenancyId)
	assert.Equal(t, tenancy2.CustomerLogin, addresses[1].CustomerLogin)
	assert.Equal(t, tenancy2.Role(), addresses[1].TenancyRole)
	assert.Equal(t, ip2, addresses[1].Ip)
	assert.Equal(t, tag2, addresses[1].Tag)

	// delete IP address from tenancy 2
	err = multiPoolCtx.RemoteTenancyController.DeleteIpAddress(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancy2.GetID(), ip2, tag2)
	require.NoError(t, err)
	// list adresses
	addresses, count, err = multiPoolCtx.RemoteTenancyController.ListIpAddresses(op_context.MakeOpContext(multiPoolCtx.ClientOp), nil)
	test_utils.DumpObject(t, addresses, "Addresses 3")
	test_utils.NoError(t, op_context.MakeOpContext(multiPoolCtx.ClientOp), err)
	require.Equal(t, int64(1), count)
	assert.Equal(t, tenancy1.GetID(), addresses[0].TenancyId)
	assert.Equal(t, tenancy1.CustomerLogin, addresses[0].CustomerLogin)
	assert.Equal(t, tenancy1.Role(), addresses[0].TenancyRole)
	assert.Equal(t, ip1, addresses[0].Ip)
	assert.Equal(t, tag1, addresses[0].Tag)

	// close apps
	multiPoolCtx.Close()
	singlePoolCtx.Close()
	pubsub_factory.ResetSingletonInmemPubsub()
}
