package pool_console

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/utils"
)

const DisablePoolCmd string = "disable_pool"
const DisablePoolDescription string = "Disable pool"

func DisablePool() Handler {
	d := &DisablePoolHandler{}
	d.Init(DisablePoolCmd, DisablePoolDescription)
	return d
}

type DisablePoolData struct {
	Pool string `long:"pool" description:"Short name of the pool" required:"true"`
}

type DisablePoolHandler struct {
	HandlerBase
	DisablePoolData
}

func (d *DisablePoolHandler) Data() interface{} {
	return &d.DisablePoolData
}

func (d *DisablePoolHandler) Execute(args []string) error {

	ctx, sctx, controller, err := d.Context(d.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	p, err := pool.DeactivatePool(controller, sctx, d.Pool, true)
	if err == nil {
		fmt.Printf("Updated pool:\n\n%s\n\n", utils.DumpPrettyJson(p))
	}

	return err
}
