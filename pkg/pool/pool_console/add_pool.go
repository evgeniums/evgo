package pool_console

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/utils"
)

const AddPoolCmd string = "add_pool"
const AddPoolDescription string = "Add pool"

func AddPool() Handler {
	a := &AddPoolHandler{}
	a.Init(AddPoolCmd, AddPoolDescription)
	return a
}

type AddPoolData struct {
	Name        string `long:"name" description:"Short name of the pool, must be unique" required:"true"`
	LongName    string `long:"long-name" description:"Long name of the pool"`
	Description string `long:"description" description:"Pool description"`
}

type AddPoolHandler struct {
	HandlerBase
	AddPoolData
}

func (a *AddPoolHandler) Data() interface{} {
	return &a.AddPoolData
}

func (a *AddPoolHandler) Execute(args []string) error {

	ctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close()

	p := pool.NewPool()
	p.SetName(a.Name)
	p.SetDescription(a.Description)
	p.SetLongName(a.LongName)

	addedPool, err := controller.AddPool(ctx, p)
	if err == nil {
		fmt.Printf("Added pool:\n%s\n", utils.DumpPrettyJson(addedPool))
	}
	return err
}
