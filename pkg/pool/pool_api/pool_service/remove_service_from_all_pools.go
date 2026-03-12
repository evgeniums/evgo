package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type RemoveServiceFromAllPoolsEndpoint struct {
	PoolEndpoint
}

func (e *RemoveServiceFromAllPoolsEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.RemoveServiceFromAllPools")
	defer request.TraceOutMethod()

	// do operation
	serviceId := request.GetResourceId("service").Value()
	err := e.service.Pools.RemoveServiceFromAllPools(sctx, serviceId)
	if err != nil {
		c.SetMessage("failed to remove services from all pools")
		return sctx, c.SetError(err)
	}

	// done
	return sctx, nil
}

func RemoveServiceFromAllPools(s *PoolService) *RemoveServiceFromAllPoolsEndpoint {
	e := &RemoveServiceFromAllPoolsEndpoint{}
	e.Construct(s, pool_api.RemoveServiceFromAllPools())
	return e
}
