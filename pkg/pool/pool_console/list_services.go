package pool_console

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/utils"
)

const ListServicesCmd string = "list_services"
const ListServicesDescription string = "List services"

func ListServices() Handler {
	a := &ListServicesHandler{}
	a.Init(ListServicesCmd, ListServicesDescription)
	return a
}

type ListServicesHandler struct {
	HandlerBase
}

func (a *ListServicesHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)
	services, _, err := controller.GetServices(sctx, nil)
	if err == nil {
		fmt.Printf("Services:\n\n%s\n\n", utils.DumpPrettyJson(services))
	}
	return err
}
