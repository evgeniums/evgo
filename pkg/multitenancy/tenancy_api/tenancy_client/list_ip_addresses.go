package tenancy_client

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_client"
	"github.com/evgeniums/go-utils/pkg/db"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/go-utils/pkg/op_context"
)

func (t *TenancyClient) ListIpAddresses(ctx op_context.Context, filter *db.Filter) ([]*multitenancy.TenancyIpAddressItem, int64, error) {

	// setup
	var err error
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
	err = handler.Exec(t.Client(), ctx, t.list_ip_addresses)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, 0, err
	}

	// done
	return handler.Result.Items, handler.Result.Count, nil
}
