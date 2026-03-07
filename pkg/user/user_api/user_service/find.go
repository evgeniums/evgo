package user_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type FindEndpoint[U user.User] struct {
	api_server.EndpointBase
	UserEndpoint[U]
}

func (e *FindEndpoint[U]) HandleRequest(request api_server.Request) error {

	var err error
	c := request.TraceInMethod("users.Find")
	defer request.TraceOutMethod()

	resp := &user_api.UserResponse[U]{}
	resp.User, err = Users(e.service, request).Find(request, request.GetResourceId(e.service.UserTypeName).Value())
	if err != nil {
		return c.SetError(err)
	}

	request.Response().SetMessage(resp)

	return nil
}

func Find[U user.User](service *UserService[U]) *FindEndpoint[U] {
	e := &FindEndpoint[U]{}
	e.service = service
	e.Construct(user_api.Find(service.UserTypeName))
	return e
}
