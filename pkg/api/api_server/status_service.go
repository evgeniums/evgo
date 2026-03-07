package api_server

import (
	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/utils"
)

type CheckStatusEndpoint struct {
	ResourceEndpoint
}

func NewCheckStatusEndpoint() *CheckStatusEndpoint {
	ep := &CheckStatusEndpoint{}
	InitResourceEndpoint(ep, "check", "CheckStatus", access_control.Get)
	return ep
}

type StatusResponse struct {
	api.ResponseStub
	Status string `json:"status"`
}

func (e *CheckStatusEndpoint) HandleRequest(request Request) error {
	resp := &StatusResponse{Status: "running"}
	request.Response().SetMessage(resp)
	return nil
}

type CheckAccess struct{}

func (e *CheckAccess) HandleRequest(request Request) error {
	resp := &StatusResponse{Status: "success"}
	request.Response().SetMessage(resp)
	return nil
}

type CheckAccessEndpoint struct {
	EndpointBase
	CheckAccess
}

func NewCheckAccessEndpoint(operationName string, accessType ...access_control.AccessType) *CheckAccessEndpoint {
	ep := &CheckAccessEndpoint{}
	ep.Init(operationName, accessType...)
	return ep
}

type CheckAccessResourceEndpoint struct {
	ResourceEndpoint
	CheckAccess
}

func NewCheckAccessResourceEndpoint(resource string, operationName string,
	accessType ...access_control.AccessType) *CheckAccessResourceEndpoint {
	ep := &CheckAccessResourceEndpoint{}
	InitResourceEndpoint(ep, resource, operationName, accessType...)
	return ep
}

type EchoEndpoint struct {
	EndpointBase
}

func (e *EchoEndpoint) HandleRequest(request Request) error {
	content := request.GetRequestContent()
	request.Response().SetPayload(content)
	return nil
}

func (e *EchoEndpoint) IsRequestPayloadNeeded() bool {
	return true
}

func NewEchoEndpoint(opName ...string) *EchoEndpoint {
	ep := &EchoEndpoint{}
	ep.Init(utils.OptionalString("Echo", opName...), access_control.Post)
	return ep
}

type StatusService struct {
	ServiceBase
}

func NewStatusService(multitenancy ...bool) *StatusService {
	s := &StatusService{}

	s.Init("status", api.PackageName, multitenancy...)
	s.AddChildren(NewCheckStatusEndpoint(),
		NewCheckAccessResourceEndpoint("csrf", "CheckCsrf"),
		NewCheckAccessResourceEndpoint("logged", "CheckLogged"),
	)
	altSmsPath := NewCheckAccessResourceEndpoint("sms-alt", "CheckSmsAlt", access_control.Post)
	altSmsPath.SetTestOnly(true)
	s.AddChild(altSmsPath)

	sms := api.NewResource("sms")
	sms.AddOperation(NewEchoEndpoint("PutSms"))
	// TODO findout why this breaks Echo endpoint
	// altSmsMethod := NewCheckAccessEndpoint("CheckSmsPut", access_control.Put)
	// altSmsMethod.SetTestOnly(true)
	// sms.AddOperation(altSmsMethod)
	s.AddChild(sms)

	echo := api.NewResource("echo")
	echo.AddOperation(NewEchoEndpoint())
	s.AddChild(echo)

	return s
}
