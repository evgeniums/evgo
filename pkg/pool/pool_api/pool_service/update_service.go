package pool_service

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/pool"
	"github.com/evgeniums/go-utils/pkg/pool/pool_api"
	"github.com/evgeniums/go-utils/pkg/validator"
)

type UpdateServiceEndpoint struct {
	PoolEndpoint
}

func (e *UpdateServiceEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("pool.UpdateService")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[api.UpdateCmd](request)
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
	s, err := e.service.Pools.UpdateService(request, serviceId, cmd.Fields)
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
