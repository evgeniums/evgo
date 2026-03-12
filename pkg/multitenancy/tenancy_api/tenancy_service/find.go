package tenancy_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type FindEndpoint struct {
	TenancyEndpoint
}

func (f *FindEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	var err error
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.Find")
	defer request.TraceOutMethod()

	// find
	resp := &tenancy_api.TenancyResponse{}
	resp.TenancyItem, err = f.service.Tenancies.Find(sctx, request.GetTenancyId())
	if err != nil {
		c.SetMessage("failed to find tenancy")
		return sctx, c.SetError(err)
	}

	// set response
	request.Response().SetMessage(resp)

	// done
	return sctx, nil
}

func Find(s *TenancyService) *FindEndpoint {
	e := &FindEndpoint{}
	e.Construct(s, tenancy_api.Find())
	return e
}
