package tenancy_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
)

type DeleteEndpoint struct {
	TenancyEndpoint
}

func (e *DeleteEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("tenancy.Delete")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[tenancy_api.DeleteTenancyCmd](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// delete
	err = e.service.Tenancies.Delete(request, request.GetTenancyId(), cmd.WithDatabase)
	if err != nil {
		return c.SetError(err)
	}

	// done
	return nil
}

func Delete(s *TenancyService) *DeleteEndpoint {
	e := &DeleteEndpoint{}
	e.Construct(s, tenancy_api.Delete())
	return e
}
