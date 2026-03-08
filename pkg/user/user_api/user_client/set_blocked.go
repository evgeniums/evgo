package user_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type SetBlocked = SetterHandler[user.UserBlocked]

func (u *UserClient[U]) SetBlocked(sctx context.Context, id string, blocked bool, idIsLogin ...bool) error {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("UserClient.SetBlocked")
	defer ctx.TraceOutMethod()

	// if idIsLogin then first find user
	userId, err := u.GetUserId(sctx, id, idIsLogin...)
	if err != nil {
		c.SetMessage("failed to get user ID")
		return c.SetError(err)
	}

	// create command
	handler := &SetBlocked{}
	handler.Cmd.BLOCKED = blocked

	// prepare and exec handler
	op := u.UserOperation(userId, "blocked", user_api.SetBlocked(u.userTypeName))
	err = handler.Exec(u.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return c.SetError(err)
	}

	// done
	return nil
}
