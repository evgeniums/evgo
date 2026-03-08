package pool_console

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/utils"
)

const ShowPoolCmd string = "show_pool"
const ShowPoolDescription string = "Show pool"

func ShowPool() Handler {
	a := &ShowPoolHandler{}
	a.Init(ShowPoolCmd, ShowPoolDescription)
	return a
}

type ShowPoolData struct {
	Pool string `long:"pool" description:"Short name of the pool" required:"true"`
}

type ShowPoolHandler struct {
	HandlerBase
	ShowPoolData
}

func (a *ShowPoolHandler) Data() interface{} {
	return &a.ShowPoolData
}

func (a *ShowPoolHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)
	pool, err := controller.FindPool(sctx, a.Pool, true)
	if err == nil {
		if pool != nil {
			fmt.Printf("Pool:\n\n%s\n\n", utils.DumpPrettyJson(pool))
		} else {
			fmt.Println("Pool not found")
		}
	}
	return err
}
