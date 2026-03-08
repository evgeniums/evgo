package pool_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
)

type FindService struct {
	result *pool_api.ServiceResponse
}

func (a *FindService) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("FindService.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, nil, a.result)
	c.SetError(err)
	return err
}

func (p *PoolClient) FindService(sctx context.Context, id string, idIsName ...bool) (pool.PoolService, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolClient.FindService")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	sId, service, err := p.serviceId(sctx, id, idIsName...)
	if err != nil {
		return nil, err
	}
	if service != nil {
		return service, nil
	}

	// prepare and exec handler
	handler := &FindService{
		result: &pool_api.ServiceResponse{},
	}
	op := api.NamedResourceOperation(p.ServiceResource, sId, pool_api.FindService())
	err = handler.Exec(p.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.result.PoolServiceBase, nil
}
