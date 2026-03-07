package user_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type SetPasswordEndpoint struct {
	SetUserFieldEndpoint
}

func (s *SetPasswordEndpoint) HandleRequest(request api_server.Request) error {

	c := request.TraceInMethod("users.SetPassword")
	defer request.TraceOutMethod()

	cmd, err := api_server.ParseValidateRequest[user.UserPlainPassword](request)
	if err != nil {
		return err
	}

	err = Setter(s.users, request).SetPassword(request, request.GetResourceId(s.userTypeName).Value(), cmd.PlainPassword)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func SetPassword(userTypeName string, users user.MainFieldSetters) api_server.ResourceEndpointI {
	e := &SetPasswordEndpoint{}
	return e.Init(e, userTypeName, "password", users, user_api.SetPassword(userTypeName))
}
