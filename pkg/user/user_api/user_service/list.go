package user_service

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type ListEndpoint[U user.User] struct {
	api_server.EndpointBase
	UserEndpoint[U]
}

func (e *ListEndpoint[U]) HandleRequest(request api_server.Request) error {

	c := request.TraceInMethod("users.List")
	defer request.TraceOutMethod()

	u := Users(e.service, request)

	queryName := request.Endpoint().Resource().ServicePathPrototype()
	filter, err := api_server.ParseDbQuery(request, u.MakeUser(), queryName)
	if err != nil {
		return c.SetError(err)
	}

	resp := &api.ResponseList[U]{}
	resp.Items, resp.Count, err = u.FindUsers(request, filter)
	if err != nil {
		return c.SetError(err)
	}

	api_server.SetResponseList(request, resp, e.service.UserTypeName)
	return nil
}

func List[U user.User](service *UserService[U]) *ListEndpoint[U] {
	e := &ListEndpoint[U]{}
	e.service = service
	e.Construct(user_api.List())
	return e
}
