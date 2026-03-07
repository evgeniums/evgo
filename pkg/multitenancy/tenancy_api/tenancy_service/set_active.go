package tenancy_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
)

type SetActiveEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetActiveEndpoint) HandleRequest(request api_server.Request) error {

	c := request.TraceInMethod("tenancy.SetActive")
	defer request.TraceOutMethod()

	// parse command
	cmd := &common.WithActiveBase{}
	cmd, err := api_server.ParseValidateRequest[common.WithActiveBase](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	if cmd.IsActive() {
		err = s.service.Tenancies.Activate(request, request.GetTenancyId())
	} else {
		err = s.service.Tenancies.Deactivate(request, request.GetTenancyId())
	}
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func SetActive(s *TenancyService) *SetActiveEndpoint {
	e := &SetActiveEndpoint{}
	e.Construct(s, e, "active", tenancy_api.SetActive())
	return e
}
