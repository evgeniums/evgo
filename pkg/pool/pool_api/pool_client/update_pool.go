package pool_client

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/pool_api"
	"github.com/evgeniums/evgo/pkg/utils"
)

type UpdatePool struct {
	cmd    *api.UpdateCmd
	result *pool_api.PoolResponse
}

func (a *UpdatePool) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("UpdatePool.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, a.cmd, a.result)
	c.SetError(err)
	return err
}

func (p *PoolClient) UpdatePool(sctx context.Context, id string, fields db.Fields, idIsName ...bool) (pool.Pool, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolClient.UpdatePool")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// adjust id
	pId, po, err := p.poolId(sctx, id, idIsName...)
	if err != nil {
		return nil, err
	}
	if utils.OptionalArg(false, idIsName...) && po == nil {
		// pool not found by name
		ctx.SetGenericErrorCode(pool.ErrorCodePoolNotFound)
		return nil, errors.New("pool not found by name")
	}

	// prepare and exec handler
	handler := &UpdatePool{
		cmd:    &api.UpdateCmd{},
		result: &pool_api.PoolResponse{},
	}
	handler.cmd.Fields = fields
	op := api.NamedResourceOperation(p.PoolResource, pId, pool_api.UpdatePool())
	err = handler.Exec(p.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.result.PoolBase, nil
}
