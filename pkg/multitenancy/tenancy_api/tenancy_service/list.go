package tenancy_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
)

type ListEndpoint struct {
	TenancyEndpoint
}

func (e *ListEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("tenancy.List")
	defer request.TraceOutMethod()

	// parse query
	queryName := request.Endpoint().Resource().ServicePathPrototype()
	filter, err := api_server.ParseDbQuery(request, &multitenancy.TenancyItem{}, queryName)
	if err != nil {
		return c.SetError(err)
	}

	// get
	resp := &tenancy_api.ListTenanciesResponse{}
	resp.Items, resp.Count, err = e.service.Tenancies.List(request, filter)
	if err != nil {
		return c.SetError(err)
	}

	// set response message
	api_server.SetResponseList(request, resp)

	// done
	return nil
}

func List(s *TenancyService) *ListEndpoint {
	e := &ListEndpoint{}
	e.Construct(s, tenancy_api.List())
	return e
}
