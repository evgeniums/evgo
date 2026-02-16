package pool_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/pool"
	"github.com/evgeniums/go-utils/pkg/pool/pool_api"
)

type AddServiceToPoolEndpoint struct {
	PoolEndpoint
}

func (e *AddServiceToPoolEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("pool.AddServiceToPool")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[pool.PoolServiceAssociationCmd](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// add service to pool
	poolId := request.GetResourceId("pool").Value()
	err = e.service.Pools.AddServiceToPool(request, poolId, cmd.SERVICE_ID, cmd.ROLE)
	if err != nil {
		c.SetMessage("failed to add service to pool")
		return c.SetError(err)
	}

	// done
	return nil
}

func AddServiceToPool(s *PoolService) *AddServiceToPoolEndpoint {
	e := &AddServiceToPoolEndpoint{}
	e.Construct(s, pool_api.AddServiceToPool())
	return e
}
