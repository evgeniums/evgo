package customer_console

import (
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_console"
	"github.com/evgeniums/evgo/pkg/utils"
)

type Commands[T customer.User] struct {
	*user_console.UserCommands[T]
}

type Config[T customer.User] struct {
	ManagerBuilder func(app app_context.Context) user.Users[T]
	Name           string
	Description    string
}

func NewCommands[T customer.User](config Config[T], handlers ...console_tool.HandlerBuilder[*user_console.UserCommands[T]]) *Commands[T] {

	a := &Commands[T]{}
	a.UserCommands = user_console.NewUserCommands(config.Name, config.Description, config.ManagerBuilder, false)

	a.AddHandlers(handlers...)

	return a
}

type CustomerCommands = Commands[*customer.Customer]

func NewCustomerCommands(managerBuilder ...func(app app_context.Context) user.Users[*customer.Customer]) *CustomerCommands {

	config := Config[*customer.Customer]{
		Name:           "customer",
		Description:    "Manage customers",
		ManagerBuilder: utils.OptionalArg(DefaultCustomerManager, managerBuilder...),
	}

	// TODO refactor adding with field setters
	return NewCommands(config, user_console.AddNoPassword[*customer.Customer],
		user_console.Password[*customer.Customer],
		user_console.Phone[*customer.Customer],
		user_console.Email[*customer.Customer],
		user_console.Block[*customer.Customer],
		user_console.Unblock[*customer.Customer],
		user_console.List[*customer.Customer],
		user_console.Show[*customer.Customer],
		Name[*customer.Customer],
		Description[*customer.Customer])
}

func DefaultCustomerManager(app app_context.Context) user.Users[*customer.Customer] {
	manager := customer.NewManager()
	manager.Init(app.Validator())
	return manager
}
