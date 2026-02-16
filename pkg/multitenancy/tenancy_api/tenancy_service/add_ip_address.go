package tenancy_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
)

type AddIpAddressEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *AddIpAddressEndpoint) HandleRequest(request api_server.Request) error {

	c := request.TraceInMethod("tenancy.AddIpAddress")
	defer request.TraceOutMethod()

	// parse command
	cmd := &tenancy_api.IpAddressCmd{}
	cmd, err := api_server.ParseValidateRequest[tenancy_api.IpAddressCmd](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	err = s.service.Tenancies.AddIpAddress(request, request.GetTenancyId(), cmd.Ip, cmd.Tag)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func AddIpAddress(s *TenancyService) *AddIpAddressEndpoint {
	e := &AddIpAddressEndpoint{}
	e.Construct(s, e, tenancy_api.IpAddressResource, tenancy_api.AddIpAddress())
	return e
}
