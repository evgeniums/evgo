package user_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type SetEmailEndpoint struct {
	SetUserFieldEndpoint
}

func (s *SetEmailEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("users.SetEmail")
	defer request.TraceOutMethod()

	cmd, err := api_server.ParseValidateRequest[user.UserEmail](sctx)
	if err != nil {
		return sctx, err
	}

	err = Setter(s.users, request).SetEmail(sctx, request.GetResourceId(s.userTypeName).Value(), cmd.EMAIL)
	if err != nil {
		return sctx, c.SetError(err)
	}

	return sctx, nil
}

func SetEmail(userTypeName string, users user.MainFieldSetters) api_server.ResourceEndpointI {
	e := &SetEmailEndpoint{}
	return e.Init(e, userTypeName, "email", users, user_api.SetEmail(userTypeName))
}
