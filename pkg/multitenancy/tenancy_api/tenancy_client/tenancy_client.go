package tenancy_client

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_client"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
)

type TenancyClient struct {
	api_client.ServiceClient

	TenanciesResource   api.Resource
	TenancyResource     api.Resource
	IpAddressesResource api.Resource

	add               api.Operation
	list              api.Operation
	exists            api.Operation
	list_ip_addresses api.Operation
}

type OperationWrapper interface {
	WrapOperaton(client api_client.Client, op api.Operation)
}

func NewTenancyClient(client api_client.Client) *TenancyClient {

	c := &TenancyClient{}

	c.Init(client, tenancy_api.ServiceName)
	c.TenancyResource = api.NamedResource(tenancy_api.TenancyResource)
	c.TenanciesResource = c.TenancyResource.Parent()
	c.AddChild(c.TenanciesResource)

	c.IpAddressesResource = api.NewResource(tenancy_api.IpAddressResource)
	c.AddChild(c.IpAddressesResource)

	c.add = tenancy_api.Add()
	c.list = tenancy_api.List()
	c.TenanciesResource.AddOperations(c.add,
		c.list,
	)

	c.list_ip_addresses = tenancy_api.ListIpAddresses()
	c.IpAddressesResource.AddOperation(c.list_ip_addresses)

	existsResource := api.NewResource("exists")
	c.TenanciesResource.AddChild(existsResource)
	c.exists = tenancy_api.Exists()
	existsResource.AddOperation(c.exists)

	return c
}
