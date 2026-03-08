package pool_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type AddPool struct {
	cmd    pool.Pool
	result *pool_api.PoolResponse
}

func (a *AddPool) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("AddPool.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, a.cmd, a.result)
	c.SetError(err)
	return err
}

func (p *PoolClient) AddPool(sctx context.Context, pool pool.Pool) (pool.Pool, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolClient.AddPool")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// prepare and exec handler
	handler := &AddPool{
		cmd:    pool,
		result: &pool_api.PoolResponse{},
	}
	err = handler.Exec(p.Client(), sctx, p.add_pool)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.result.PoolBase, nil
}
