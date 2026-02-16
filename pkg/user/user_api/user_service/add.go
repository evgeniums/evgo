package user_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/user"
	"github.com/evgeniums/go-utils/pkg/user/user_api"
)

type AddEndpoint[U user.User] struct {
	api_server.EndpointBase
	UserEndpoint[U]
	setterBuilder func() user.UserFieldsSetter[U]
}

func (e *AddEndpoint[U]) HandleRequest(request api_server.Request) error {

	c := request.TraceInMethod("users.Add")
	defer request.TraceOutMethod()

	// TODO implement special case to fill object from request
	cmd := e.setterBuilder()
	err := request.ParseAndValidate(cmd)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	resp := &user_api.UserResponse[U]{}
	resp.User, err = Users(e.service, request).Add(request, cmd.Login(), cmd.Password(), cmd.SetUserFields)
	if err != nil {
		return c.SetError(err)
	}

	request.Response().SetMessage(resp)

	return nil
}

func Add[U user.User](service *UserService[U], setterBuilder func() user.UserFieldsSetter[U]) *AddEndpoint[U] {
	e := &AddEndpoint[U]{}
	e.service = service
	e.setterBuilder = setterBuilder
	e.Construct(user_api.Add(service.UserTypeName))
	return e
}
