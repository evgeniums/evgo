package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type ListServicePoolsEndpoint struct {
	PoolEndpoint
}

func (e *ListServicePoolsEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	var err error
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.ListServicePools")
	defer request.TraceOutMethod()

	// find service
	resp := &pool_api.ListServicePoolsResponse{}
	resp.Items, err = e.service.Pools.GetServiceBindings(sctx, request.GetResourceId("service").Value())
	if err != nil {
		c.SetMessage("failed to get service bindings")
		return sctx, c.SetError(err)
	}

	// set response
	request.Response().SetMessage(resp)

	// done
	return sctx, nil
}

func ListServicePools(s *PoolService) *ListServicePoolsEndpoint {
	e := &ListServicePoolsEndpoint{}
	e.Construct(s, pool_api.ListServicePools())
	return e
}
