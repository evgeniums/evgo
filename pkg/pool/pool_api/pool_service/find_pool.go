package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type FindPoolEndpoint struct {
	PoolEndpoint
}

func (e *FindPoolEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.FindPool")
	defer request.TraceOutMethod()

	// find pool
	p, err := e.service.Pools.FindPool(sctx, request.GetResourceId("pool").Value())
	if err != nil {
		c.SetMessage("failed to find pool")
		return sctx, c.SetError(err)
	}

	// set response
	resp := &pool_api.PoolResponse{}
	resp.PoolBase = p.(*pool.PoolBase)
	request.Response().SetMessage(resp)

	// done
	return sctx, nil
}

func FindPool(s *PoolService) *FindPoolEndpoint {
	e := &FindPoolEndpoint{}
	e.Construct(s, pool_api.FindPool())
	return e
}
