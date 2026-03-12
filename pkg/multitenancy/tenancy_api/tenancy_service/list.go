package tenancy_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type ListEndpoint struct {
	TenancyEndpoint
}

func (e *ListEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.List")
	defer request.TraceOutMethod()

	// parse query
	queryName := request.Endpoint().Resource().ServicePathPrototype()
	filter, err := api_server.ParseDbQuery(sctx, &multitenancy.TenancyItem{}, queryName)
	if err != nil {
		return sctx, c.SetError(err)
	}

	// get
	resp := &tenancy_api.ListTenanciesResponse{}
	resp.Items, resp.Count, err = e.service.Tenancies.List(sctx, filter)
	if err != nil {
		return sctx, c.SetError(err)
	}

	// set response message
	api_server.SetResponseList(request, resp)

	// done
	return sctx, nil
}

func List(s *TenancyService) *ListEndpoint {
	e := &ListEndpoint{}
	e.Construct(s, tenancy_api.List())
	return e
}
