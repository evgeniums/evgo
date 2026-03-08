package tenancy_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

func (t *TenancyClient) Find(sctx context.Context, id string, idIsDisplay ...bool) (*multitenancy.TenancyItem, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("TenancyClient.Find")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// adjust ID
	tenancyId, tenancy, err := multitenancy.TenancyId(t, sctx, id, idIsDisplay...)
	if err != nil {
		c.SetMessage("failed to get tenancy ID")
		return nil, err
	}
	if tenancy != nil {
		return tenancy, nil
	}

	// prepare and exec handler
	handler := api_client.NewHandlerResult(&multitenancy.TenancyItem{})
	op := api.NamedResourceOperation(t.TenancyResource,
		tenancyId,
		tenancy_api.Find())
	err = handler.Exec(t.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.Result, nil
}
