package pool_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type AddServiceToPool struct {
	pool.PoolServiceAssociationCmd
}

func (a *AddServiceToPool) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("AddServiceToPool.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, a, nil)
	c.SetError(err)
	return err
}

func (p *PoolClient) AddServiceToPool(sctx context.Context, poolId string, serviceId string, role string, idIsName ...bool) error {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolClient.AddServiceToPool")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// adjust ids
	pId, _, err := p.poolId(sctx, poolId, idIsName...)
	if err != nil {
		return err
	}
	sId, _, err := p.serviceId(sctx, serviceId, idIsName...)
	if err != nil {
		return err
	}

	// prepare and exec handler
	handler := &AddServiceToPool{}
	handler.ROLE = role
	handler.SERVICE_ID = sId
	resource := p.resourceForPoolServices(pId)
	op := pool_api.AddServiceToPool()
	resource.AddOperation(op)
	err = handler.Exec(p.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return err
	}

	// done
	return nil
}
