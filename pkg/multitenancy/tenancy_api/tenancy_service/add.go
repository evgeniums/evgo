package tenancy_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type AddEndpoint struct {
	TenancyEndpoint
}

func (e *AddEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.Add")
	defer request.TraceOutMethod()

	// parse command
	cmd := &multitenancy.TenancyData{}
	cmd, err := api_server.ParseValidateRequest[multitenancy.TenancyData](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return sctx, err
	}

	// add
	resp := &tenancy_api.TenancyResponse{}
	resp.TenancyItem, err = e.service.Tenancies.Add(sctx, cmd)
	if err != nil {
		c.SetMessage("failed to add tenancy")
		return sctx, c.SetError(err)
	}

	// set response
	request.Response().SetMessage(resp)

	// done
	return sctx, nil
}

func Add(s *TenancyService) *AddEndpoint {
	e := &AddEndpoint{}
	e.Construct(s, tenancy_api.Add())
	return e
}
