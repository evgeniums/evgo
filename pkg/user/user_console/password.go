package user_console

import (
	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/user"
)

const PasswordCmd string = "password"
const PasswordDescription string = "Set new password"

func Password[T user.User]() console_tool.Handler[*UserCommands[T]] {
	a := &PasswordHandler[T]{}
	a.Init(PasswordCmd, PasswordDescription)
	return a
}

type PasswordHandler[T user.User] struct {
	HandlerBase[T]
	LoginData
}

func (a *PasswordHandler[T]) Data() interface{} {
	return &a.LoginData
}

func (a *PasswordHandler[T]) Execute(args []string) error {

	ctx, sctx, ctrl, err := a.Context(a.Data(), a.Login)
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	password := console_tool.ReadPassword()
	return ctrl.SetPassword(sctx, a.Login, password, true)
}
