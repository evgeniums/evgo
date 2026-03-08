package tenancy_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type SetActiveEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetActiveEndpoint) HandleRequest(sctx context.Context) error {

	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("tenancy.SetActive")
	defer request.TraceOutMethod()

	// parse command
	cmd := &common.WithActiveBase{}
	cmd, err := api_server.ParseValidateRequest[common.WithActiveBase](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	if cmd.IsActive() {
		err = s.service.Tenancies.Activate(sctx, request.GetTenancyId())
	} else {
		err = s.service.Tenancies.Deactivate(sctx, request.GetTenancyId())
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
