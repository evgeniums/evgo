package tenancy_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

func (t *TenancyClient) Delete(sctx context.Context, id string, withDb bool, idIsDisplay ...bool) error {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("TenancyClient.Delete")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// adjust ID
	tenancyId, _, err := multitenancy.TenancyId(t, sctx, id, idIsDisplay...)
	if err != nil {
		c.SetMessage("failed to get tenancy ID")
		return err
	}

	// prepare and exec handler
	handler := api_client.NewHandlerRequest(&tenancy_api.DeleteTenancyCmd{WithDatabase: withDb})
	op := api.NamedResourceOperation(t.TenancyResource,
		tenancyId,
		tenancy_api.Delete())
	err = handler.Exec(t.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return err
	}

	// done
	return nil
}
