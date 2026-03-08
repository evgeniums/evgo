package tenancy_console

import "github.com/evgeniums/evgo/pkg/multitenancy"

const DeleteIpAddressCmd string = "ip-delete"
const DeleteIpAddressDescription string = "Delete allowed IP address to tenancy"

func DeleteIpAddress() Handler {
	a := &DeleteIpAddressHandler{}
	a.Init(DeleteIpAddressCmd, DeleteIpAddressDescription)
	return a
}

type DeleteIpAddressHandler struct {
	FindHandler
	multitenancy.IpAddressCmd
}

func (a *DeleteIpAddressHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	id, idIsDisplay := a.PrepareId()
	return controller.DeleteIpAddress(sctx, id, a.Ip, a.Tag, idIsDisplay)
}
