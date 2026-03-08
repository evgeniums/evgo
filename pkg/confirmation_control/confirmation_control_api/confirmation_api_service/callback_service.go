package confirmation_api_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/confirmation_control"
	"github.com/evgeniums/evgo/pkg/confirmation_control/confirmation_control_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type CallbackEndpoint struct {
	service *ConfirmationCallbackService
	api_server.EndpointBase
}

func (e *CallbackEndpoint) Construct(service *ConfirmationCallbackService, op api.Operation) {
	e.service = service
	e.EndpointBase.Construct(op)
}

type ConfirmationCallbackService struct {
	api_server.ServiceBase
	CallbackResource            api.Resource
	ConfirmationCallbackHandler confirmation_control.ConfirmationCallbackHandler
}

func NewConfirmationCallbackService(confirmationCallbackHandler confirmation_control.ConfirmationCallbackHandler) *ConfirmationCallbackService {

	s := &ConfirmationCallbackService{ConfirmationCallbackHandler: confirmationCallbackHandler}

	s.Init(confirmation_control_api.ServiceName, confirmation_control.PackageName, true)
	s.CallbackResource = api.NewResource(confirmation_control_api.CallbackResource)
	s.AddChild(s.CallbackResource)
	s.CallbackResource.AddOperation(CallbackConfirmation(s))

	return s
}

type CallbackConfirmationEndpoint struct {
	CallbackEndpoint
}

func (e *CallbackConfirmationEndpoint) HandleRequest(sctx context.Context) error {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("ConfirmationCallbackService.CallbackConfirmation")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[confirmation_control_api.CallbackConfirmationCmd](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
	}

	request.SetLoggerField("confirmation_id", cmd.Id)

	// invoke callback
	resp := &confirmation_control_api.CallbackConfirmationResponse{}
	resp.Url, err = e.service.ConfirmationCallbackHandler.ConfirmationCallback(sctx, cmd.Id, &cmd.ConfirmationResult)
	if err != nil {
		c.SetMessage("failed to invoke callback")
		return c.SetError(err)
	}

	// set response
	request.Response().SetMessage(resp)

	// done
	return nil
}

func CallbackConfirmation(s *ConfirmationCallbackService) *CallbackConfirmationEndpoint {
	e := &CallbackConfirmationEndpoint{}
	e.Construct(s, confirmation_control_api.CallbackConfirmation())
	return e
}
