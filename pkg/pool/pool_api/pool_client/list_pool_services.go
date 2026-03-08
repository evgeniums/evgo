package pool_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type ListPoolServices struct {
	result *pool_api.ListServicePoolsResponse
}

func (a *ListPoolServices) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("ListPoolServices.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, nil, a.result)
	c.SetError(err)
	return err
}

func (p *PoolClient) GetPoolBindings(sctx context.Context, id string, idIsName ...bool) ([]*pool.PoolServiceBinding, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolClient.GetServiceBindings")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// adjust pool ID
	pId, _, err := p.poolId(sctx, id, idIsName...)
	if err != nil {
		return nil, err
	}

	// prepare and exec handler
	handler := &ListPoolServices{
		result: &pool_api.ListServicePoolsResponse{},
	}
	resource := p.resourceForPoolServices(pId)
	op := pool_api.ListPoolServices()
	resource.AddOperation(op)
	err = handler.Exec(p.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.result.Items, nil
}
