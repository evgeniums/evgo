package user_api

import (
	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/utils"
)

func PrepareResources(userTypeName ...string) (userType string, serviceName string, collectionResource api.Resource, userResource api.Resource) {

	userType = utils.OptionalArg("user", userTypeName...)
	serviceName = utils.ConcatStrings(userType, "s")

	userResource = UserResource(userType)
	collectionResource = userResource.Parent()

	return
}

func NamedUserResource(id string, userTypeName ...string) (userResource api.Resource) {
	r := UserResource(userTypeName...)
	r.SetId(id)
	return r
}

func UserResource(resourceType ...string) api.Resource {
	return api.NamedResource(utils.OptionalArg("user", resourceType...))
}

type UserResponse[T user.User] struct {
	api.ResponseBase
	User T `json:"user"`
}

func List() api.Operation {
	return api.NewOperation("list", access_control.Read)
}

func Add(name string) api.Operation {
	return api.NewOperation(utils.ConcatStrings("add_", name), access_control.Create)
}

func Find(name string) api.Operation {
	return api.NewOperation(utils.ConcatStrings("find_", name), access_control.Get)
}

func SetPassword(name string) api.Operation {
	return api.NewOperation(utils.ConcatStrings("find_", name, "_password"), access_control.Put)
}

func SetEmail(name string) api.Operation {
	return api.NewOperation(utils.ConcatStrings("find_", name, "_email"), access_control.Put)
}

func SetPhone(name string) api.Operation {
	return api.NewOperation(utils.ConcatStrings("find_", name, "_phone"), access_control.Put)
}

func SetBlocked(name string) api.Operation {
	return api.NewOperation(utils.ConcatStrings("find_", name, "_blocked"), access_control.Put)
}
