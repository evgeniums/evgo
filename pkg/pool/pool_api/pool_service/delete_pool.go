package pool_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/pool/pool_api"
)

type DeletePoolEndpoint struct {
	PoolEndpoint
}

func (e *DeletePoolEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("pool.DeletePool")
	defer request.TraceOutMethod()

	// delete pool
	err := e.service.Pools.DeletePool(request, request.GetResourceId("pool").Value())
	if err != nil {
		c.SetMessage("failed to delete pool")
		return c.SetError(err)
	}

	// done
	return nil
}

func DeletePool(s *PoolService) *DeletePoolEndpoint {
	e := &DeletePoolEndpoint{}
	e.Construct(s, pool_api.DeletePool())
	return e
}
