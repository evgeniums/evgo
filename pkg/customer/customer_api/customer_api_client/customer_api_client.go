package customer_api_client

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/user/user_api/user_client"
	"github.com/evgeniums/evgo/pkg/utils"
)

type Client[T customer.User] struct {
	*user_client.UserClient[T]
}

func NewClient[T customer.User](client api_client.Client, userBuilder user_client.UserBuilder[T], userTypeName ...string) *Client[T] {
	c := &Client[T]{}
	c.UserClient = user_client.NewUserClient(client, userBuilder, utils.OptionalArg("customer", userTypeName...))
	return c
}

type CustomerClient = Client[*customer.Customer]

func NewCustomerClient(client api_client.Client, visitors ...api.OperationVisitors) *CustomerClient {
	c := NewClient(client, customer.NewCustomer)
	c.ApplyVisitors(visitors...)
	return c
}
