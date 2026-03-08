package tenancy_console

const ActivateCmd string = "activate"
const ActivateDescription string = "Activate tenancy"

const DeactivateCmd string = "deactivate"
const DeactivateDescription string = "Deactivate tenancy"

func Activate() Handler {
	a := &ActivateHandler{}
	a.Init(ActivateCmd, ActivateDescription)
	return a
}

type ActivateHandler struct {
	FindHandler
}

func (a *ActivateHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	id, idIsDisplay := a.PrepareId()
	return controller.Activate(sctx, id, idIsDisplay)
}

func Deactivate() Handler {
	a := &DeactivateHandler{}
	a.Init(DeactivateCmd, DeactivateDescription)
	return a
}

type DeactivateHandler struct {
	FindHandler
}

func (a *DeactivateHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	id, idIsDisplay := a.PrepareId()
	return controller.Deactivate(sctx, id, idIsDisplay)
}
