package customer_api_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/common"
	"github.com/evgeniums/go-utils/pkg/customer"
	"github.com/evgeniums/go-utils/pkg/customer/customer_api"
)

type SetNameEndpoint[T customer.User] struct {
	Endpoint[T]
}

func (s *SetNameEndpoint[T]) HandleRequest(request api_server.Request) error {

	c := request.TraceInMethod("SetName")
	defer request.TraceOutMethod()

	cmd := &common.WithNameBase{}
	cmd, err := api_server.ParseValidateRequest[common.WithNameBase](request)
	if err != nil {
		return err
	}

	err = Setter(s.service.Controller, request).SetName(request, request.GetResourceId(s.service.UserTypeName).Value(), cmd.Name())
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func SetName[T customer.User](service *Service[T]) api_server.ResourceEndpointI {
	e := &SetNameEndpoint[T]{}
	return e.Init(e, "name", service, customer_api.SetName())
}
