package user_service

import (
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type SetPhoneEndpoint struct {
	SetUserFieldEndpoint
}

func (s *SetPhoneEndpoint) HandleRequest(request api_server.Request) error {

	c := request.TraceInMethod("users.SetPhone")
	defer request.TraceOutMethod()

	cmd, err := api_server.ParseValidateRequest[user.UserPhone](request)
	if err != nil {
		return err
	}

	err = Setter(s.users, request).SetPhone(request, request.GetResourceId(s.userTypeName).Value(), cmd.PHONE)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func SetPhone(userTypeName string, users user.MainFieldSetters) api_server.ResourceEndpointI {
	e := &SetPhoneEndpoint{}
	return e.Init(e, userTypeName, "phone", users, user_api.SetPhone(userTypeName))
}
