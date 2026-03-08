package customer_api_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/customer/customer_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type SetNameEndpoint[T customer.User] struct {
	Endpoint[T]
}

func (s *SetNameEndpoint[T]) HandleRequest(sctx context.Context) error {

	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("SetName")
	defer request.TraceOutMethod()

	cmd := &common.WithNameBase{}
	cmd, err := api_server.ParseValidateRequest[common.WithNameBase](sctx)
	if err != nil {
		return err
	}

	err = Setter(s.service.Controller, request).SetName(sctx, request.GetResourceId(s.service.UserTypeName).Value(), cmd.Name())
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func SetName[T customer.User](service *Service[T]) api_server.ResourceEndpointI {
	e := &SetNameEndpoint[T]{}
	return e.Init(e, "name", service, customer_api.SetName())
}
