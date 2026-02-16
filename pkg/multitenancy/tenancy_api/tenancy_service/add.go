package tenancy_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
)

type AddEndpoint struct {
	TenancyEndpoint
}

func (e *AddEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("tenancy.Add")
	defer request.TraceOutMethod()

	// parse command
	cmd := &multitenancy.TenancyData{}
	cmd, err := api_server.ParseValidateRequest[multitenancy.TenancyData](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// add
	resp := &tenancy_api.TenancyResponse{}
	resp.TenancyItem, err = e.service.Tenancies.Add(request, cmd)
	if err != nil {
		c.SetMessage("failed to add tenancy")
		return c.SetError(err)
	}

	// set response
	request.Response().SetMessage(resp)

	// done
	return nil
}

func Add(s *TenancyService) *AddEndpoint {
	e := &AddEndpoint{}
	e.Construct(s, tenancy_api.Add())
	return e
}
