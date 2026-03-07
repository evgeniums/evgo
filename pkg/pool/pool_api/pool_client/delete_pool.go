package pool_client

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
	"github.com/evgeniums/evgo/pkg/utils"
)

type DeletePool struct{}

func (a *DeletePool) Exec(client api_client.Client, ctx op_context.Context, operation api.Operation) error {

	c := ctx.TraceInMethod("DeletePool.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, nil, nil)
	c.SetError(err)
	return err
}

func (p *PoolClient) DeletePool(ctx op_context.Context, id string, idIsName ...bool) error {

	// setup
	var err error
	c := ctx.TraceInMethod("PoolClient.DeletePool")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	pId, pool, err := p.poolId(ctx, id, idIsName...)
	if err != nil {
		return err
	}
	if utils.OptionalArg(false, idIsName...) && pool == nil {
		// pool not found by name
		return nil
	}

	// prepare and exec handler
	handler := &DeletePool{}
	op := api.NamedResourceOperation(p.PoolResource, pId, pool_api.DeletePool())
	err = handler.Exec(p.Client(), ctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return err
	}

	// done
	return nil
}
