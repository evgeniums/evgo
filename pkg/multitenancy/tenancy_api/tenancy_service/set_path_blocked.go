package tenancy_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
)

type SetPathBlockedEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetPathBlockedEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("tenancy.SetPathBlocked")
	defer request.TraceOutMethod()

	// parse command
	cmd := &multitenancy.BlockPathCmd{}
	cmd, err := api_server.ParseValidateRequest[multitenancy.BlockPathCmd](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// apply
	err = s.service.Tenancies.SetPathBlocked(request, request.GetTenancyId(), cmd.Block, cmd.Mode)
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
