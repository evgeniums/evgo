package pool_client

import (
	"errors"

	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_client"
	"github.com/evgeniums/go-utils/pkg/db"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/pool"
	"github.com/evgeniums/go-utils/pkg/pool/pool_api"
	"github.com/evgeniums/go-utils/pkg/utils"
)

type UpdatePool struct {
	cmd    *api.UpdateCmd
	result *pool_api.PoolResponse
}

func (a *UpdatePool) Exec(client api_client.Client, ctx op_context.Context, operation api.Operation) error {

	c := ctx.TraceInMethod("UpdatePool.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, a.cmd, a.result)
	c.SetError(err)
	return err
}

func (p *PoolClient) UpdatePool(ctx op_context.Context, id string, fields db.Fields, idIsName ...bool) (pool.Pool, error) {

	// setup
	var err error
	c := ctx.TraceInMethod("PoolClient.UpdatePool")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// adjust id
	pId, po, err := p.poolId(ctx, id, idIsName...)
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
	err = handler.Exec(p.Client(), ctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.result.PoolBase, nil
}
