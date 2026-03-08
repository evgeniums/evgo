package tenancy_console

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/utils"
)

const AddCmd string = "add"
const AddDescription string = "Add tenancy"

func Add() Handler {
	a := &AddHandler{}
	a.Init(AddCmd, AddDescription)
	return a
}

type AddHandler struct {
	HandlerBase
	multitenancy.TenancyData
}

func (a *AddHandler) Data() interface{} {
	return &a.TenancyData
}

func (a *AddHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	added, err := controller.Add(sctx, &a.TenancyData)
	if err == nil {
		fmt.Printf("Added tenancy:\n%s\n", utils.DumpPrettyJson(added))
	}
	return err
}
