package pool_console

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/utils"
)

const EnablePoolCmd string = "enable_pool"
const EnablePoolDescription string = "Enable pool"

func EnablePool() Handler {
	a := &EnablePoolHandler{}
	a.Init(EnablePoolCmd, EnablePoolDescription)
	return a
}

type EnablePoolData struct {
	Pool string `long:"pool" description:"Short name of the pool" required:"true"`
}

type EnablePoolHandler struct {
	HandlerBase
	EnablePoolData
}

func (a *EnablePoolHandler) Data() interface{} {
	return &a.EnablePoolData
}

func (a *EnablePoolHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	p, err := pool.ActivatePool(controller, sctx, a.Pool, true)
	if err == nil {
		fmt.Printf("Updated pool:\n\n%s\n\n", utils.DumpPrettyJson(p))
	}
	return err
}
