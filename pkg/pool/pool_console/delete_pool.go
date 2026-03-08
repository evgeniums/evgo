package pool_console

const DeletePoolCmd string = "delete_pool"
const DeletePoolDescription string = "Delete pool"

func DeletePool() Handler {
	a := &DeletePoolHandler{}
	a.Init(DeletePoolCmd, DeletePoolDescription)
	return a
}

type DeletePoolData struct {
	Pool string `long:"pool" description:"Short name of the pool" required:"true"`
}

type DeletePoolHandler struct {
	HandlerBase
	DeletePoolData
}

func (d *DeletePoolHandler) Data() interface{} {
	return &d.DeletePoolData
}

func (d *DeletePoolHandler) Execute(args []string) error {

	ctx, sctx, controller, err := d.Context(d.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	return controller.DeletePool(sctx, d.Pool, true)
}
