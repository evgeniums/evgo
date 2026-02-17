package confirmation_api_client

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_client"
	"github.com/evgeniums/go-utils/pkg/confirmation_control"
	"github.com/evgeniums/go-utils/pkg/confirmation_control/confirmation_control_api"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
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

func (cl *ConfirmationCallbackClient) ConfirmationCallback(ctx multitenancy.TenancyContext, operationId string, result *confirmation_control.ConfirmationResult) (string, error) {

	// setup
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
	err = handler.Exec(cl.ApiClient(), ctx, cl.callback_confirmation)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return "", err
	}

	// done
	return handler.Result.Url, nil
}
