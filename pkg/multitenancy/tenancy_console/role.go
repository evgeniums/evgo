package tenancy_console

const RoleCmd string = "role"
const RoleDescription string = "Set new tenancy's role"

func Role() Handler {
	a := &RoleHandler{}
	a.Init(RoleCmd, RoleDescription)
	return a
}

type RoleData struct {
	TenancySelector
	NewRole string `long:"new-role" description:"New role"`
}

type RoleHandler struct {
	HandlerBase
	RoleData
}

func (a *RoleHandler) Data() interface{} {
	return &a.RoleData
}

func (a *RoleHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	id, idIsDisplay := PrepareId(a.Id, a.Role, a.Role)
	return controller.SetRole(sctx, id, a.NewRole, idIsDisplay)
}
