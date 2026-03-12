package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type RemoveAllServicesFromPoolEndpoint struct {
	PoolEndpoint
}

func (e *RemoveAllServicesFromPoolEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.RemoveAllServicesFromPool")
	defer request.TraceOutMethod()

	// do operation
	poolId := request.GetResourceId("pool").Value()
	err := e.service.Pools.RemoveAllServicesFromPool(sctx, poolId)
	if err != nil {
		c.SetMessage("failed to remove services from pool")
		return sctx, c.SetError(err)
	}

	// done
	return sctx, nil
}

func RemoveAllServicesFromPool(s *PoolService) *RemoveAllServicesFromPoolEndpoint {
	e := &RemoveAllServicesFromPoolEndpoint{}
	e.Construct(s, pool_api.RemoveAllServicesFromPool())
	return e
}
