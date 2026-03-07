package tenancy_service

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
)

type ExistsEndpoint struct {
	TenancyEndpoint
}

func (e *ExistsEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("tenancy.Exists")
	defer request.TraceOutMethod()

	// parse query
	queryName := request.Endpoint().Resource().ServicePathPrototype()
	filter, err := api_server.ParseDbQuery(request, &multitenancy.TenancyDb{}, queryName)
	if err != nil {
		return c.SetError(err)
	}

	// check existence
	resp := &api.ResponseExists{}
	resp.Exists, err = e.service.Tenancies.Exists(request, filter.Fields)
	if err != nil {
		return c.SetError(err)
	}

	// set response
	request.Response().SetMessage(resp)

	// done
	return nil
}

func Exists(s *TenancyService) *ExistsEndpoint {
	e := &ExistsEndpoint{}
	e.Construct(s, tenancy_api.Exists())
	return e
}
