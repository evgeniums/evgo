package customer_api_client

import (
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/customer/customer_api"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user/user_api/user_client"
)

type SetName = user_client.SetterHandler[common.WithNameBase]

func (u *Client[T]) SetName(ctx op_context.Context, id string, name string, idIsLogin ...bool) error {

	// setup
	c := ctx.TraceInMethod("Client.SetName")
	defer ctx.TraceOutMethod()

	// if idIsLogin then first find user
	userId, err := u.GetUserId(ctx, id, idIsLogin...)
	if err != nil {
		c.SetMessage("failed to get user ID")
		return c.SetError(err)
	}

	// create command
	handler := &SetName{}
	handler.Cmd.SetName(name)

	// prepare and exec handler
	op := u.UserOperation(userId, "name", customer_api.SetName())
	err = handler.Exec(u.Client(), ctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return c.SetError(err)
	}

	// done
	return nil
}
