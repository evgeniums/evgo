package tenancy_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type SetRoleEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetRoleEndpoint) HandleRequest(sctx context.Context) error {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.SetRole")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[multitenancy.WithRole](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// apply
	err = s.service.Tenancies.SetRole(sctx, request.GetTenancyId(), cmd.Role())
	if err != nil {
		return c.SetError(err)
	}

	// done
	return nil
}

func SetRole(s *TenancyService) *SetRoleEndpoint {
	e := &SetRoleEndpoint{}
	e.Construct(s, e, "role", tenancy_api.SetRole())
	return e
}
