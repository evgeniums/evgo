package admin_api

import (
	"github.com/evgeniums/evgo/pkg/admin"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
)

type AdminFieldsSetter struct {
	user.UserFieldsSetterBase[*admin.Admin]
}

func (a *AdminFieldsSetter) SetUserFields(ctx op_context.Context, admin *admin.Admin) ([]user.CheckDuplicateField, error) {
	return a.UserFieldsSetterBase.SetUserFields(ctx, admin)
}

func NewAdminFieldsSetter() user.UserFieldsSetter[*admin.Admin] {
	s := &AdminFieldsSetter{}
	return s
}
