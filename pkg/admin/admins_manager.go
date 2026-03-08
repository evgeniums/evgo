package admin

import (
	"context"

	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/user"
)

type Manager struct {
	*user.UsersWithSessionBase[*Admin, *AdminSession, *AdminSessionClient]
}

type AdminControllers = user.UsersWithSessionBaseConfig[*Admin]

func NewManager(controllers ...AdminControllers) *Manager {
	m := &Manager{UsersWithSessionBase: user.NewUsersWithSession(NewAdmin, NewAdminSession, NewAdminSessionClient, NewOplog, controllers...)}
	return m
}

func (m *Manager) AddAdmin(sctx context.Context, login string, password string, phone string) (*Admin, error) {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("AddAdmin")
	defer ctx.TraceOutMethod()

	admin, err := m.UsersWithSessionBase.Add(sctx, login, password, user.Phone(phone, &Admin{}))
	return admin, c.SetError(err)
}
