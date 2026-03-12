package tenancy_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type SetDbRoleEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetDbRoleEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.SetDbRole")
	defer request.TraceOutMethod()

	// parse command
	cmd := &multitenancy.WithRole{}
	cmd, err := api_server.ParseValidateRequest[multitenancy.WithRole](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return sctx, err
	}

	// apply
	err = s.service.Tenancies.SetDbRole(sctx, request.GetTenancyId(), cmd.Role())
	if err != nil {
		return sctx, c.SetError(err)
	}

	// done
	return sctx, nil
}

func SetDbRole(s *TenancyService) *SetDbRoleEndpoint {
	e := &SetDbRoleEndpoint{}
	e.Construct(s, e, "db_role", tenancy_api.SetDbRole())
	return e
}
