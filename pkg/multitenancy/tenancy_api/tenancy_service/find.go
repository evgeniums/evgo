package tenancy_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
)

type FindEndpoint struct {
	TenancyEndpoint
}

func (f *FindEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	var err error
	c := request.TraceInMethod("tenancy.Find")
	defer request.TraceOutMethod()

	// find
	resp := &tenancy_api.TenancyResponse{}
	resp.TenancyItem, err = f.service.Tenancies.Find(request, request.GetTenancyId())
	if err != nil {
		c.SetMessage("failed to find tenancy")
		return c.SetError(err)
	}

	// set response
	request.Response().SetMessage(resp)

	// done
	return nil
}

func Find(s *TenancyService) *FindEndpoint {
	e := &FindEndpoint{}
	e.Construct(s, tenancy_api.Find())
	return e
}
