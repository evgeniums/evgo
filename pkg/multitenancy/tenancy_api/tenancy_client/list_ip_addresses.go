package tenancy_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

func (t *TenancyClient) ListIpAddresses(sctx context.Context, filter *db.Filter) ([]*multitenancy.TenancyIpAddressItem, int64, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("TenancyClient.ListIpAddresses")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// set query
	cmd := api.NewDbQuery(filter)

	// prepare and exec handler
	handler := api_client.NewHandler(cmd, &tenancy_api.ListIpAddressesResponse{})
	err = handler.Exec(t.Client(), sctx, t.list_ip_addresses)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, 0, err
	}

	// done
	return handler.Result.Items, handler.Result.Count, nil
}
