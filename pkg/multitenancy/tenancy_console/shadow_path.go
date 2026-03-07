package tenancy_console

import "github.com/evgeniums/evgo/pkg/multitenancy"

const ShadowPathCmd string = "shadow-path"
const ShadowPathDescription string = "Set new tenancy's shadow path"

func ShadowPath() Handler {
	a := &ShadowPathHandler{}
	a.Init(ShadowPathCmd, ShadowPathDescription)
	return a
}

type ShadowPathData struct {
	TenancySelector
	multitenancy.WithPath
}

type ShadowPathHandler struct {
	HandlerBase
	ShadowPathData
}

func (a *ShadowPathHandler) Data() interface{} {
	return &a.ShadowPathData
}

func (a *ShadowPathHandler) Execute(args []string) error {

	ctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close()

	path := a.SHADOW_PATH
	if path == "" {
		path = a.PATH
	}

	id, idIsDisplay := PrepareId(a.Id, a.Customer, a.Role)
	return controller.SetShadowPath(ctx, id, a.SHADOW_PATH, idIsDisplay)
}
