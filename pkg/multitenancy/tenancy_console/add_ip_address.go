package tenancy_console

import "github.com/evgeniums/evgo/pkg/multitenancy"

const AddIpAddressCmd string = "ip-add"
const AddIpAddressDescription string = "Add allowed IP address to tenancy"

func AddIpAddress() Handler {
	a := &AddIpAddressHandler{}
	a.Init(AddIpAddressCmd, AddIpAddressDescription)
	return a
}

type AddIpAddressHandler struct {
	FindHandler
	multitenancy.IpAddressCmd
}

func (a *AddIpAddressHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	id, idIsDisplay := a.PrepareId()
	return controller.AddIpAddress(sctx, id, a.Ip, a.Tag, idIsDisplay)
}
