package tenancy_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
)

type DeleteIpAddressEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *DeleteIpAddressEndpoint) HandleRequest(request api_server.Request) error {

	c := request.TraceInMethod("tenancy.DeleteIpAddress")
	defer request.TraceOutMethod()

	// parse command
	cmd := &tenancy_api.IpAddressCmd{}
	err := request.ParseValidate(cmd)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	err = s.service.Tenancies.DeleteIpAddress(request, request.GetResourceId(tenancy_api.TenancyResource).Value(), cmd.Ip, cmd.Tag)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func DeleteIpAddress(s *TenancyService) *DeleteIpAddressEndpoint {
	e := &DeleteIpAddressEndpoint{}
	e.Construct(s, e, tenancy_api.IpAddressResource, tenancy_api.DeleteIpAddress())
	return e
}
