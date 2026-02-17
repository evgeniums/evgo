package user_client

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_client"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/user"
	"github.com/evgeniums/go-utils/pkg/user/user_api"
)

type Add[U user.User] struct {
	cmd    interface{}
	result *user_api.UserResponse[U]
}

func (a *Add[U]) Exec(client api_client.Client, ctx op_context.Context, operation api.Operation) error {

	c := ctx.TraceInMethod("Add.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, a.cmd, a.result)
	c.SetError(err)
	return err
}

func (u *UserClient[U]) Add(ctx op_context.Context, login string, password string, extraFieldsSetters ...user.SetUserFields[U]) (U, error) {

	// setup
	var err error
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
		_, err := setter(ctx, user)
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
	err = handler.Exec(u.Client(), ctx, u.add)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nilU, err
	}

	// return result
	return handler.result.User, nil
}
