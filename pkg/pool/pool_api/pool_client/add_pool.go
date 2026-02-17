package pool_client

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_client"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/pool"
	"github.com/evgeniums/go-utils/pkg/pool/pool_api"
)

type AddPool struct {
	cmd    pool.Pool
	result *pool_api.PoolResponse
}

func (a *AddPool) Exec(client api_client.Client, ctx op_context.Context, operation api.Operation) error {

	c := ctx.TraceInMethod("AddPool.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, a.cmd, a.result)
	c.SetError(err)
	return err
}

func (p *PoolClient) AddPool(ctx op_context.Context, pool pool.Pool) (pool.Pool, error) {

	// setup
	var err error
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
	err = handler.Exec(p.Client(), ctx, p.add_pool)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.result.PoolBase, nil
}
