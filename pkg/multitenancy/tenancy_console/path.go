package tenancy_console

import "github.com/evgeniums/evgo/pkg/multitenancy"

const PathCmd string = "path"
const PathDescription string = "Set new tenancy's path"

func Path() Handler {
	a := &PathHandler{}
	a.Init(PathCmd, PathDescription)
	return a
}

type PathData struct {
	TenancySelector
	multitenancy.WithPath
}

type PathHandler struct {
	HandlerBase
	PathData
}

func (a *PathHandler) Data() interface{} {
	return &a.PathData
}

func (a *PathHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	id, idIsDisplay := PrepareId(a.Id, a.Customer, a.Role)
	return controller.SetPath(sctx, id, a.PATH, idIsDisplay)
}
