package pool_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
	"github.com/evgeniums/evgo/pkg/validator"
)

type UpdateServiceEndpoint struct {
	PoolEndpoint
}

func (e *UpdateServiceEndpoint) HandleRequest(sctx context.Context) error {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("pool.UpdateService")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[api.UpdateCmd](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return c.SetError(err)
	}
	// validate fields
	vErr := validator.ValidateMap(request.App().Validator(), cmd.Fields, &pool.PoolServiceBaseEssentials{})
	if vErr != nil {
		c.SetMessage("failed to validate fields")
		request.SetGenericError(vErr.GenericError())
		return c.SetError(vErr.Err)
	}

	// update service
	serviceId := request.GetResourceId("service").Value()
	s, err := e.service.Pools.UpdateService(sctx, serviceId, cmd.Fields)
	if err != nil {
		c.SetMessage("failed to update service")
		return c.SetError(err)
	}

	// set response
	resp := &pool_api.ServiceResponse{}
	resp.PoolServiceBase = s.(*pool.PoolServiceBase)
	request.Response().SetMessage(resp)

	// done
	return nil
}

func UpdateService(s *PoolService) *UpdateServiceEndpoint {
	e := &UpdateServiceEndpoint{}
	e.Construct(s, pool_api.UpdateService())
	return e
}
