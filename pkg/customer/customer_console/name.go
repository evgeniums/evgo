package customer_console

import (
	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/user/user_console"
)

const NameCmd string = "name"
const NameDescription string = "Set name"

func Name[T customer.User]() console_tool.Handler[*user_console.UserCommands[T]] {
	a := &NameHandler[T]{}
	a.Init(NameCmd, NameDescription)
	return a
}

type NameData struct {
	Name string `long:"name" description:"Name of the subject"`
}

type WithNameData struct {
	user_console.LoginData
	NameData
}

type NameHandler[T customer.User] struct {
	user_console.HandlerBase[T]
	WithNameData
}

func (a *NameHandler[T]) Data() interface{} {
	return &a.WithNameData
}

func (a *NameHandler[T]) Execute(args []string) error {

	ctx, sctx, ctrl, err := a.Context(a.Data(), a.Login)
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	setter, ok := ctrl.(customer.NameAndDescriptionSetter)
	if !ok {
		panic("Invalid type of user controller")
	}

	return setter.SetName(sctx, a.Login, a.Name, true)
}
