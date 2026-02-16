package tenancy_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
)

type SetPathEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetPathEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("tenancy.SetPath")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[multitenancy.WithPath](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// apply
	err = s.service.Tenancies.SetPath(request, request.GetTenancyId(), cmd.Path())
	if err != nil {
		return c.SetError(err)
	}

	// done
	return nil
}

func SetPath(s *TenancyService) *SetPathEndpoint {
	e := &SetPathEndpoint{}
	e.Construct(s, e, "path", tenancy_api.SetPath())
	return e
}

type SetShadowPathEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetShadowPathEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("tenancy.SetShadowPath")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[multitenancy.WithPath](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// apply
	err = s.service.Tenancies.SetShadowPath(request, request.GetTenancyId(), cmd.ShadowPath())
	if err != nil {
		return c.SetError(err)
	}

	// done
	return nil
}

func SetShadowPath(s *TenancyService) *SetShadowPathEndpoint {
	e := &SetShadowPathEndpoint{}
	e.Construct(s, e, "shadow-path", tenancy_api.SetShadowPath())
	return e
}
