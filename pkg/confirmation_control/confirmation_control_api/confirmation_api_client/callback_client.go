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

type ConfirmationCallbackClient struct {
	api_client.ServiceClient

	CallbackResource      api.Resource
	callback_confirmation api.Operation
}

func NewConfirmationCallbackClient(client api_client.Client) *ConfirmationCallbackClient {

	c := &ConfirmationCallbackClient{}

	c.Init(client, confirmation_control_api.ServiceName)

	c.CallbackResource = api.NewResource(confirmation_control_api.CallbackResource)
	c.AddChild(c.CallbackResource)

	c.callback_confirmation = confirmation_control_api.CallbackConfirmation()
	c.CallbackResource.AddOperation(c.callback_confirmation)

	api.NewTenancyResource().AddChild(c)

	return c
}

func (cl *ConfirmationCallbackClient) ConfirmationCallback(sctx context.Context, operationId string, result *confirmation_control.ConfirmationResult) (string, error) {

	// setup
	ctx := op_context.OpContext[multitenancy.TenancyContext](sctx)
	c := ctx.TraceInMethod("ConfirmationCallbackClient.ConfirmationCallback")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// prepare and exec handler
	cmd := &confirmation_control_api.CallbackConfirmationCmd{
		Id:                 operationId,
		ConfirmationResult: *result,
	}
	handler := api_client.NewHandlerInTenancy(cmd, &confirmation_control_api.CallbackConfirmationResponse{})
	err = handler.Exec(cl.ApiClient(), sctx, cl.callback_confirmation)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return "", err
	}

	// done
	return handler.Result.Url, nil
}
