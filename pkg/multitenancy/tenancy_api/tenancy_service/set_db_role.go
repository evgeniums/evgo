package tenancy_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
)

type SetDbRoleEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetDbRoleEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("tenancy.SetDbRole")
	defer request.TraceOutMethod()

	// parse command
	cmd := &multitenancy.WithRole{}
	cmd, err := api_server.ParseValidateRequest[multitenancy.WithRole](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// apply
	err = s.service.Tenancies.SetDbRole(request, request.GetTenancyId(), cmd.Role())
	if err != nil {
		return c.SetError(err)
	}

	// done
	return nil
}

func SetDbRole(s *TenancyService) *SetDbRoleEndpoint {
	e := &SetDbRoleEndpoint{}
	e.Construct(s, e, "db_role", tenancy_api.SetDbRole())
	return e
}
