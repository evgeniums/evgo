package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type ListServicesEndpoint struct {
	PoolEndpoint
}

func (e *ListServicesEndpoint) HandleRequest(sctx context.Context) error {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.ListServices")
	defer request.TraceOutMethod()

	// parse query
	queryName := request.Endpoint().Resource().ServicePathPrototype()
	filter, err := api_server.ParseDbQuery(sctx, &pool.PoolServiceBase{}, queryName)
	if err != nil {
		return c.SetError(err)
	}

	// get services
	resp := &pool_api.ListServicesResponse{}
	resp.Items, resp.Count, err = e.service.Pools.GetServices(sctx, filter)
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
