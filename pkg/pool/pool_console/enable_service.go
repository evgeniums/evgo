package pool_console

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/utils"
)

const EnableServiceCmd string = "enable_service"
const EnableServiceDescription string = "Enable service"

func EnableService() Handler {
	a := &EnableServiceHandler{}
	a.Init(EnableServiceCmd, EnableServiceDescription)
	return a
}

type EnableServiceData struct {
	Service string `long:"pool" description:"Short name of the service" required:"true"`
}

type EnableServiceHandler struct {
	HandlerBase
	EnableServiceData
}

func (a *EnableServiceHandler) Data() interface{} {
	return &a.EnableServiceData
}

func (a *EnableServiceHandler) Execute(args []string) error {

	ctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close()

	s, err := pool.ActivateService(controller, ctx, a.Service, true)
	if err == nil {
		fmt.Printf("Updated service:\n\n%s\n\n", utils.DumpPrettyJson(s))
	}

	return err
}
