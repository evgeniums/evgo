package tenancy_service

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
)

type TenancyEndpoint struct {
	service *TenancyService
	api_server.EndpointBase
}

func (e *TenancyEndpoint) Construct(service *TenancyService, op api.Operation) {
	e.service = service
	e.EndpointBase.Construct(op)
}

type TenancyUpdateEndpoint struct {
	service *TenancyService
	api_server.ResourceEndpoint
}

func (e *TenancyUpdateEndpoint) Construct(service *TenancyService, ep api_server.ResourceEndpointI, epName string, op api.Operation) {
	e.service = service
	api_server.ConstructResourceEndpoint(ep, epName, op)
}

type TenancyService struct {
	api_server.ServiceBase
	Tenancies multitenancy.TenancyController

	TenanciesResource   api.Resource
	TenancyResource     api.Resource
	IpAddressesResource api.Resource
}

func NewTenancyService(tenancyController multitenancy.TenancyController) *TenancyService {

	s := &TenancyService{}
	s.AppendErrorExtender(tenancyController)
	s.ErrorsExtenderBase.AddErrors(customer.ErrorDescriptions, customer.ErrorHttpCodes)
	s.Tenancies = tenancyController

	s.Init(tenancy_api.ServiceName, api.PackageName)
	s.TenancyResource = api.NamedResource(tenancy_api.TenancyResource)
	s.TenanciesResource = s.TenancyResource.Parent()
	s.AddChild(s.TenanciesResource)

	listOp := List(s)
	s.TenanciesResource.AddOperations(Add(s), listOp)
	existsResource := api.NewResource("exists")
	existsResource.AddOperation(Exists(s))
	s.TenanciesResource.AddChild(existsResource)

	s.TenancyResource.AddOperation(Find(s), true)
	s.TenancyResource.AddOperation(Delete(s))
	s.TenancyResource.AddChildren(SetActive(s),
		SetPath(s),
		SetShadowPath(s),
		SetRole(s),
		SetCustomer(s),
		ChangePoolOrDb(s),
		AddIpAddress(s),
		DeleteIpAddress(s),
		SetPathBlocked(s),
	)

	s.IpAddressesResource = api.NewResource(tenancy_api.IpAddressResource)
	s.AddChild(s.IpAddressesResource)
	listIpAddressesOp := ListIpAddresses(s)
	s.IpAddressesResource.AddOperation(listIpAddressesOp)

	tenancyTableConfig := &api_server.DynamicTableConfig{Model: &multitenancy.TenancyItem{}, Operation: listOp}
	ipAddressesTableConfig := &api_server.DynamicTableConfig{Model: &multitenancy.TenancyIpAddressItem{}, Operation: listIpAddressesOp}
	s.AddDynamicTables(tenancyTableConfig, ipAddressesTableConfig)

	return s
}
