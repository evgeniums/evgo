package confirmation_api_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/confirmation_control"
	"github.com/evgeniums/evgo/pkg/confirmation_control/confirmation_control_api"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type ConfirmationExternalClient struct {
	api_client.ServiceClient

	OperationResource api.Resource
}

func NewConfirmationExternalClient(client api_client.Client) *ConfirmationExternalClient {

	c := &ConfirmationExternalClient{}

	c.Init(client, confirmation_control_api.ServiceName)

	c.OperationResource = api.NamedResource(confirmation_control_api.OperationResource)
	c.AddChild(c.OperationResource.Parent())

	return c
}

func (cl *ConfirmationExternalClient) CheckConfirmation(sctx context.Context, operationId string, result *confirmation_control.ConfirmationResult) (string, error) {

	// setup
	ctx := op_context.OpContext[multitenancy.TenancyContext](sctx)
	c := ctx.TraceInMethod("ConfirmationExternalClient.CheckConfirmation")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// prepare and exec handler
	cmd := result
	handler := api_client.NewHandlerInTenancy(cmd, &confirmation_control_api.CheckConfirmationResponse{})
	op := api.NamedResourceOperation(cl.OperationResource, operationId, confirmation_control_api.CheckConfirmation())
	err = handler.Exec(cl.ApiClient(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return "", err
	}

	// done
	return handler.Result.RedirectUrl, nil
}

func (cl *ConfirmationExternalClient) PrepareCheckConfirmation(sctx context.Context, operationId string) (string, error) {

	// setup
	ctx := op_context.OpContext[multitenancy.TenancyContext](sctx)
	c := ctx.TraceInMethod("ConfirmationExternalClient.PrepareCheckConfirmation")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// prepare and exec handler
	handler := api_client.NewHandlerInTenancyResult(&confirmation_control_api.PrepareCheckConfirmationResponse{})
	op := api.NamedResourceOperation(cl.OperationResource, operationId, confirmation_control_api.PrepareCheckConfirmation())
	err = handler.Exec(cl.ApiClient(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return "", err
	}

	// done
	return handler.Result.FailedUrl, nil
}
