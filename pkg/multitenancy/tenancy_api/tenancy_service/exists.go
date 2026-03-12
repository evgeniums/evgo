package tenancy_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type ExistsEndpoint struct {
	TenancyEndpoint
}

func (e *ExistsEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.Exists")
	defer request.TraceOutMethod()

	// parse query
	queryName := request.Endpoint().Resource().ServicePathPrototype()
	filter, err := api_server.ParseDbQuery(sctx, &multitenancy.TenancyDb{}, queryName)
	if err != nil {
		return sctx, c.SetError(err)
	}

	// check existence
	resp := &api.ResponseExists{}
	resp.Exists, err = e.service.Tenancies.Exists(sctx, filter.Fields)
	if err != nil {
		return sctx, c.SetError(err)
	}

	// set response
	request.Response().SetMessage(resp)

	// done
	return sctx, nil
}

func Exists(s *TenancyService) *ExistsEndpoint {
	e := &ExistsEndpoint{}
	e.Construct(s, tenancy_api.Exists())
	return e
}
