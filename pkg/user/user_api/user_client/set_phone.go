package user_client

import (
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type SetPhone = SetterHandler[user.UserPhone]

func (u *UserClient[U]) SetPhone(ctx op_context.Context, id string, phone string, idIsLogin ...bool) error {

	// setup
	c := ctx.TraceInMethod("UserClient.SetPhone")
	defer ctx.TraceOutMethod()

	// if idIsLogin then first find user
	userId, err := u.GetUserId(ctx, id, idIsLogin...)
	if err != nil {
		c.SetMessage("failed to get user ID")
		return c.SetError(err)
	}

	// create command
	handler := &SetPhone{}
	handler.Cmd.PHONE = phone

	// prepare and exec handler
	op := u.UserOperation(userId, "phone", user_api.SetPhone(u.userTypeName))
	err = handler.Exec(u.Client(), ctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return c.SetError(err)
	}

	// done
	return nil
}
