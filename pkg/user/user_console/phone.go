package user_console

import (
	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/user"
)

const PhoneCmd string = "phone"
const PhoneDescription string = "Set phone number"

func Phone[T user.User]() console_tool.Handler[*UserCommands[T]] {
	a := &PhoneHandler[T]{}
	a.Init(PhoneCmd, PhoneDescription)
	return a
}

type PhoneHandler[T user.User] struct {
	HandlerBase[T]
	WithPhoneData
}

func (a *PhoneHandler[T]) Data() interface{} {
	return &a.WithPhoneData
}

func (a *PhoneHandler[T]) Execute(args []string) error {

	ctx, sctx, ctrl, err := a.Context(a.Data(), a.Login)
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	return ctrl.SetPhone(sctx, a.Login, a.Phone, true)
}
