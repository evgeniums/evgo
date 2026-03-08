package pool_console

const RemoveServiceFromAllPoolsCmd string = "remove_service_from_all_pools"
const RemoveServiceFromAllPoolsDescription string = "Remove service from all pools"

func RemoveServiceFromAllPools() Handler {
	a := &RemoveServiceFromAllPoolsHandler{}
	a.Init(RemoveServiceFromAllPoolsCmd, RemoveServiceFromAllPoolsDescription)
	return a
}

type RemoveServiceFromAllPoolsData struct {
	Service string `long:"service" description:"Short name of the service" required:"true"`
}

type RemoveServiceFromAllPoolsHandler struct {
	HandlerBase
	RemoveServiceFromAllPoolsData
}

func (a *RemoveServiceFromAllPoolsHandler) Data() interface{} {
	return &a.RemoveServiceFromAllPoolsData
}

func (a *RemoveServiceFromAllPoolsHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	return controller.RemoveServiceFromAllPools(sctx, a.Service, true)
}
