package pool_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type ListPoolServicesEndpoint struct {
	PoolEndpoint
}

func (e *ListPoolServicesEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	var err error
	c := request.TraceInMethod("pool.ListPoolServices")
	defer request.TraceOutMethod()

	// find service
	resp := &pool_api.ListServicePoolsResponse{}
	resp.Items, err = e.service.Pools.GetPoolBindings(request, request.GetResourceId("pool").Value())
	if err != nil {
		c.SetMessage("failed to get service bindings")
		return c.SetError(err)
	}

	// set response
	request.Response().SetMessage(resp)

	// done
	return nil
}

func ListPoolServices(s *PoolService) *ListPoolServicesEndpoint {
	e := &ListPoolServicesEndpoint{}
	e.Construct(s, pool_api.ListPoolServices())
	return e
}
