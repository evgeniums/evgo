package user_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type SetBlockedEndpoint struct {
	SetUserFieldEndpoint
}

func (s *SetBlockedEndpoint) HandleRequest(sctx context.Context) error {

	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("users.SetBlocked")
	defer request.TraceOutMethod()

	cmd, err := api_server.ParseValidateRequest[user.UserBlocked](sctx)
	if err != nil {
		return err
	}

	err = Setter(s.users, request).SetBlocked(sctx, request.GetResourceId(s.userTypeName).Value(), cmd.BLOCKED)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func SetBlocked(userTypeName string, users user.MainFieldSetters) api_server.ResourceEndpointI {
	e := &SetBlockedEndpoint{}
	return e.Init(e, userTypeName, "blocked", users, user_api.SetBlocked(userTypeName))
}
