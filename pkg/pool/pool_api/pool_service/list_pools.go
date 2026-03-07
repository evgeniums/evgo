package pool_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type ListPoolsEndpoint struct {
	PoolEndpoint
}

func (e *ListPoolsEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("pool.ListPools")
	defer request.TraceOutMethod()

	// parse query
	queryName := request.Endpoint().Resource().ServicePathPrototype()
	filter, err := api_server.ParseDbQuery(request, &pool.PoolBase{}, queryName)
	if err != nil {
		return c.SetError(err)
	}

	// get services
	resp := &pool_api.ListPoolsResponse{}
	resp.Items, resp.Count, err = e.service.Pools.GetPools(request, filter)
	if err != nil {
		return c.SetError(err)
	}

	// set response message
	api_server.SetResponseList(request, resp)

	// done
	return nil
}

func ListPools(s *PoolService) *ListPoolsEndpoint {
	e := &ListPoolsEndpoint{}
	e.Construct(s, pool_api.ListPools())
	return e
}
