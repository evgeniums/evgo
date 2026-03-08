package tenancy_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

func (t *TenancyClient) ChangePoolOrDb(sctx context.Context, id string, poolId string, dbName string, idIsDisplay ...bool) error {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("TenancyClient.ChangePoolOrDb")
	defer ctx.TraceOutMethod()

	// setup ID
	tenancyId, _, err := multitenancy.TenancyId(t, sctx, id, idIsDisplay...)
	if err != nil {
		c.SetMessage("failed to get ID")
		return c.SetError(err)
	}

	// create command
	handler := api_client.NewHandlerRequest(&multitenancy.WithPoolAndDb{POOL_ID: poolId, DBNAME: dbName})

	// prepare and exec handler
	op := api.OperationAsResource(t.TenancyResource, "pool-db", tenancyId, tenancy_api.ChangePoolOrDb())
	err = handler.Exec(t.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return c.SetError(err)
	}

	// done
	return nil

}
