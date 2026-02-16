package customer_api_service

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/customer"
	"github.com/evgeniums/go-utils/pkg/customer/customer_api"
	"github.com/evgeniums/go-utils/pkg/user"
	"github.com/evgeniums/go-utils/pkg/user/user_api/user_service"
	"github.com/evgeniums/go-utils/pkg/utils"
)

type Service[T customer.User] struct {
	*user_service.UserService[T]
	Controller customer.UserNameAndDescriptionController[T]
}

func NewService[T customer.User](users user.Users[T], userTypeName ...string) *Service[T] {
	return NewServiceExtended(users, customer_api.NewFieldsSetter[T], userTypeName...)
}

func NewServiceExtended[T customer.User](users user.Users[T], setterBuilder func() user.UserFieldsSetter[T], userTypeName ...string) *Service[T] {

	c := &Service[T]{}
	c.UserService = user_service.NewUserService(users,
		setterBuilder,
		utils.OptionalArg("customer", userTypeName...))
	c.Users = users
	c.UserService.UserResource().AddChildren(SetName(c), SetDescription(c))

	return c
}

type Endpoint[T customer.User] struct {
	api_server.ResourceEndpoint
	service *Service[T]
}

func (e *Endpoint[T]) Init(ep api_server.ResourceEndpointI, fieldName string, s *Service[T], op api.Operation) api_server.ResourceEndpointI {
	api_server.ConstructResourceEndpoint(ep, fieldName, op)
	e.service = s
	return ep
}

type CustomerService = Service[*customer.Customer]

func NewCustomerService(customers *customer.Manager) *CustomerService {

	s := NewService(customers)

	customerTableConfig := &api_server.DynamicTableConfig{Model: &customer.Customer{}, Operation: s.ListOperation()}
	s.AddDynamicTables(customerTableConfig)

	s.AppendErrorExtender(customers.CustomerController)

	return s
}

type TenancyWithSetters interface {
	CustomerFieldSetter() customer.NameAndDescriptionSetter
}

func Setter(setter customer.NameAndDescriptionSetter, request api_server.Request) customer.NameAndDescriptionSetter {

	t := request.GetTenancy()
	if t != nil {
		ts, ok := t.(TenancyWithSetters)
		if ok {
			return ts.CustomerFieldSetter()
		}
	}

	return setter
}
