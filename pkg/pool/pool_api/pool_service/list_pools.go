package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type ListPoolsEndpoint struct {
	PoolEndpoint
}

func (e *ListPoolsEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.ListPools")
	defer request.TraceOutMethod()

	// parse query
	queryName := request.Endpoint().Resource().ServicePathPrototype()
	filter, err := api_server.ParseDbQuery(sctx, &pool.PoolBase{}, queryName)
	if err != nil {
		return sctx, c.SetError(err)
	}

	// get services
	resp := &pool_api.ListPoolsResponse{}
	resp.Items, resp.Count, err = e.service.Pools.GetPools(sctx, filter)
	if err != nil {
		return sctx, c.SetError(err)
	}

	// set response message
	api_server.SetResponseList(request, resp)

	// done
	return sctx, nil
}

func ListPools(s *PoolService) *ListPoolsEndpoint {
	e := &ListPoolsEndpoint{}
	e.Construct(s, pool_api.ListPools())
	return e
}
