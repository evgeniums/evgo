package tenancy_client

import (
	"github.com/evgeniums/go-utils/pkg/api/api_client"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/go-utils/pkg/op_context"
)

func (t *TenancyClient) Add(ctx op_context.Context, tenancy *multitenancy.TenancyData) (*multitenancy.TenancyItem, error) {

	// setup
	var err error
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
	err = handler.Exec(t.Client(), ctx, t.add)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.Result.TenancyItem, nil
}
