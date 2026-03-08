package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type AddPoolEndpoint struct {
	PoolEndpoint
}

func (e *AddPoolEndpoint) HandleRequest(sctx context.Context) error {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.AddPool")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest(sctx, pool.InitPool)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// add pool
	p, err := e.service.Pools.AddPool(sctx, cmd)
	if err != nil {
		c.SetMessage("failed to add pool")
		return c.SetError(err)
	}

	// set response
	resp := &pool_api.PoolResponse{}
	resp.PoolBase = p.(*pool.PoolBase)
	request.Response().SetMessage(resp)

	// done
	return nil
}

func AddPool(s *PoolService) *AddPoolEndpoint {
	e := &AddPoolEndpoint{}
	e.Construct(s, pool_api.AddPool())
	return e
}
