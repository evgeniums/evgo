package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type RemoveServiceFromPoolEndpoint struct {
	PoolEndpoint
}

func (e *RemoveServiceFromPoolEndpoint) HandleRequest(sctx context.Context) error {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.RemoveServiceToPool")
	defer request.TraceOutMethod()

	// do operation
	poolId := request.GetResourceId("pool").Value()
	role := request.GetResourceId("role").Value()
	err := e.service.Pools.RemoveServiceFromPool(sctx, poolId, role)
	if err != nil {
		c.SetMessage("failed to remove service from pool")
		return c.SetError(err)
	}

	// done
	return nil
}

func RemoveServiceFromPool(s *PoolService) *RemoveServiceFromPoolEndpoint {
	e := &RemoveServiceFromPoolEndpoint{}
	e.Construct(s, pool_api.RemoveServiceFromPool())
	return e
}
