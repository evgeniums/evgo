package user_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type SetPasswordEndpoint struct {
	SetUserFieldEndpoint
}

func (s *SetPasswordEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("users.SetPassword")
	defer request.TraceOutMethod()

	cmd, err := api_server.ParseValidateRequest[user.UserPlainPassword](sctx)
	if err != nil {
		return sctx, err
	}

	err = Setter(s.users, request).SetPassword(sctx, request.GetResourceId(s.userTypeName).Value(), cmd.PlainPassword)
	if err != nil {
		return sctx, c.SetError(err)
	}

	return sctx, nil
}

func SetPassword(userTypeName string, users user.MainFieldSetters) api_server.ResourceEndpointI {
	e := &SetPasswordEndpoint{}
	return e.Init(e, userTypeName, "password", users, user_api.SetPassword(userTypeName))
}
