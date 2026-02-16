package pool_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/pool"
	"github.com/evgeniums/go-utils/pkg/pool/pool_api"
)

type FindPoolEndpoint struct {
	PoolEndpoint
}

func (e *FindPoolEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("pool.FindPool")
	defer request.TraceOutMethod()

	// find pool
	p, err := e.service.Pools.FindPool(request, request.GetResourceId("pool").Value())
	if err != nil {
		c.SetMessage("failed to find pool")
		return c.SetError(err)
	}

	// set response
	resp := &pool_api.PoolResponse{}
	resp.PoolBase = p.(*pool.PoolBase)
	request.Response().SetMessage(resp)

	// done
	return nil
}

func FindPool(s *PoolService) *FindPoolEndpoint {
	e := &FindPoolEndpoint{}
	e.Construct(s, pool_api.FindPool())
	return e
}
