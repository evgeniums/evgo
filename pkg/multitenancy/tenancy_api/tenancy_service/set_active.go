package tenancy_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/common"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
)

type SetActiveEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetActiveEndpoint) HandleRequest(request api_server.Request) error {

	c := request.TraceInMethod("tenancy.SetActive")
	defer request.TraceOutMethod()

	// parse command
	cmd := &common.WithActiveBase{}
	err := request.ParseValidate(cmd)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	if cmd.IsActive() {
		err = s.service.Tenancies.Activate(request, request.GetResourceId(tenancy_api.TenancyResource).Value())
	} else {
		err = s.service.Tenancies.Deactivate(request, request.GetResourceId(tenancy_api.TenancyResource).Value())
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
