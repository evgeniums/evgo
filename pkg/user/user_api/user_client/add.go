package user_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type Add[U user.User] struct {
	cmd    interface{}
	result *user_api.UserResponse[U]
}

func (a *Add[U]) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("Add.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, a.cmd, a.result)
	c.SetError(err)
	return err
}

func (u *UserClient[U]) Add(sctx context.Context, login string, password string, extraFieldsSetters ...user.SetUserFields[U]) (U, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("UserClient.Add", logger.Fields{"login": login, "user_type": u.userTypeName})
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	var nilU U

	// create user
	user := u.userBuilder()
	user.SetLogin(login)
	for _, setter := range extraFieldsSetters {
		_, err := setter(sctx, user)
		if err != nil {
			c.SetMessage("failed to set extra field")
			return nilU, err
		}
	}

	// create command from user
	cmd := user.ToCmd(password)

	// prepare and exec handler
	handler := &Add[U]{
		cmd:    cmd,
		result: &user_api.UserResponse[U]{},
	}
	err = handler.Exec(u.Client(), sctx, u.add)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nilU, err
	}

	// return result
	return handler.result.User, nil
}
