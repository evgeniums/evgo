package admin_api

import (
	"context"

	"github.com/evgeniums/evgo/pkg/admin"
	"github.com/evgeniums/evgo/pkg/user"
)

type AdminFieldsSetter struct {
	user.UserFieldsSetterBase[*admin.Admin]
}

func (a *AdminFieldsSetter) SetUserFields(sctx context.Context, admin *admin.Admin) ([]user.CheckDuplicateField, error) {
	return a.UserFieldsSetterBase.SetUserFields(sctx, admin)
}

func NewAdminFieldsSetter() user.UserFieldsSetter[*admin.Admin] {
	s := &AdminFieldsSetter{}
	return s
}
