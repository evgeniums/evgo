package pool_console

const RemoveAllServicesFromPoolCmd string = "remove_all_services_from_pool"
const RemoveAllServicesFromPoolDescription string = "Remove all services from pool"

func RemoveAllServicesFromPool() Handler {
	a := &RemoveAllServicesFromPoolHandler{}
	a.Init(RemoveAllServicesFromPoolCmd, RemoveAllServicesFromPoolDescription)
	return a
}

type RemoveAllServicesFromPoolData struct {
	Pool string `long:"pool" description:"Short name of the pool" required:"true"`
}

type RemoveAllServicesFromPoolHandler struct {
	HandlerBase
	RemoveAllServicesFromPoolData
}

func (a *RemoveAllServicesFromPoolHandler) Data() interface{} {
	return &a.RemoveAllServicesFromPoolData
}

func (a *RemoveAllServicesFromPoolHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	return controller.RemoveAllServicesFromPool(sctx, a.Pool, true)
}
