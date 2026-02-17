package pool_client

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_client"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/pool"
	"github.com/evgeniums/go-utils/pkg/pool/pool_api"
)

type FindPool struct {
	result *pool_api.PoolResponse
}

func (a *FindPool) Exec(client api_client.Client, ctx op_context.Context, operation api.Operation) error {

	c := ctx.TraceInMethod("FindPool.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, nil, a.result)
	c.SetError(err)
	return err
}

func (p *PoolClient) FindPool(ctx op_context.Context, id string, idIsName ...bool) (pool.Pool, error) {

	// setup
	var err error
	c := ctx.TraceInMethod("PoolClient.FindPool")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	pId, pool, err := p.poolId(ctx, id, idIsName...)
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
	err = handler.Exec(p.Client(), ctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.result.PoolBase, nil
}
