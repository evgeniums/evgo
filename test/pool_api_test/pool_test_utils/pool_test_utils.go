package pool_test_utils

import (
	"encoding/json"
	"testing"

	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api/pool_client"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/test/api_test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type PoolTestContext struct {
	*api_test.TestContext

	LocalPoolController  pool.PoolController
	RemotePoolController *pool_client.PoolClient
}

func AddPool(t *testing.T, ctx *PoolTestContext, poolName ...string) pool.Pool {

	// fill sample
	p1Sample := &pool.PoolBaseData{}
	p1Sample.SetName(utils.OptionalArg("pool1", poolName...))
	p1Sample.SetLongName("pool1 long name")
	p1Sample.SetDescription("pool description")

	// create pool
	p1 := pool.NewPool()
	p1.SetName(p1Sample.Name())
	p1.SetDescription(p1Sample.Description())
	p1.SetLongName(p1Sample.LongName())

	// add pool
	addedPool1, err := ctx.RemotePoolController.AddPool(ctx.ClientOp, p1)
	require.NoError(t, err)
	require.NotNil(t, addedPool1)
	assert.Equal(t, p1Sample.Name(), addedPool1.Name())
	assert.Equal(t, p1Sample.LongName(), addedPool1.LongName())
	assert.Equal(t, p1Sample.Description(), addedPool1.Description())
	assert.True(t, addedPool1.IsActive())
	assert.NotEmpty(t, addedPool1.GetID())

	// find pool locally
	dbPool1, err := ctx.LocalPoolController.FindPool(ctx.AdminOp, p1Sample.Name(), true)
	require.NoError(t, err)
	require.NotNil(t, dbPool1)
	b1, _ := json.Marshal(addedPool1)
	b2, _ := json.Marshal(dbPool1)
	assert.Equal(t, string(b1), string(b2))

	// find pool remotely
	remotePool1, err := ctx.RemotePoolController.FindPool(ctx.ClientOp, p1Sample.Name(), true)
	require.NoError(t, err)
	require.NotNil(t, dbPool1)
	b3, _ := json.Marshal(remotePool1)
	assert.Equal(t, string(b1), string(b3))

	// try to add pool with duplicate name
	p2 := pool.NewPool()
	p2.SetName(p1Sample.Name())
	p2.SetDescription(p1Sample.Description())
	p2.SetLongName(p1Sample.LongName())
	_, err = ctx.RemotePoolController.AddPool(ctx.ClientOp, p2)
	test_utils.CheckGenericError(t, err, pool.ErrorCodePoolNameConflict)

	// done
	return addedPool1
}

func DefaultServiceConfig(serviceName ...string) *pool.PoolServiceBaseEssentials {

	p1Sample := &pool.PoolServiceBaseEssentials{}
	p1Sample.SetName(utils.OptionalArg("service1", serviceName...))
	p1Sample.SetLongName("service1 long name")
	p1Sample.SetDescription("service description")
	p1Sample.SetTypeName("database")
	p1Sample.SetRefId("reference id")

	p1Sample.ServiceConfigBase.PROVIDER = "sqlite"
	p1Sample.ServiceConfigBase.PUBLIC_HOST = "pubhost"
	p1Sample.ServiceConfigBase.PUBLIC_PORT = 1122
	p1Sample.ServiceConfigBase.PUBLIC_URL = "puburl"
	p1Sample.ServiceConfigBase.PRIVATE_HOST = "privhost"
	p1Sample.ServiceConfigBase.PRIVATE_PORT = 3344
	p1Sample.ServiceConfigBase.PRIVATE_URL = "privurl"
	p1Sample.ServiceConfigBase.PARAMETER1 = "param1"
	p1Sample.ServiceConfigBase.PARAMETER2 = "param2"
	p1Sample.ServiceConfigBase.PARAMETER3 = "param3"
	p1Sample.ServiceConfigBase.DB_NAME = "dbname1"

	return p1Sample
}

func AddService(t *testing.T, ctx *PoolTestContext, serviceConfig *pool.PoolServiceBaseEssentials) pool.PoolService {

	// fill data
	p1Sample := serviceConfig

	// create service
	p1 := pool.NewService()
	p1.SetName(p1Sample.Name())
	p1.SetLongName(p1Sample.LongName())
	p1.SetDescription(p1Sample.Description())
	p1.SetTypeName(p1Sample.TypeName())
	p1.SetRefId(p1Sample.RefId())
	p1.PoolServiceBaseEssentials.ServiceConfigBase = p1Sample.ServiceConfigBase
	p1.SECRET1 = serviceConfig.SECRET1
	if p1.SECRET1 == "" && p1.Provider() != "redis" {
		p1.SECRET1 = "secret1"
	}
	p1.SECRET2 = "secret2"
	p1.SetActive(true)

	// add service
	addedService1, err := ctx.RemotePoolController.AddService(ctx.ClientOp, p1)
	require.NoError(t, err)
	require.NotNil(t, addedService1)
	assert.NotEmpty(t, addedService1.GetID())
	addedB1, ok := addedService1.(*pool.PoolServiceBase)
	require.True(t, ok)
	assert.Equal(t, p1.PoolServiceBaseEssentials, addedB1.PoolServiceBaseEssentials)
	assert.Equal(t, p1.Secret1(), addedService1.Secret1())
	assert.Equal(t, p1.Secret2(), addedService1.Secret2())

	// find locally
	dbService1, err := ctx.LocalPoolController.FindService(ctx.AdminOp, p1Sample.Name(), true)
	require.NoError(t, err)
	require.NotNil(t, dbService1)

	b1, _ := json.Marshal(addedService1)
	b2, _ := json.Marshal(dbService1)
	assert.Equal(t, string(b1), string(b2))

	// find remotely
	remoteService1, err := ctx.RemotePoolController.FindService(ctx.ClientOp, p1Sample.Name(), true)
	require.NoError(t, err)
	require.NotNil(t, remoteService1)

	b3, _ := json.Marshal(remoteService1)
	assert.Equal(t, string(b1), string(b3))

	// try to add service with duplicate name
	p2 := pool.NewService()
	p2.SetName(p1Sample.Name())
	p2.SetLongName(p1Sample.LongName())
	p2.SetDescription(p1Sample.Description())
	p2.SetTypeName(p1Sample.TypeName())
	p2.SetRefId(p1Sample.RefId())
	p2.PoolServiceBaseEssentials.ServiceConfigBase = p1Sample.ServiceConfigBase
	p2.SECRET1 = "secret1"
	p2.SECRET2 = "secret2"
	_, err = ctx.RemotePoolController.AddService(ctx.ClientOp, p2)
	test_utils.CheckGenericError(t, err, pool.ErrorCodeServiceNameConflict)

	// done
	return addedService1
}
