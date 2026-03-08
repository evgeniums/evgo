package pool_console

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/utils"
)

const ListPoolsCmd string = "list_pools"
const ListPoolsDescription string = "List pools"

func ListPools() Handler {
	a := &ListPoolsHandler{}
	a.Init(ListPoolsCmd, ListPoolsDescription)
	return a
}

type ListPoolsHandler struct {
	HandlerBase
}

func (a *ListPoolsHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)
	pools, _, err := controller.GetPools(sctx, nil)
	if err == nil {
		fmt.Printf("Pools:\n\n%s\n\n", utils.DumpPrettyJson(pools))
	}
	return err
}
