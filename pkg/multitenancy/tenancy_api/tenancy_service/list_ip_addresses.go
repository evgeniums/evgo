package tenancy_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
)

type ListIpAddressesEndpoint struct {
	TenancyEndpoint
}

func (e *ListIpAddressesEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("tenancy.ListIpAddresses")
	defer request.TraceOutMethod()

	// parse query
	queryName := request.Endpoint().Resource().ServicePathPrototype()
	filter, err := api_server.ParseDbQuery(request, &multitenancy.TenancyItem{}, queryName)
	if err != nil {
		return c.SetError(err)
	}

	// get
	resp := &tenancy_api.ListIpAddressesResponse{}
	resp.Items, resp.Count, err = e.service.Tenancies.ListIpAddresses(request, filter)
	if err != nil {
		return c.SetError(err)
	}

	// set response message
	api_server.SetResponseList(request, resp)

	// done
	return nil
}

func ListIpAddresses(s *TenancyService) *ListIpAddressesEndpoint {
	e := &ListIpAddressesEndpoint{}
	e.Construct(s, tenancy_api.ListIpAddresses())
	return e
}
