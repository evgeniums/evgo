package tenancy_console

import "github.com/evgeniums/evgo/pkg/multitenancy"

const SetPathBlockedCmd string = "block-path"
const SetPathBlockedDescription string = "Block tenancy path(s)"

func SetPathBlocked() Handler {
	a := &SetPathBlockedHandler{}
	a.Init(SetPathBlockedCmd, SetPathBlockedDescription)
	return a
}

type SetPathBlockedData struct {
	TenancySelector
	multitenancy.BlockPathCmd
}

type SetPathBlockedHandler struct {
	HandlerBase
	SetPathBlockedData
}

func (a *SetPathBlockedHandler) Data() interface{} {
	return &a.SetPathBlockedData
}

func (a *SetPathBlockedHandler) Execute(args []string) error {

	ctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close()

	id, idIsDisplay := PrepareId(a.Id, a.Customer, a.Role)
	return controller.SetPathBlocked(ctx, id, a.Block, a.Mode, idIsDisplay)
}
