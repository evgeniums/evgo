package pool_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type RemoveServiceFromAllPools struct{}

func (a *RemoveServiceFromAllPools) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("RemoveServiceFromAllPools.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, nil, nil)
	c.SetError(err)
	return err
}

func (p *PoolClient) RemoveServiceFromAllPools(sctx context.Context, id string, idIsName ...bool) error {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolClient.RemoveServiceFromAllPools")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// adjust service ID
	sId, _, err := p.serviceId(sctx, id, idIsName...)
	if err != nil {
		return err
	}

	// prepare and exec handler
	handler := &RemoveServiceFromAllPools{}
	resource := p.resourceForServicePools(sId)
	op := pool_api.RemoveServiceFromAllPools()
	resource.AddOperation(op)
	err = handler.Exec(p.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return err
	}

	// done
	return nil
}
