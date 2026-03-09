package user_client

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_api"
)

type SetterHandler[T interface{}] struct {
	Cmd T
}

func (s *SetterHandler[T]) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {
	return client.Exec(sctx, operation, s.Cmd, nil)
}

type UserBuilder[U user.User] func() U

type UserClient[U user.User] struct {
	api_client.ServiceClient
	userTypeName string
	userBuilder  UserBuilder[U]

	CollectionResource api.Resource
	UserResource       api.Resource

	add  api.Operation
	list api.Operation
}

func NewUserClient[U user.User](client api_client.Client,
	userBuilder UserBuilder[U],
	userTypeName ...string) *UserClient[U] {

	var serviceName string
	c := &UserClient[U]{}
	c.userTypeName, serviceName, c.CollectionResource, c.UserResource = user_api.PrepareResources(userTypeName...)
	c.ServiceClient.Init(client, serviceName)

	c.AddChild(c.CollectionResource)
	c.add = user_api.Add(c.userTypeName)
	c.list = user_api.List()
	c.CollectionResource.AddOperations(c.add, c.list)

	return c
}

func (c *UserClient[U]) ApplyVisitors(
	visitors ...api.OperationVisitors,
) {
	api.VisitOperation(c.add, visitors...)
	api.VisitOperation(c.list, visitors...)
}

func (c *UserClient[U]) OpLog(sctx context.Context, op string, userId string, login string) {}

func (c *UserClient[U]) SetTenancy(tenancyResource api.Resource) {
	tenancyResource.AddChild(c)
}

func (c *UserClient[U]) SetUserBuilder(userBuilder func() U) {
	c.userBuilder = userBuilder
}

func (c *UserClient[U]) SetOplogBuilder(userBuilder func() user.OpLogUserI) {
}

func (c *UserClient[U]) MakeUser() U {
	return c.userBuilder()
}

func (u *UserClient[U]) UserOperation(userId string, resourceName string, op api.Operation) api.Operation {
	opResource := api.NewResource(resourceName)
	opResource.AddOperation(op)
	userResource := u.UserResource.CloneChain(false)
	userResource.SetId(userId)
	userResource.AddChild(opResource)
	return op
}

func (u *UserClient[U]) FindAuthUser(sctx context.Context, login string, parameters ...map[string]string) (auth.User, error) {
	return nil, errors.New("unsupported method")
}

func (a *UserClient[U]) FillAuthUser(sctx context.Context, useExistingSessisonParams ...bool) error {
	return nil
}
