package tenancy_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
)

type SetRoleEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetRoleEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("tenancy.SetRole")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[multitenancy.WithRole](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// apply
	err = s.service.Tenancies.SetRole(request, request.GetTenancyId(), cmd.Role())
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
