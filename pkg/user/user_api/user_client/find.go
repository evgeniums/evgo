package user_client

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
	"github.com/evgeniums/evgo/pkg/utils"
)

type Find[U user.User] struct {
	result *user_api.UserResponse[U]
}

func (a *Find[U]) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("Find.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, nil, a.result)
	c.SetError(err)
	return err
}

func (u *UserClient[U]) Find(sctx context.Context, id string) (U, error) {

	var nilU U

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("UserClient.Find")
	defer ctx.TraceOutMethod()

	// create command
	handler := &Find[U]{}
	handler.result = &user_api.UserResponse[U]{}

	// prepare and exec handler
	op := api.NamedResourceOperation(u.UserResource, id, user_api.Find(u.userTypeName))
	err := handler.Exec(u.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nilU, c.SetError(err)
	}

	// done
	return handler.result.User, nil
}

func (u *UserClient[U]) FindByLogin(sctx context.Context, login string) (U, error) {

	var nilU U
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("UserClient.FindByLogin")
	defer ctx.TraceOutMethod()

	filter := db.NewFilter()
	filter.AddField("login", login)

	users, _, err := u.FindUsers(sctx, filter)
	if err != nil {
		return nilU, c.SetError(err)
	}

	if len(users) < 1 {
		return nilU, c.SetError(errors.New("user not found"))
	}

	return users[0], nil
}

func (u *UserClient[U]) GetUserId(sctx context.Context, id string, idIsLogin ...bool) (string, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("UserClient.SetBlocked")
	defer ctx.TraceOutMethod()

	if !utils.OptionalArg(false, idIsLogin...) {
		return id, nil
	}

	user, err := u.FindByLogin(sctx, id)
	if err != nil {
		return "", c.SetError(err)
	}

	return user.GetID(), nil
}
