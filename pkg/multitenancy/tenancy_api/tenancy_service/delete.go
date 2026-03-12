package tenancy_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type DeleteEndpoint struct {
	TenancyEndpoint
}

func (e *DeleteEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.Delete")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[tenancy_api.DeleteTenancyCmd](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return sctx, err
	}

	// delete
	err = e.service.Tenancies.Delete(sctx, request.GetTenancyId(), cmd.WithDatabase)
	if err != nil {
		return sctx, c.SetError(err)
	}

	// done
	return sctx, nil
}

func Delete(s *TenancyService) *DeleteEndpoint {
	e := &DeleteEndpoint{}
	e.Construct(s, tenancy_api.Delete())
	return e
}
