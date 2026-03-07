package customer_api_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/customer/customer_api"
)

type SetDescriptionEndpoint[T customer.User] struct {
	Endpoint[T]
}

func (s *SetDescriptionEndpoint[T]) HandleRequest(request api_server.Request) error {

	c := request.TraceInMethod("customer.SetDescription")
	defer request.TraceOutMethod()

	cmd, err := api_server.ParseValidateRequest[common.WithDescriptionBase](request)
	if err != nil {
		return err
	}

	err = Setter(s.service.Controller, request).SetName(request, request.GetResourceId(s.service.UserTypeName).Value(), cmd.Description())
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func SetDescription[T customer.User](service *Service[T]) api_server.ResourceEndpointI {
	e := &SetDescriptionEndpoint[T]{}
	return e.Init(e, "description", service, customer_api.SetDescription())
}
