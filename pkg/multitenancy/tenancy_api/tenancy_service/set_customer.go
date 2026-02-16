package tenancy_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
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
	err := request.ParseValidate(cmd)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// apply
	err = s.service.Tenancies.SetCustomer(request, request.GetResourceId(tenancy_api.TenancyResource).Value(), cmd.CustomerId())
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
