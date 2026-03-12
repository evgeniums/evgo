package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type AddServiceToPoolEndpoint struct {
	PoolEndpoint
}

func (e *AddServiceToPoolEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.AddServiceToPool")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[pool.PoolServiceAssociationCmd](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return sctx, err
	}

	// add service to pool
	poolId := request.GetResourceId("pool").Value()
	err = e.service.Pools.AddServiceToPool(sctx, poolId, cmd.SERVICE_ID, cmd.ROLE)
	if err != nil {
		c.SetMessage("failed to add service to pool")
		return sctx, c.SetError(err)
	}

	// done
	return sctx, nil
}

func AddServiceToPool(s *PoolService) *AddServiceToPoolEndpoint {
	e := &AddServiceToPoolEndpoint{}
	e.Construct(s, pool_api.AddServiceToPool())
	return e
}
