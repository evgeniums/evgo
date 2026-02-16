package pool_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/pool"
	"github.com/evgeniums/go-utils/pkg/pool/pool_api"
)

type AddServiceEndpoint struct {
	PoolEndpoint
}

func (e *AddServiceEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("pool.AddService")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest(request, pool.InitService)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// add service
	s, err := e.service.Pools.AddService(request, cmd)
	if err != nil {
		c.SetMessage("failed to add service")
		return c.SetError(err)
	}

	// set response
	resp := &pool_api.ServiceResponse{}
	resp.PoolServiceBase = s.(*pool.PoolServiceBase)
	request.Response().SetMessage(resp)

	// done
	return nil
}

func AddService(s *PoolService) *AddServiceEndpoint {
	e := &AddServiceEndpoint{}
	e.Construct(s, pool_api.AddService())
	return e
}
