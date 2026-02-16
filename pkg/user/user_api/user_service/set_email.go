package user_service

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/user"
	"github.com/evgeniums/go-utils/pkg/user/user_api"
)

type SetEmailEndpoint struct {
	SetUserFieldEndpoint
}

func (s *SetEmailEndpoint) HandleRequest(request api_server.Request) error {

	c := request.TraceInMethod("users.SetEmail")
	defer request.TraceOutMethod()

	cmd := &user.UserEmail{}
	err := request.ParseValidate(cmd)
	if err != nil {
		return err
	}

	err = Setter(s.users, request).SetEmail(request, request.GetResourceId(s.userTypeName).Value(), cmd.EMAIL)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func SetEmail(userTypeName string, users user.MainFieldSetters) api_server.ResourceEndpointI {
	e := &SetEmailEndpoint{}
	return e.Init(e, userTypeName, "email", users, user_api.SetEmail(userTypeName))
}
