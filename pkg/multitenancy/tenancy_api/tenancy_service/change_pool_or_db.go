package tenancy_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
)

type ChangePoolOrDbEndpoint struct {
	TenancyUpdateEndpoint
}

func (s *ChangePoolOrDbEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("tenancy.ChangePoolOrDb")
	defer request.TraceOutMethod()

	// parse command
	cmd := &multitenancy.WithPoolAndDb{}
	cmd, err := api_server.ParseValidateRequest[multitenancy.WithPoolAndDb](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	// apply
	err = s.service.Tenancies.ChangePoolOrDb(request, request.GetTenancyId(), cmd.PoolId(), cmd.DbName())
	if err != nil {
		return c.SetError(err)
	}

	// done
	return nil
}

func ChangePoolOrDb(s *TenancyService) *ChangePoolOrDbEndpoint {
	e := &ChangePoolOrDbEndpoint{}
	e.Construct(s, e, "pool-db", tenancy_api.ChangePoolOrDb())
	return e
}
