package pool_client

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
	"github.com/evgeniums/evgo/pkg/utils"
)

const poolIdType string = "pool"
const serviceIdType string = "service"

type PoolClient struct {
	api_client.ServiceClient

	PoolsResource    api.Resource
	PoolResource     api.Resource
	ServicesResource api.Resource
	ServiceResource  api.Resource

	add_pool    api.Operation
	add_service api.Operation

	list_pools    api.Operation
	list_services api.Operation
}

func NewPoolClient(client api_client.Client, operationVisitors ...api.OperationVisitors) *PoolClient {

	c := &PoolClient{}
	var serviceName string
	serviceName, c.PoolsResource, c.PoolResource = api.PrepareCollectionAndNameResource(poolIdType)
	c.Init(client, serviceName)
	c.AddChild(c.PoolsResource)

	_, c.ServicesResource, c.ServiceResource = api.PrepareCollectionAndNameResource(serviceIdType)
	c.AddChild(c.ServicesResource)

	c.add_pool = pool_api.AddPool()
	c.list_pools = pool_api.ListPools()
	c.PoolsResource.AddOperations(c.add_pool,
		c.list_pools,
	)

	c.add_service = pool_api.AddService()
	c.list_services = pool_api.ListServices()
	c.ServicesResource.AddOperations(c.add_service,
		c.list_services,
	)

	api.VisitOperation(c.add_pool, operationVisitors...)
	api.VisitOperation(c.list_pools, operationVisitors...)
	api.VisitOperation(c.add_service, operationVisitors...)
	api.VisitOperation(c.list_services, operationVisitors...)

	return c
}

func (p *PoolClient) namedPoolResource(poolId string) api.Resource {
	poolResource := p.PoolResource.CloneChain(false)
	poolResource.SetId(poolId)
	return poolResource
}

func (p *PoolClient) namedServiceResource(serviceId string) api.Resource {
	serviceResource := p.ServiceResource.CloneChain(false)
	serviceResource.SetId(serviceId)
	return serviceResource
}

func (p *PoolClient) resourceForPoolServices(poolId string) api.Resource {
	poolResource := p.namedPoolResource(poolId)
	servicesResource := api.NewResource(serviceIdType)
	poolResource.AddChild(servicesResource)
	return servicesResource
}

func (p *PoolClient) resourceForServicePools(serviceId string) api.Resource {
	serviceResource := p.namedServiceResource(serviceId)
	poolsResource := api.NewResource(poolIdType)
	serviceResource.AddChild(poolsResource)
	return poolsResource
}

func (p *PoolClient) poolId(sctx context.Context, id string, idIsName ...bool) (string, pool.Pool, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolClient.serviceId")
	defer ctx.TraceOutMethod()

	if !utils.OptionalArg(false, idIsName...) {
		return id, nil, nil
	}

	filter := db.NewFilter()
	filter.AddField("name", id)
	pools, _, err := p.GetPools(sctx, filter)
	if err != nil {
		return "", nil, c.SetError(err)
	}
	if len(pools) == 0 {
		ctx.SetGenericErrorCode(pool.ErrorCodePoolNotFound)
		return "", nil, c.SetError(errors.New("pool not found"))
	}

	pool := pools[0]
	return pool.GetID(), pool, nil
}

func (p *PoolClient) serviceId(sctx context.Context, id string, idIsName ...bool) (string, pool.PoolService, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolClient.serviceId")
	defer ctx.TraceOutMethod()

	if !utils.OptionalArg(false, idIsName...) {
		return id, nil, nil
	}

	filter := db.NewFilter()
	filter.AddField("name", id)
	services, _, err := p.GetServices(sctx, filter)
	if err != nil {
		return "", nil, c.SetError(err)
	}
	if len(services) == 0 {
		ctx.SetGenericErrorCode(pool.ErrorCodeServiceNotFound)
		return "", nil, c.SetError(errors.New("service not found"))
	}

	service := services[0]
	return service.GetID(), service, nil
}
