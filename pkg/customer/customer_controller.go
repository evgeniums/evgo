package customer

import (
	"context"
	"net/http"

	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
)

const (
	ErrorCodeCustomerNotFound string = "customer_not_found"
)

var ErrorDescriptions = map[string]string{
	ErrorCodeCustomerNotFound: "Customer not found",
}

var ErrorHttpCodes = map[string]int{
	ErrorCodeCustomerNotFound: http.StatusNotFound,
}

type NameAndDescriptionSetter interface {
	SetName(sctx context.Context, id string, name string, idIsLogin ...bool) error
	SetDescription(sctx context.Context, id string, description string, idIsLogin ...bool) error
}

type UserNameAndDescriptionController[T user.User] interface {
	user.UserController[T]
	NameAndDescriptionSetter
}

type UserNameAndDescriptionControllerB[T user.User] struct {
	*user.UserControllerBase[T]
}

func (cu *UserNameAndDescriptionControllerB[T]) SetName(sctx context.Context, id string, name string, idIsLogin ...bool) error {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	ctx.SetLoggerField("name", name)
	c := ctx.TraceInMethod("Users.SetName")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find user
	user, err := user.FindUser(cu.UserControllerBase, sctx, id, idIsLogin...)
	if err != nil {
		return err
	}

	// set name
	err = cu.CRUD().Update(sctx, user, db.Fields{"name": name})
	if err != nil {
		return err
	}

	// done
	cu.OpLog(sctx, "set_name", user.GetID(), user.Login())
	return nil
}

func (cu *UserNameAndDescriptionControllerB[T]) SetDescription(sctx context.Context, id string, description string, idIsLogin ...bool) error {
	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("Users.SetDescription")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find user
	user, err := user.FindUser(cu.UserControllerBase, sctx, id, idIsLogin...)
	if err != nil {
		return err
	}

	// set description
	err = cu.CRUD().Update(sctx, user, db.Fields{"description": description})
	if err != nil {
		return err
	}

	// done
	cu.OpLog(sctx, "set_description", user.GetID(), user.Login())
	return nil
}

func LocalCustomerController() *CustomersControllerBase {
	c := &CustomersControllerBase{}
	c.ErrorsExtenderBase.Init(ErrorDescriptions, ErrorHttpCodes)
	c.UserControllerBase = user.LocalUserController[*Customer]()
	c.SetUserBuilder(NewCustomer)
	c.SetOplogBuilder(NewOplog)
	return c
}

type CustomerController interface {
	generic_error.ErrorsExtender
	UserNameAndDescriptionController[*Customer]
}

type CustomersControllerBase struct {
	generic_error.ErrorsExtenderBase
	UserNameAndDescriptionControllerB[*Customer]
}
