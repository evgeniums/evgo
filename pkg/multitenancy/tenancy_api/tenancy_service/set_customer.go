package tenancy_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
)

type SetCustomerEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *SetCustomerEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("tenancy.SetCustomer")
	defer request.TraceOutMethod()

	// parse command
	cmd := &multitenancy.WithCustomerId{}
	cmd, err := api_server.ParseValidateRequest[multitenancy.WithCustomerId](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// apply
	err = s.service.Tenancies.SetCustomer(request, request.GetTenancyId(), cmd.CustomerId())
	if err != nil {
		return c.SetError(err)
	}

	// done
	return nil
}

func SetCustomer(s *TenancyService) *SetCustomerEndpoint {
	e := &SetCustomerEndpoint{}
	e.Construct(s, e, "customer", tenancy_api.SetCustomer())
	return e
}
