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

type UpdateService struct {
	cmd    *api.UpdateCmd
	result *pool_api.ServiceResponse
}

func (a *UpdateService) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("UpdateService.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, a.cmd, a.result)
	c.SetError(err)
	return err
}

func (p *PoolClient) UpdateService(sctx context.Context, id string, fields db.Fields, idIsName ...bool) (pool.PoolService, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolClient.UpdateService")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// adjust id
	sId, service, err := p.serviceId(sctx, id, idIsName...)
	if err != nil {
		return nil, err
	}
	if utils.OptionalArg(false, idIsName...) && service == nil {
		// service not found by name
		ctx.SetGenericErrorCode(pool.ErrorCodeServiceNotFound)
		return nil, errors.New("service not found by name")
	}

	// prepare and exec handler
	handler := &UpdateService{
		cmd:    &api.UpdateCmd{},
		result: &pool_api.ServiceResponse{},
	}
	handler.cmd.Fields = fields
	op := api.NamedResourceOperation(p.ServiceResource, sId, pool_api.UpdateService())
	err = handler.Exec(p.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.result.PoolServiceBase, nil
}
