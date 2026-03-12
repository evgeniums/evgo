package tenancy_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type AddIpAddressEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *AddIpAddressEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.AddIpAddress")
	defer request.TraceOutMethod()

	// parse command
	cmd := &tenancy_api.IpAddressCmd{}
	cmd, err := api_server.ParseValidateRequest[tenancy_api.IpAddressCmd](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return sctx, err
	}

	err = s.service.Tenancies.AddIpAddress(sctx, request.GetTenancyId(), cmd.Ip, cmd.Tag)
	if err != nil {
		return sctx, c.SetError(err)
	}

	return sctx, nil
}

func AddIpAddress(s *TenancyService) *AddIpAddressEndpoint {
	e := &AddIpAddressEndpoint{}
	e.Construct(s, e, tenancy_api.IpAddressResource, tenancy_api.AddIpAddress())
	return e
}
