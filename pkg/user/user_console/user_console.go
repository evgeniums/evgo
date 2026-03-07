package user_console

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/utils"
)

type UserCommands[T user.User] struct {
	console_tool.Commands[*UserCommands[T]]
	MakeController func(app app_context.Context) user.Users[T]
}

func NewUserCommands[T user.User](groupName string, groupDescription string, controllerBuilder func(app app_context.Context) user.Users[T], loadDefaultHandlers ...bool) *UserCommands[T] {
	p := &UserCommands[T]{}
	p.Construct(p, groupName, groupDescription)
	p.MakeController = controllerBuilder
	if utils.OptionalArg(true, loadDefaultHandlers...) {
		p.LoadDefaultHandlers()
	}
	return p
}

func (p *UserCommands[T]) LoadDefaultHandlers() {

	p.AddHandlers(AddWithPhone[T],
		Password[T],
		Phone[T],
		Email[T],
		Block[T],
		Unblock[T],
		List[T],
		Show[T],
	)
}

type HandlerBase[T user.User] struct {
	console_tool.HandlerBase[*UserCommands[T]]
}

func (b *HandlerBase[T]) Context(data interface{}, login ...string) (op_context.Context, user.Users[T], error) {
	ctx, err := b.HandlerBase.Context(data)
	if err != nil {
		return ctx, nil, err
	}
	ctrl := b.Group.MakeController(ctx.App())
	if len(login) != 0 {
		err := ctrl.ValidateLogin(login[0])
		if err != nil {
			app_context.AbortFatal(ctx.App(), fmt.Sprintf("Invalid login format: %s", err))
		}
	}
	return ctx, ctrl, nil
}
