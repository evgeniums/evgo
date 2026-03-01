package pool_service

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/pool"
)

type PoolEndpoint struct {
	service *PoolService
	api_server.EndpointBase
}

func (e *PoolEndpoint) Construct(service *PoolService, op api.Operation) {
	e.service = service
	e.EndpointBase.Construct(op)
}

type PoolService struct {
	api_server.ServiceBase
	Pools pool.PoolController

	PoolsResource    api.Resource
	PoolResource     api.Resource
	ServicesResource api.Resource
	ServResource     api.Resource
}

func NewPoolService(poolController pool.PoolController) *PoolService {

	s := &PoolService{}
	s.ErrorsExtenderBase.Init(pool.ErrorDescriptions, pool.ErrorHttpCodes)
	s.Pools = poolController

	var serviceName string
	serviceName, s.PoolsResource, s.PoolResource = api.PrepareCollectionAndNameResource("pool")
	s.Init(serviceName, api.PackageName)
	s.AddChild(s.PoolsResource)

	_, s.ServicesResource, s.ServResource = api.PrepareCollectionAndNameResource("service")
	s.AddChild(s.ServicesResource)

	listPools := ListPools(s)
	s.PoolsResource.AddOperations(AddPool(s), listPools)
	s.PoolResource.AddOperation(FindPool(s), true)
	s.PoolResource.AddOperations(UpdatePool(s), DeletePool(s))

	poolServiceAssociations := api.NewResource("service")
	listPoolServices := ListPoolServices(s)
	poolServiceAssociations.AddOperations(listPoolServices, AddServiceToPool(s), RemoveAllServicesFromPool(s))
	s.PoolResource.AddChild(poolServiceAssociations)

	poolServiceAssociation := api.NamedResource("role")
	poolServiceAssociation.AddOperation(RemoveServiceFromPool(s))
	poolServiceAssociations.AddChild(poolServiceAssociation)

	listServices := ListServices(s)
	s.ServicesResource.AddOperations(AddService(s), listServices)
	s.ServResource.AddOperation(FindService(s), true)
	s.ServResource.AddOperations(UpdateService(s), DeleteService(s))

	servicePoolAssociations := api.NewResource("pool")
	listServicePools := ListServicePools(s)
	servicePoolAssociations.AddOperations(listServicePools, RemoveServiceFromAllPools(s))
	s.ServResource.AddChild(servicePoolAssociations)

	poolTableConfig := &api_server.DynamicTableConfig{Model: &pool.PoolBase{}, Operation: listPools}
	serviceTableConfig := &api_server.DynamicTableConfig{Model: &pool.PoolServiceBase{}, Operation: listServices}
	poolServicesTableConfig := &api_server.DynamicTableConfig{Model: &pool.PoolServiceBinding{}, Operation: listPoolServices}
	servicePoolsTableConfig := &api_server.DynamicTableConfig{Model: &pool.PoolServiceBinding{}, Operation: listServicePools}
	s.AddDynamicTables(poolTableConfig, serviceTableConfig, poolServicesTableConfig, servicePoolsTableConfig)

	return s
}
