package user_client

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_client"
	"github.com/evgeniums/go-utils/pkg/db"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/user"
)

type List[U user.User] struct {
	cmd    api.Query
	result *api.ResponseList[U]
}

func (a *List[U]) Exec(client api_client.Client, ctx op_context.Context, operation api.Operation) error {

	c := ctx.TraceInMethod("List.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, a.cmd, a.result)
	c.SetError(err)
	return err
}

func (u *UserClient[U]) FindUsers(ctx op_context.Context, filter *db.Filter) ([]U, int64, error) {

	// setup
	var err error
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
	err = handler.Exec(u.Client(), ctx, u.list)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, 0, err
	}

	// return result
	return handler.result.Items, handler.result.Count, nil
}
