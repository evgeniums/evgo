package pool_console

const DeleteServiceCmd string = "delete_service"
const DeleteServiceDescription string = "Delete service"

func DeleteService() Handler {
	a := &DeleteServiceHandler{}
	a.Init(DeleteServiceCmd, DeleteServiceDescription)
	return a
}

type DeleteServiceData struct {
	Service string `long:"service" description:"Short name of the service" required:"true"`
}

type DeleteServiceHandler struct {
	HandlerBase
	DeleteServiceData
}

func (d *DeleteServiceHandler) Data() interface{} {
	return &d.DeleteServiceData
}

func (d *DeleteServiceHandler) Execute(args []string) error {

	ctx, sctx, controller, err := d.Context(d.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	return controller.DeleteService(sctx, d.Service, true)
}
