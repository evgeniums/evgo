package customer_api

import (
	"context"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/user"
)

type FieldsSetter[T customer.User] struct {
	user.UserFieldsSetterBase[T]
	common.WithNameBase
	common.WithDescriptionBase
}

func (c *FieldsSetter[T]) SetUserFields(sctx context.Context, user T) ([]user.CheckDuplicateField, error) {
	user.SetName(c.Name())
	user.SetDescription(c.Description())
	return c.UserFieldsSetterBase.SetUserFields(sctx, user)
}

func NewFieldsSetter[T customer.User]() user.UserFieldsSetter[T] {
	s := &FieldsSetter[T]{}
	return s
}

func SetName() api.Operation {
	return api.NewOperation("set_name", access_control.Put)
}

func SetDescription() api.Operation {
	return api.NewOperation("set_description", access_control.Put)
}
