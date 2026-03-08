package pool_console

const AddServiceToPoolCmd string = "add_service_to_pool"
const AddServiceToPoolDescription string = "Add service to pool"

func AddServiceToPool() Handler {
	a := &AddServiceToPoolHandler{}
	a.Init(AddServiceToPoolCmd, AddServiceToPoolDescription)
	return a
}

type AddServiceToPoolData struct {
	Pool    string `long:"pool" description:"Short name of the pool" required:"true"`
	Service string `long:"service" description:"Short name of the service" required:"true"`
	Role    string `long:"role" description:"Role of the service in the pool, must be unique per the pool and alphanumeric" required:"true"`
}

type AddServiceToPoolHandler struct {
	HandlerBase
	AddServiceToPoolData
}

func (a *AddServiceToPoolHandler) Data() interface{} {
	return &a.AddServiceToPoolData
}

func (a *AddServiceToPoolHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	return controller.AddServiceToPool(sctx, a.Pool, a.Service, a.Role, true)
}
