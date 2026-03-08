package tenancy_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type DeleteIpAddressEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *DeleteIpAddressEndpoint) HandleRequest(sctx context.Context) error {

	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.DeleteIpAddress")
	defer request.TraceOutMethod()

	// parse command
	cmd := &tenancy_api.IpAddressCmd{}
	cmd, err := api_server.ParseValidateRequest[tenancy_api.IpAddressCmd](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	err = s.service.Tenancies.DeleteIpAddress(sctx, request.GetTenancyId(), cmd.Ip, cmd.Tag)
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
