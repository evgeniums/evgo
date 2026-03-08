package pool_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type ListPools struct {
	cmd    api.Query
	result *pool_api.ListPoolsResponse
}

func (a *ListPools) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("ListPools.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, a.cmd, a.result)
	c.SetError(err)
	return err
}

func (p *PoolClient) GetPools(sctx context.Context, filter *db.Filter) ([]*pool.PoolBase, int64, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolClient.GetPools")
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
	handler := &ListPools{
		cmd:    cmd,
		result: &pool_api.ListPoolsResponse{},
	}
	err = handler.Exec(p.Client(), sctx, p.list_pools)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, 0, err
	}

	// done
	return handler.result.Items, handler.result.Count, nil
}
