package pool_service

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
	"github.com/evgeniums/evgo/pkg/validator"
)

type UpdatePoolEndpoint struct {
	PoolEndpoint
}

func (e *UpdatePoolEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.UpdatePool")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[api.UpdateCmd](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return sctx, c.SetError(err)
	}
	// validate fields
	vErr := validator.ValidateMap(request.App().Validator(), cmd.Fields, &pool.PoolBaseData{})
	if vErr != nil {
		c.SetMessage("failed to validate fields")
		request.SetGenericError(vErr.GenericError())
		return sctx, c.SetError(vErr.Err)
	}

	// update pool
	poolId := request.GetResourceId("pool").Value()
	p, err := e.service.Pools.UpdatePool(sctx, poolId, cmd.Fields)
	if err != nil {
		c.SetMessage("failed to update pool")
		return sctx, c.SetError(err)
	}

	// find updated pool
	if p == nil {
		request.SetGenericErrorCode(pool.ErrorCodePoolNotFound)
		return sctx, c.SetError(errors.New("pool not found"))
	}

	// set response
	resp := &pool_api.PoolResponse{}
	resp.PoolBase = p.(*pool.PoolBase)
	request.Response().SetMessage(resp)

	// done
	return sctx, nil
}

func UpdatePool(s *PoolService) *UpdatePoolEndpoint {
	e := &UpdatePoolEndpoint{}
	e.Construct(s, pool_api.UpdatePool())
	return e
}
