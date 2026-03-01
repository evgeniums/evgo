package user_service

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/user"
	"github.com/evgeniums/go-utils/pkg/user/user_api"
)

type UserEndpoint[U user.User] struct {
	service *UserService[U]
}

type UserService[U user.User] struct {
	api_server.ServiceBase
	Users        user.Users[U]
	UserTypeName string

	collectionResource api.Resource
	userResource       api.Resource

	listOperation api.Operation
}

func NewUserService[U user.User](userController user.Users[U],
	setterBuilder func() user.UserFieldsSetter[U],
	userTypeName ...string) *UserService[U] {

	s := &UserService[U]{}

	s.ErrorsExtenderBase.Init(user.ErrorDescriptions, user.ErrorHttpCodes)

	userType, serviceName, collectionResource, userResource := user_api.PrepareResources(userTypeName...)
	s.collectionResource = collectionResource
	s.userResource = userResource

	s.Init(serviceName, api.PackageName)
	s.UserTypeName = userType

	s.Users = userController
	s.AddChild(s.collectionResource)

	listOp := List(s)
	s.listOperation = listOp

	s.collectionResource.AddOperation(listOp)
	s.collectionResource.AddOperation(Add(s, setterBuilder))

	s.userResource.AddOperation(Find(s), true)
	s.userResource.AddChild(SetPhone(s.UserTypeName, s.Users))
	s.userResource.AddChild(SetEmail(s.UserTypeName, s.Users))
	s.userResource.AddChild(SetBlocked(s.UserTypeName, s.Users))
	s.userResource.AddChild(SetPassword(s.UserTypeName, s.Users))

	return s
}

func (s *UserService[U]) CollectionResource() api.Resource {
	return s.collectionResource
}

func (s *UserService[U]) UserResource() api.Resource {
	return s.userResource
}

func (s *UserService[U]) ListOperation() api.Operation {
	return s.listOperation
}

type SetUserFieldEndpoint struct {
	userTypeName string
	api_server.ResourceEndpoint
	users user.MainFieldSetters
}

func (e *SetUserFieldEndpoint) Init(ep api_server.ResourceEndpointI, userTypeName string, fieldName string, users user.MainFieldSetters, op api.Operation) api_server.ResourceEndpointI {
	api_server.ConstructResourceEndpoint(ep, fieldName, op)
	e.users = users
	e.userTypeName = userTypeName
	return ep
}

type TenancyWithUserSetter interface {
	UserFieldSetter() user.MainFieldSetters
}

type TenancyWithUsers[U user.User] interface {
	UserController() user.Users[U]
}

func Users[U user.User](service *UserService[U], request api_server.Request) user.Users[U] {

	t := request.GetTenancy()
	if t != nil {
		ts, ok := t.(TenancyWithUsers[U])
		if ok {
			return ts.UserController()
		}
	}

	return service.Users
}

func Setter(setters user.MainFieldSetters, request api_server.Request) user.MainFieldSetters {

	t := request.GetTenancy()
	if t != nil {
		ts, ok := t.(TenancyWithUserSetter)
		if ok {
			return ts.UserFieldSetter()
		}
	}

	return setters
}
