package customer_api_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/customer/customer_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type SetDescriptionEndpoint[T customer.User] struct {
	Endpoint[T]
}

func (s *SetDescriptionEndpoint[T]) HandleRequest(sctx context.Context) (context.Context, error) {

	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("customer.SetDescription")
	defer request.TraceOutMethod()

	cmd, err := api_server.ParseValidateRequest[common.WithDescriptionBase](sctx)
	if err != nil {
		return sctx, err
	}

	err = Setter(s.service.Controller, request).SetName(sctx, request.GetResourceId(s.service.UserTypeName).Value(), cmd.Description())
	if err != nil {
		return sctx, c.SetError(err)
	}

	return sctx, nil
}

func SetDescription[T customer.User](service *Service[T]) api_server.ResourceEndpointI {
	e := &SetDescriptionEndpoint[T]{}
	return e.Init(e, "description", service, customer_api.SetDescription())
}
