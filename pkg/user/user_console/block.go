package user_console

import (
	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/user"
)

const BlockCmd string = "block"
const BlockDescription string = "Block access"

func Block[T user.User]() console_tool.Handler[*UserCommands[T]] {
	a := &BlockHandler[T]{}
	a.Init(BlockCmd, BlockDescription)
	return a
}

type BlockHandler[T user.User] struct {
	HandlerBase[T]
	LoginData
}

func (a *BlockHandler[T]) Data() interface{} {
	return &a.LoginData
}

func (a *BlockHandler[T]) Execute(args []string) error {

	ctx, sctx, ctrl, err := a.Context(a.Data(), a.Login)
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	return ctrl.SetBlocked(sctx, a.Login, true, true)
}
