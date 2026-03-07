package pool_service

import (
	"errors"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
	"github.com/evgeniums/evgo/pkg/validator"
)

type UpdatePoolEndpoint struct {
	PoolEndpoint
}

func (e *UpdatePoolEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("pool.UpdatePool")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[api.UpdateCmd](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return c.SetError(err)
	}
	// validate fields
	vErr := validator.ValidateMap(request.App().Validator(), cmd.Fields, &pool.PoolBaseData{})
	if vErr != nil {
		c.SetMessage("failed to validate fields")
		request.SetGenericError(vErr.GenericError())
		return c.SetError(vErr.Err)
	}

	// update pool
	poolId := request.GetResourceId("pool").Value()
	p, err := e.service.Pools.UpdatePool(request, poolId, cmd.Fields)
	if err != nil {
		c.SetMessage("failed to update pool")
		return c.SetError(err)
	}

	// find updated pool
	if p == nil {
		request.SetGenericErrorCode(pool.ErrorCodePoolNotFound)
		return c.SetError(errors.New("pool not found"))
	}

	// set response
	resp := &pool_api.PoolResponse{}
	resp.PoolBase = p.(*pool.PoolBase)
	request.Response().SetMessage(resp)

	// done
	return nil
}

func UpdatePool(s *PoolService) *UpdatePoolEndpoint {
	e := &UpdatePoolEndpoint{}
	e.Construct(s, pool_api.UpdatePool())
	return e
}
