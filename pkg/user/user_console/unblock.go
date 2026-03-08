package user_console

import (
	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/user"
)

const UnblockCmd string = "unblock"
const UnblockDescription string = "Unblock access"

func Unblock[T user.User]() console_tool.Handler[*UserCommands[T]] {
	a := &UnblockHandler[T]{}
	a.Init(UnblockCmd, UnblockDescription)
	return a
}

type UnblockHandler[T user.User] struct {
	HandlerBase[T]
	LoginData
}

func (a *UnblockHandler[T]) Data() interface{} {
	return &a.LoginData
}

func (a *UnblockHandler[T]) Execute(args []string) error {

	ctx, sctx, ctrl, err := a.Context(a.Data(), a.Login)
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	return ctrl.SetBlocked(sctx, a.Login, false, true)
}
