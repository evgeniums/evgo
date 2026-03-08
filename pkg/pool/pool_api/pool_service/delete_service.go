package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type DeleteServiceEndpoint struct {
	PoolEndpoint
}

func (e *DeleteServiceEndpoint) HandleRequest(sctx context.Context) error {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.DeleteService")
	defer request.TraceOutMethod()

	// delete pool
	err := e.service.Pools.DeleteService(sctx, request.GetResourceId("service").Value())
	if err != nil {
		c.SetMessage("failed to delete service")
		return c.SetError(err)
	}

	// done
	return nil
}

func DeleteService(s *PoolService) *DeleteServiceEndpoint {
	e := &DeleteServiceEndpoint{}
	e.Construct(s, pool_api.DeleteService())
	return e
}
