package tenancy_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

func (t *TenancyClient) Add(sctx context.Context, tenancy *multitenancy.TenancyData) (*multitenancy.TenancyItem, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("TenancyClient.Add")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// prepare and exec handler
	handler := api_client.NewHandler(tenancy, &tenancy_api.TenancyResponse{})
	err = handler.Exec(t.Client(), sctx, t.add)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.Result.TenancyItem, nil
}
