package pool_service

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type FindServiceEndpoint struct {
	PoolEndpoint
}

func (e *FindServiceEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.FindService")
	defer request.TraceOutMethod()

	// find service
	s, err := e.service.Pools.FindService(sctx, request.GetResourceId("service").Value())
	if err != nil {
		c.SetMessage("failed to find service")
		return sctx, c.SetError(err)
	}
	if s == nil {
		request.SetGenericErrorCode(pool.ErrorCodeServiceNotFound)
		return sctx, c.SetError(errors.New("service not found"))
	}

	// set response
	resp := &pool_api.ServiceResponse{}
	resp.PoolServiceBase = s.(*pool.PoolServiceBase)
	request.Response().SetMessage(resp)

	// done
	return sctx, nil
}

func FindService(s *PoolService) *FindServiceEndpoint {
	e := &FindServiceEndpoint{}
	e.Construct(s, pool_api.FindService())
	return e
}
