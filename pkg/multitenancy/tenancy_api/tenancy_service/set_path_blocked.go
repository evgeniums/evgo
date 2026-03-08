package tenancy_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type SetPathBlockedEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetPathBlockedEndpoint) HandleRequest(sctx context.Context) error {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.SetPathBlocked")
	defer request.TraceOutMethod()

	// parse command
	cmd := &multitenancy.BlockPathCmd{}
	cmd, err := api_server.ParseValidateRequest[multitenancy.BlockPathCmd](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// apply
	err = s.service.Tenancies.SetPathBlocked(sctx, request.GetTenancyId(), cmd.Block, cmd.Mode)
	if err != nil {
		return c.SetError(err)
	}

	// done
	return nil
}

func SetPathBlocked(s *TenancyService) *SetPathBlockedEndpoint {
	e := &SetPathBlockedEndpoint{}
	e.Construct(s, e, "block-path", tenancy_api.SetPathBlocked())
	return e
}
