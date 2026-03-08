package user_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
)

type List[U user.User] struct {
	cmd    api.Query
	result *api.ResponseList[U]
}

func (a *List[U]) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("List.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, a.cmd, a.result)
	c.SetError(err)
	return err
}

func (u *UserClient[U]) FindUsers(sctx context.Context, filter *db.Filter) ([]U, int64, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("UserClient.FindUsers", logger.Fields{"user_type": u.userTypeName})
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// set query
	cmd := &api.DbQuery{}
	if filter != nil {
		cmd.SetQuery(filter.ToQueryString())
	}

	// prepare and exec handler
	handler := &List[U]{
		cmd:    cmd,
		result: &api.ResponseList[U]{},
	}
	err = handler.Exec(u.Client(), sctx, u.list)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, 0, err
	}

	// return result
	return handler.result.Items, handler.result.Count, nil
}
