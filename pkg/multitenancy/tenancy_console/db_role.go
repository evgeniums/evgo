package tenancy_console

import "github.com/evgeniums/evgo/pkg/multitenancy"

const DbRoleCmd string = "db-role"
const DbRoleDescription string = "Set role of database service to be used by tenancy"

func DbRole() Handler {
	a := &DbRoleHandler{}
	a.Init(DbRoleCmd, DbRoleDescription)
	return a
}

type DbRoleData struct {
	TenancySelector
	multitenancy.WithDbRole
}

type DbRoleHandler struct {
	HandlerBase
	DbRoleData
}

func (a *DbRoleHandler) Data() interface{} {
	return &a.DbRoleData
}

func (a *DbRoleHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	id, idIsDisplay := PrepareId(a.Id, a.Customer, a.Role)
	return controller.SetDbRole(sctx, id, a.DbRole(), idIsDisplay)
}
