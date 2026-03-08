package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type ListPoolServicesEndpoint struct {
	PoolEndpoint
}

func (e *ListPoolServicesEndpoint) HandleRequest(sctx context.Context) error {

	// setup
	var err error
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.ListPoolServices")
	defer request.TraceOutMethod()

	// find service
	resp := &pool_api.ListServicePoolsResponse{}
	resp.Items, err = e.service.Pools.GetPoolBindings(sctx, request.GetResourceId("pool").Value())
	if err != nil {
		c.SetMessage("failed to get service bindings")
		return c.SetError(err)
	}

	// set response
	request.Response().SetMessage(resp)

	// done
	return nil
}

func ListPoolServices(s *PoolService) *ListPoolServicesEndpoint {
	e := &ListPoolServicesEndpoint{}
	e.Construct(s, pool_api.ListPoolServices())
	return e
}
