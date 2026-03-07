package tenancy_console

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/utils"
)

const ListCmd string = "list"
const ListDescription string = "List tenancies"

func List() Handler {
	a := &ListHandler{}
	a.Init(ListCmd, ListDescription)
	return a
}

type ListHandler struct {
	HandlerBase
	console_tool.QueryData
}

func (a *ListHandler) Data() interface{} {
	return &a.QueryData
}

func (a *ListHandler) Execute(args []string) error {

	ctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close()

	filter, err := db.ParseQuery(ctx.Db(), a.Query, &multitenancy.TenancyItem{}, "")
	if err != nil {
		return fmt.Errorf("failed to parse query: %s", err)
	}

	tenancies, count, err := controller.List(ctx, filter)
	if err == nil {
		fmt.Printf("Tenancies:\n\n%s\n\nTotal count %d\n\n", utils.DumpPrettyJson(tenancies), count)
	}
	return err
}
