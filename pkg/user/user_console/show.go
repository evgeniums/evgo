package user_console

import (
	"encoding/json"
	"fmt"

	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/utils"
)

const ShowCmd string = "show"
const ShowDescription string = "Show object"

func Show[T user.User]() console_tool.Handler[*UserCommands[T]] {
	a := &ListHandler[T]{}
	a.Init(ShowCmd, ShowDescription)
	return a
}

type ShowHandler[T user.User] struct {
	HandlerBase[T]
	LoginData
}

func (a *ShowHandler[T]) Data() interface{} {
	return &a.LoginData
}

func (a *ShowHandler[T]) Execute(args []string) error {

	ctx, sctx, ctrl, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	user, err := ctrl.FindByLogin(sctx, a.Login)
	if err != nil {
		return err
	}
	if utils.IsNil(user) {
		return fmt.Errorf("user not found")
	}

	b, err := json.MarshalIndent(user, "", "   ")
	if err != nil {
		return fmt.Errorf("failed to serialize result: %s", err)
	}
	fmt.Printf("\n\n%s\n\n", string(b))
	return nil
}
