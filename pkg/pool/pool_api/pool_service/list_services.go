package pool_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type ListServicesEndpoint struct {
	PoolEndpoint
}

func (e *ListServicesEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("pool.ListServices")
	defer request.TraceOutMethod()

	// parse query
	queryName := request.Endpoint().Resource().ServicePathPrototype()
	filter, err := api_server.ParseDbQuery(request, &pool.PoolServiceBase{}, queryName)
	if err != nil {
		return c.SetError(err)
	}

	// get services
	resp := &pool_api.ListServicesResponse{}
	resp.Items, resp.Count, err = e.service.Pools.GetServices(request, filter)
	if err != nil {
		return c.SetError(err)
	}

	// set response message
	api_server.SetResponseList(request, resp)

	// done
	return nil
}

func ListServices(s *PoolService) *ListServicesEndpoint {
	e := &ListServicesEndpoint{}
	e.Construct(s, pool_api.ListServices())
	return e
}
