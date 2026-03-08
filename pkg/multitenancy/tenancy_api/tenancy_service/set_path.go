package tenancy_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type SetPathEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetPathEndpoint) HandleRequest(sctx context.Context) error {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.SetPath")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[multitenancy.WithPath](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// apply
	err = s.service.Tenancies.SetPath(sctx, request.GetTenancyId(), cmd.Path())
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

func (s *SetShadowPathEndpoint) HandleRequest(sctx context.Context) error {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.SetShadowPath")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[multitenancy.WithPath](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// apply
	err = s.service.Tenancies.SetShadowPath(sctx, request.GetTenancyId(), cmd.ShadowPath())
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
