package user_client

import (
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type SetEmail = SetterHandler[user.UserEmail]

func (u *UserClient[U]) SetEmail(ctx op_context.Context, id string, email string, idIsLogin ...bool) error {

	// setup
	c := ctx.TraceInMethod("UserClient.SetEmail")
	defer ctx.TraceOutMethod()

	// if idIsLogin then first find user
	userId, err := u.GetUserId(ctx, id, idIsLogin...)
	if err != nil {
		c.SetMessage("failed to get user ID")
		return c.SetError(err)
	}

	// create command
	handler := &SetEmail{}
	handler.Cmd.EMAIL = email

	// prepare and exec handler
	op := u.UserOperation(userId, "email", user_api.SetEmail(u.userTypeName))
	err = handler.Exec(u.Client(), ctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return c.SetError(err)
	}

	// done
	return nil
}
