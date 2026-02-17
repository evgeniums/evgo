package pool_client

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_client"
	"github.com/evgeniums/go-utils/pkg/db"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/pool"
	"github.com/evgeniums/go-utils/pkg/pool/pool_api"
)

type ListPools struct {
	cmd    api.Query
	result *pool_api.ListPoolsResponse
}

func (a *ListPools) Exec(client api_client.Client, ctx op_context.Context, operation api.Operation) error {

	c := ctx.TraceInMethod("ListPools.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, a.cmd, a.result)
	c.SetError(err)
	return err
}

func (p *PoolClient) GetPools(ctx op_context.Context, filter *db.Filter) ([]*pool.PoolBase, int64, error) {

	// setup
	var err error
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
	err = handler.Exec(p.Client(), ctx, p.list_pools)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, 0, err
	}

	// done
	return handler.result.Items, handler.result.Count, nil
}
