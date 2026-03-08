package pool_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type FindPool struct {
	result *pool_api.PoolResponse
}

func (a *FindPool) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("FindPool.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, nil, a.result)
	c.SetError(err)
	return err
}

func (p *PoolClient) FindPool(sctx context.Context, id string, idIsName ...bool) (pool.Pool, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolClient.FindPool")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	pId, pool, err := p.poolId(sctx, id, idIsName...)
	if err != nil {
		return nil, err
	}
	if pool != nil {
		return pool, nil
	}

	// prepare and exec handler
	handler := &FindPool{
		result: &pool_api.PoolResponse{},
	}
	op := api.NamedResourceOperation(p.PoolResource, pId, pool_api.FindPool())
	err = handler.Exec(p.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.result.PoolBase, nil
}
