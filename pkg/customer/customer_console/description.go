package customer_console

import (
	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/user/user_console"
)

const DescriptionCmd string = "description"
const DescriptionDescription string = "Set description"

func Description[T customer.User]() console_tool.Handler[*user_console.UserCommands[T]] {
	a := &DescriptionHandler[T]{}
	a.Init(DescriptionCmd, DescriptionDescription)
	return a
}

type DescriptionData struct {
	Description string `long:"description" description:"Additional description"`
}

type WithDescriptionData struct {
	user_console.LoginData
	DescriptionData
}

type DescriptionHandler[T customer.User] struct {
	user_console.HandlerBase[T]
	WithDescriptionData
}

func (a *DescriptionHandler[T]) Data() interface{} {
	return &a.WithDescriptionData
}

func (a *DescriptionHandler[T]) Execute(args []string) error {

	ctx, ctrl, err := a.Context(a.Data(), a.Login)
	if err != nil {
		return err
	}
	defer ctx.Close()

	setter, ok := ctrl.(customer.NameAndDescriptionSetter)
	if !ok {
		panic("Invalid type of user controller")
	}

	return setter.SetDescription(ctx, a.Login, a.Description, true)
}
