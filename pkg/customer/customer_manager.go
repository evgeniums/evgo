package customer

import (
	"github.com/evgeniums/evgo/pkg/auth/auth_session"
	"github.com/evgeniums/evgo/pkg/user"
)

type Manager struct {
	CustomersBase
	auth_session.SessionController
}

func (m *Manager) SessionManager() auth_session.SessionController {
	return m
}

type CustomersBase struct {
	user.UsersValidator
	auth_session.AuthUserFinder
	CustomerController
}

func (c *CustomersBase) Construct(customerController CustomerController, authUserFinder ...auth_session.AuthUserFinder) {
	c.CustomerController = customerController
	if len(authUserFinder) != 0 {
		c.AuthUserFinder = authUserFinder[0]
	} else {
		c.AuthUserFinder = user.NewAuthUserFinder(func() user.User { return c.CustomerController.MakeUser() })
	}
}

func (m *CustomersBase) AuthUserManager() auth_session.AuthUserManager {
	return m
}

type ManagerConfig struct {
	CustomerController CustomerController
	SessionController  auth_session.SessionController
}

func NewManager(config ...ManagerConfig) *Manager {

	m := &Manager{}

	if len(config) != 0 {
		m.CustomersBase.Construct(config[0].CustomerController)
		m.SessionController = config[0].SessionController
	}
	if m.CustomerController == nil {
		c := LocalCustomerController()
		m.CustomersBase.Construct(c)
		c.SetUserValidators(m)
	}
	if m.SessionController == nil {
		m.SessionController = auth_session.LocalSessionController()
	}

	m.SetUserBuilder(NewCustomer)
	m.SetSessionBuilder(func() auth_session.Session { return NewCustomerSession() })
	m.SetSessionClientBuilder(func() auth_session.SessionClient { return NewCustomerSessionClient() })
	m.SetOplogBuilder(NewOplog)

	return m
}
