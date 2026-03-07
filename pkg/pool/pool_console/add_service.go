package pool_console

import (
	"errors"
	"fmt"

	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/utils"
)

const AddServiceCmd string = "add_service"
const AddServiceDescription string = "Add service"

func AddService() Handler {
	a := &AddServiceHandler{}
	a.Init(AddServiceCmd, AddServiceDescription)
	return a
}

type AddServiceData struct {
	pool.ServiceConfigBase
	pool.SecretsBase
	Name        string `long:"name" description:"Short name of the service, must be unique" required:"true"`
	Type        string `long:"type" description:"Service type" required:"true"`
	LongName    string `long:"long-name" description:"Long name of the service"`
	Description string `long:"description" description:"Service description"`
	Active      string `long:"active" description:"Service is active" default:"true"`

	Pool string `long:"pool" description:"Short name of the pool where to add service to"`
	Role string `long:"role" description:"Role of the service in the pool, must be unique per the pool and alphanumeric"`
}

type AddServiceHandler struct {
	HandlerBase
	AddServiceData
}

func (a *AddServiceHandler) Data() interface{} {
	return &a.AddServiceData
}

func (a *AddServiceHandler) Execute(args []string) error {

	ctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close()

	if a.SECRET1 != "" {
		a.SECRET1 = console_tool.ReadPassword("Please, enter secret 1:")
	}
	if a.SECRET2 != "" {
		a.SECRET2 = console_tool.ReadPassword("Please, enter secret 2:")
	}

	s := pool.NewService()
	s.SetName(a.Name)
	s.SetDescription(a.Description)
	s.SetLongName(a.LongName)
	s.SetTypeName(a.Type)
	s.PoolServiceBaseData.ServiceConfigBase = a.ServiceConfigBase
	s.SecretsBase = a.SecretsBase
	s.SetActive(a.Active == "true")

	addedService, err := controller.AddService(ctx, s)
	if err != nil {
		return err
	}

	fmt.Printf("Added service:\n%s\n", utils.DumpPrettyJson(addedService))

	// add service to pool
	if a.Pool != "" {
		if a.Role == "" {
			err = errors.New("role must be specified")
			return err
		}

		err = controller.AddServiceToPool(ctx, a.Pool, a.Name, a.Role, true)
		if err != nil {
			return err
		}

		fmt.Println("Service added to pool")
	}

	return nil
}
