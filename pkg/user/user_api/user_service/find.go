package user_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type FindEndpoint[U user.User] struct {
	api_server.EndpointBase
	UserEndpoint[U]
}

func (e *FindEndpoint[U]) HandleRequest(sctx context.Context) (context.Context, error) {

	var err error
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("users.Find")
	defer request.TraceOutMethod()

	resp := &user_api.UserResponse[U]{}
	resp.User, err = Users(e.service, request).Find(sctx, request.GetResourceId(e.service.UserTypeName).Value())
	if err != nil {
		return sctx, c.SetError(err)
	}

	request.Response().SetMessage(resp)

	return sctx, nil
}

func Find[U user.User](service *UserService[U]) *FindEndpoint[U] {
	e := &FindEndpoint[U]{}
	e.service = service
	e.Construct(user_api.Find(service.UserTypeName))
	return e
}
