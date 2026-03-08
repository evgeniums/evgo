package pool_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type RemoveServiceFromPool struct{}

func (a *RemoveServiceFromPool) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("RemoveServiceFromPool.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, nil, nil)
	c.SetError(err)
	return err
}

func (p *PoolClient) RemoveServiceFromPool(sctx context.Context, poolId string, role string, idIsName ...bool) error {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolClient.RemoveServiceFromPool")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// adjust pool ID
	pId, _, err := p.poolId(sctx, poolId, idIsName...)
	if err != nil {
		return err
	}

	// prepare and exec handler
	handler := &RemoveServiceFromPool{}
	poolServiceResource := p.resourceForPoolServices(pId)
	roleResource := api.NamedResource("role")
	poolServiceResource.AddChild(roleResource)
	roleResource.SetId(role)
	op := pool_api.RemoveServiceFromPool()
	roleResource.AddOperation(op)
	err = handler.Exec(p.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return err
	}

	// done
	return nil
}
