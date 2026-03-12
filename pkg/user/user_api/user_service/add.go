package user_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type AddEndpoint[U user.User] struct {
	api_server.EndpointBase
	UserEndpoint[U]
	setterBuilder func() user.UserFieldsSetter[U]
}

func (e *AddEndpoint[U]) HandleRequest(sctx context.Context) (context.Context, error) {

	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("users.Add")
	defer request.TraceOutMethod()

	// TODO implement special case to fill object from request
	cmd := e.setterBuilder()
	err := request.ParseAndValidate(sctx, cmd)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return sctx, err
	}

	resp := &user_api.UserResponse[U]{}
	resp.User, err = Users(e.service, request).Add(sctx, cmd.Login(), cmd.Password(), cmd.SetUserFields)
	if err != nil {
		return sctx, c.SetError(err)
	}

	request.Response().SetMessage(resp)

	return sctx, nil
}

func Add[U user.User](service *UserService[U], setterBuilder func() user.UserFieldsSetter[U]) *AddEndpoint[U] {
	e := &AddEndpoint[U]{}
	e.service = service
	e.setterBuilder = setterBuilder
	e.Construct(user_api.Add(service.UserTypeName))
	return e
}
