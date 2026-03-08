package customer_api_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/customer/customer_api"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user/user_api/user_client"
)

type SetDescription = user_client.SetterHandler[common.WithDescriptionBase]

func (u *Client[T]) SetDescription(sctx context.Context, id string, description string, idIsLogin ...bool) error {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("Client.SetDescription")
	defer ctx.TraceOutMethod()

	// if idIsLogin then first find user
	userId, err := u.GetUserId(sctx, id, idIsLogin...)
	if err != nil {
		c.SetMessage("failed to get user ID")
		return c.SetError(err)
	}

	// create command
	handler := &SetDescription{}
	handler.Cmd.SetDescription(description)

	// prepare and exec handler
	op := u.UserOperation(userId, "description", customer_api.SetDescription())
	err = handler.Exec(u.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return c.SetError(err)
	}

	// done
	return nil
}
