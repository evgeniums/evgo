package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type AddServiceEndpoint struct {
	PoolEndpoint
}

func (e *AddServiceEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.AddService")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest(sctx, pool.InitService)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return sctx, err
	}

	// add service
	s, err := e.service.Pools.AddService(sctx, cmd)
	if err != nil {
		c.SetMessage("failed to add service")
		return sctx, c.SetError(err)
	}

	// set response
	resp := &pool_api.ServiceResponse{}
	resp.PoolServiceBase = s.(*pool.PoolServiceBase)
	request.Response().SetMessage(resp)

	// done
	return sctx, nil
}

func AddService(s *PoolService) *AddServiceEndpoint {
	e := &AddServiceEndpoint{}
	e.Construct(s, pool_api.AddService())
	return e
}
