package pool_console

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/utils"
)

const ShowServiceCmd string = "show_service"
const ShowServiceDescription string = "Show service"

func ShowService() Handler {
	a := &ShowServiceHandler{}
	a.Init(ShowServiceCmd, ShowServiceDescription)
	return a
}

type ShowServiceData struct {
	Service string `long:"service" description:"Short name of the service" required:"true"`
}

type ShowServiceHandler struct {
	HandlerBase
	ShowServiceData
}

func (a *ShowServiceHandler) Data() interface{} {
	return &a.ShowServiceData
}

func (a *ShowServiceHandler) Execute(args []string) error {

	ctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close()
	service, err := controller.FindService(ctx, a.Service, true)
	if err == nil {
		if service != nil {
			fmt.Printf("Service:\n\n%s\n\n", utils.DumpPrettyJson(service))
		} else {
			fmt.Println("Service not found")
		}
	}
	return err
}
