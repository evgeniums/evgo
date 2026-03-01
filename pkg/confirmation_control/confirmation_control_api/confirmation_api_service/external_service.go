package confirmation_api_service

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/confirmation_control"
	"github.com/evgeniums/go-utils/pkg/confirmation_control/confirmation_control_api"
)

type ExternalEndpoint struct {
	service *ConfirmationExternalService
	api_server.EndpointBase
}

func (e *ExternalEndpoint) Construct(service *ConfirmationExternalService, op api.Operation) {
	e.service = service
	e.EndpointBase.Construct(op)
}

type ConfirmationExternalService struct {
	api_server.ServiceBase
	ConfirmationCallbackHandler confirmation_control.ConfirmationCallbackHandler

	OperationResource api.Resource
	FailedResource    api.Resource

	CheckCode bool
}

func NewConfirmationExternalService(confirmationCallbackHandler confirmation_control.ConfirmationCallbackHandler, checkCode bool) *ConfirmationExternalService {

	s := &ConfirmationExternalService{CheckCode: checkCode}
	s.ConfirmationCallbackHandler = confirmationCallbackHandler

	s.Init(confirmation_control_api.ServiceName, confirmation_control.PackageName, true)
	s.OperationResource = api.NamedResource(confirmation_control_api.OperationResource)
	s.AddChild(s.OperationResource.Parent())
	s.OperationResource.AddOperations(CheckConfirmation(s), PrepareCheckConfirmation(s))

	s.FailedResource = api.NewResource(confirmation_control_api.FailedResource)
	s.OperationResource.AddChild(s.FailedResource)
	s.FailedResource.AddOperation(FailedConfirmation(s))

	return s
}

type CheckConfirmationEndpoint struct {
	ExternalEndpoint
}

func (e *CheckConfirmationEndpoint) PrecheckRequestBeforeAuth(request api_server.Request, smsMessage *string, skipSms *bool) error {

	// setup
	c := request.TraceInMethod("ConfirmationExternalService.PrecheckRequestBeforeAuth")
	defer request.TraceOutMethod()

	// get token from cache
	cacheToken, err := confirmation_control_api.GetTokenFromCache(request)
	if err != nil {
		return c.SetError(err)
	}

	// get SMS message from parameters
	if len(cacheToken.Parameters) != 0 {
		smsMsgIf, ok := cacheToken.Parameters["sms"]
		if ok {
			smsMsg, ok := smsMsgIf.(string)
			if ok {
				*smsMessage = smsMsg
			}
		}
	}

	// done
	return nil
}

func (e *CheckConfirmationEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	var err error
	c := request.TraceInMethod("ConfirmationExternalService.CheckConfirmation")
	defer request.TraceOutMethod()

	confirmationId := request.GetResourceId(confirmation_control_api.OperationResource)
	request.SetLoggerField("confirmation_id", confirmationId)

	// fill code or status
	var result = &confirmation_control.ConfirmationResult{}
	if e.service.CheckCode {
		// parse command
		cmd, err := api_server.ParseValidateRequest[confirmation_control.ConfirmationResult](request)
		if err != nil {
			c.SetLoggerField("request_content", string(request.GetRequestContent()))
			c.SetMessage("failed to parse/validate command")
			return c.SetError(err)
		}
		result.Code = cmd.Code
	} else {
		result.Status = confirmation_control.StatusSuccess
	}

	// invoke callback
	resp := &confirmation_control_api.CheckConfirmationResponse{}
	resp.RedirectUrl, err = e.service.ConfirmationCallbackHandler.ConfirmationCallback(request, confirmationId.Value(), result)
	request.SetLoggerField("redirect_url", resp.RedirectUrl)
	if err != nil {
		c.SetMessage("failed to invoke callback")
		return c.SetError(err)
	}

	// set response
	request.Response().SetMessage(resp)

	// delete token from cache
	confirmation_control_api.DeleteTokenFromCache(request)

	// done
	return nil
}

func CheckConfirmation(s *ConfirmationExternalService) *CheckConfirmationEndpoint {
	e := &CheckConfirmationEndpoint{}
	e.Construct(s, confirmation_control_api.CheckConfirmation())
	return e
}

type PrepareCheckConfirmationEndpoint struct {
	ExternalEndpoint
}

func (e *PrepareCheckConfirmationEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("ConfirmationExternalService.PrepareCheckConfirmation")
	defer request.TraceOutMethod()

	// get token from cache
	cacheToken, err := confirmation_control_api.GetTokenFromCache(request)
	if err != nil {
		return c.SetError(err)
	}

	// set response
	resp := &confirmation_control_api.PrepareCheckConfirmationResponse{}
	resp.FailedUrl = cacheToken.FailedUrl
	resp.CodeInBody = e.service.CheckCode
	resp.Parameters = cacheToken.Parameters
	delete(resp.Parameters, "sms")
	request.SetLoggerField("failed_url", resp.FailedUrl)
	request.SetLoggerField("code_in_body", resp.CodeInBody)
	request.SetLoggerField("parameters", resp.Parameters)
	request.Response().SetMessage(resp)

	// done
	return nil
}

func PrepareCheckConfirmation(s *ConfirmationExternalService) *PrepareCheckConfirmationEndpoint {
	e := &PrepareCheckConfirmationEndpoint{}
	e.Construct(s, confirmation_control_api.PrepareCheckConfirmation())
	return e
}

type FailedConfirmationEndpoint struct {
	ExternalEndpoint
}

func (e *FailedConfirmationEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	var err error
	c := request.TraceInMethod("ConfirmationExternalService.FailedConfirmation")
	defer request.TraceOutMethod()

	confirmationId := request.GetResourceId(confirmation_control_api.OperationResource)
	request.SetLoggerField("confirmation_id", confirmationId)

	// parse command
	result, err := api_server.ParseValidateRequest[confirmation_control.ConfirmationResult](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return c.SetError(err)
	}
	// fill failed status
	if result.Status != confirmation_control.StatusCancelled {
		result.Status = confirmation_control.StatusFailed
	}

	// invoke callback
	resp := &confirmation_control_api.CheckConfirmationResponse{}
	resp.RedirectUrl, err = e.service.ConfirmationCallbackHandler.ConfirmationCallback(request, confirmationId.Value(), result)
	request.SetLoggerField("redirect_url", confirmationId)
	if err != nil {
		c.SetMessage("failed to invoke callback")
		return c.SetError(err)
	}

	// set response
	request.Response().SetMessage(resp)

	// done
	return nil
}

func FailedConfirmation(s *ConfirmationExternalService) *FailedConfirmationEndpoint {
	e := &FailedConfirmationEndpoint{}
	e.Construct(s, confirmation_control_api.FailedConfirmation())
	return e
}
