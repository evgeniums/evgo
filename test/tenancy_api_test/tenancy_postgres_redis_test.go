package tenancy_api_test

import (
	"encoding/json"
	"testing"

	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_manager"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_providers/pubsub_factory"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_providers/pubsub_redis"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initDatabase(t *testing.T) (string, *test_utils.PostgresDbConfig) {
	pgConfig := test_utils.NewPostgresDbConfig()
	dbName := "bhelpers_db"
	test_utils.DropDatabase(t, pgConfig, dbName)
	test_utils.CreateDatabase(t, pgConfig, dbName)

	test_utils.DropDatabase(t, pgConfig, "tenancy_customer1_dev")
	test_utils.DropDatabase(t, pgConfig, "tenancy_customer1_stage")

	return dbName, pgConfig
}

func TestPostgresRedis(t *testing.T) {

	t.Skip("Run this test manually after preparing postgres and redis service.")

	// prepare pools with postgres and redis services
	dbName, pgConfig := initDatabase(t)
	prepareCtx := initContext(t, true, "postgres")

	doc1 := &SampleModel1{}
	doc1.InitObject()
	doc1.Field1 = "value1"
	doc1.Field2 = "value2"
	require.NoError(t, prepareCtx.ServerApp.Db().Create(prepareCtx.ServerApp, doc1), "failed to create doc1 in database")

	docDb1 := &SampleModel1{}
	found, err := prepareCtx.ServerApp.Db().FindByFields(prepareCtx.ServerApp, db.Fields{"field1": "value1"}, docDb1)
	require.NoError(t, err, "failed to find doc1 in database")
	assert.Equal(t, found, true)
	assert.True(t, doc1.GetCreatedAt().Equal(docDb1.GetCreatedAt()))

	pools1 := prepareCtx.AppWithTenancy.Pools()
	require.NotNil(t, pools1)
	require.Empty(t, pools1.Pools())
	selfPool1, err := pools1.SelfPool()
	assert.Error(t, err)
	assert.Nil(t, selfPool1)

	poolConfig1 := &TenancyPoolConfig{
		PoolName: "pool1",
	}
	poolConfig1.DbService = TenancyServiceConfig{Name: "database_service1", Type: pool.TypeDatabase, Provider: "postgres"}
	poolConfig1.DbService.DbConfig = *pgConfig
	poolConfig1.DbService.DbName = dbName
	poolConfig1.PubsubService = TenancyServiceConfig{Name: "pubsub_service1", Type: pool.TypePubsub, Provider: pubsub_redis.Provider}
	poolConfig1.PubsubService.DbHost = "127.0.0.1"
	poolConfig1.PubsubService.DbPort = 6379
	poolConfig1.PubsubService.DbName = "0"
	p1 := preparePoolServices(t, prepareCtx, poolConfig1)

	poolConfig2 := &TenancyPoolConfig{
		PoolName: "pool2",
	}
	poolConfig2.DbService = TenancyServiceConfig{Name: "database_service2", Type: pool.TypeDatabase, Provider: "postgres"}
	poolConfig2.DbService.DbConfig = *pgConfig
	poolConfig2.DbService.DbName = dbName
	poolConfig2.PubsubService = TenancyServiceConfig{Name: "pubsub_service2", Type: pool.TypePubsub, Provider: pubsub_redis.Provider}
	poolConfig2.PubsubService.DbHost = "127.0.0.1"
	poolConfig2.PubsubService.DbPort = 6379
	poolConfig2.PubsubService.DbName = "1"
	p2 := preparePoolServices(t, prepareCtx, poolConfig2)

	_, err = pool.ActivatePool(prepareCtx.RemotePoolController, prepareCtx.ClientOp, p1.GetID())
	require.NoError(t, err)
	_, err = pool.ActivatePool(prepareCtx.RemotePoolController, prepareCtx.ClientOp, p2.GetID())
	require.NoError(t, err)

	prepareCtx.Close()

	// prepare app with multiple pools
	multiPoolCtx := initContext(t, false, "postgres")

	// add customers
	customer1, err := multiPoolCtx.LocalCustomerManager.Add(multiPoolCtx.AdminOp, "customer1", "12345678")
	require.NoError(t, err)
	require.NotNil(t, customer1)
	customer2, err := multiPoolCtx.LocalCustomerManager.Add(multiPoolCtx.AdminOp, "customer2", "12345678")
	require.NoError(t, err)
	require.NotNil(t, customer2)

	// add tenancies
	addedTenancy1, _ := AddTenancies(t, multiPoolCtx)

	// check if tenancy was loaded
	loadedTenancy1, err := multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(addedTenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, loadedTenancy1)
	loadedT1, ok := loadedTenancy1.(*tenancy_manager.TenancyBase)
	require.True(t, ok)
	b2, _ := json.MarshalIndent(loadedT1.TenancyDb, "", "  ")
	t.Logf("Loaded tenancy: \n\n%s\n\n", string(b2))
	assert.Equal(t, addedTenancy1.TenancyDb, loadedT1.TenancyDb)

	// check if database tables were created
	sample1 := &InTenancySample{Field1: "hello world", Field2: 10}
	sample1.GenerateID()
	err = loadedTenancy1.Db().Create(multiPoolCtx.AdminOp, sample1)
	require.NoError(t, err)
	readSample1 := &InTenancySample{}
	found, err = loadedTenancy1.Db().FindByField(multiPoolCtx.AdminOp, "field2", 10, readSample1)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, sample1, readSample1)

	// check partitions
	inPart1 := &PartitionedItem{}
	inPart1.InitObject()
	inPart1.Field4 = "p1_field4"
	inPart1.Field5 = 1010
	err = loadedTenancy1.Db().Create(multiPoolCtx.AdminOp, inPart1)
	require.NoError(t, err)
	readInPart1 := &PartitionedItem{}
	found, err = loadedTenancy1.Db().FindByField(multiPoolCtx.AdminOp, "field5", 1010, readInPart1)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, inPart1.GetID(), readInPart1.GetID())
	assert.True(t, inPart1.GetCreatedAt().Equal(readInPart1.GetCreatedAt()))
	assert.Equal(t, inPart1.GetMonth(), readInPart1.GetMonth())
	assert.Equal(t, inPart1.Field4, readInPart1.Field4)
	assert.Equal(t, inPart1.Field5, readInPart1.Field5)

	// TODO check explicit partitions

	// close apps
	multiPoolCtx.Close()
	pubsub_factory.ResetSingletonInmemPubsub()
}

func TestPostgresDup(t *testing.T) {

	t.Skip("Run this test manually after preparing postgres")

	initDatabase(t)
	prepareCtx := initContext(t, true, "postgres")

	doc1 := &SampleModel1{}
	doc1.InitObject()
	doc1.Field1 = "value1"
	doc1.Field2 = "value2"
	require.NoError(t, prepareCtx.ServerApp.Db().Create(prepareCtx.ServerApp, doc1), "failed to create doc1 in database")

	dup, err := prepareCtx.ServerApp.Db().CreateDup(prepareCtx.ServerApp, doc1)
	test_utils.DumpError(t, err)
	require.True(t, dup)
}
