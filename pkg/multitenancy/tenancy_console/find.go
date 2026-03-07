package tenancy_console

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/utils"
)

const FindCmd string = "show"
const FindDescription string = "Show tenancy"

func Find() Handler {
	a := &FindHandler{}
	a.Init(FindCmd, FindDescription)
	return a
}

type TenancySelector struct {
	Customer string `long:"customer" description:"Name of customer the tenancy belongs to, used only if ID is not set"`
	Role     string `long:"role" description:"Role of tenancy, used only if ID is not set"`
	Id       string `long:"id" description:"ID of tenancy, if not set then customer and role will be used to look for tenancy"`
}

type FindHandler struct {
	HandlerBase
	TenancySelector
}

func (a *FindHandler) Data() interface{} {
	return &a.TenancySelector
}

func PrepareId(id string, customer string, role string) (string, bool) {
	iD := id
	idIsDisplay := false
	if iD == "" {
		iD = multitenancy.TenancySelector(customer, role)
		idIsDisplay = true
	} else {
		_, _, err := multitenancy.ParseTenancyDisplay(iD)
		idIsDisplay = err == nil
	}
	return iD, idIsDisplay
}

func (a *FindHandler) PrepareId() (string, bool) {
	return PrepareId(a.Id, a.Customer, a.Role)
}

func (a *FindHandler) Execute(args []string) error {

	ctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close()

	id, idIsDisplay := a.PrepareId()

	tenancy, err := controller.Find(ctx, id, idIsDisplay)
	if err == nil {
		fmt.Printf("Tenancy:\n%s\n", utils.DumpPrettyJson(tenancy))
	}
	return err
}
