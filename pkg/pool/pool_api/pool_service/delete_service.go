package pool_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type DeleteServiceEndpoint struct {
	PoolEndpoint
}

func (e *DeleteServiceEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("pool.DeleteService")
	defer request.TraceOutMethod()

	// delete pool
	err := e.service.Pools.DeleteService(request, request.GetResourceId("service").Value())
	if err != nil {
		c.SetMessage("failed to delete service")
		return c.SetError(err)
	}

	// done
	return nil
}

func DeleteService(s *PoolService) *DeleteServiceEndpoint {
	e := &DeleteServiceEndpoint{}
	e.Construct(s, pool_api.DeleteService())
	return e
}
