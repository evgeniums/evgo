package api_server

import (
	"github.com/evgeniums/go-utils/pkg/access_control"
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/utils"
)

type EndpointMessageHandlers interface {
	SetTransportToLogicMessageMapper(mapper func(interface{}) RequestMessage)
	GetTransportToLogicMessageMapper() func(interface{}) RequestMessage

	SetLogicToTransportMessageMapper(mapper func(interface{}) interface{})
	GetLogicToTransportMessageMapper() func(interface{}) interface{}

	SetTransportMessageBuilder(builder func() interface{})
	GetTransportMessageBuilder() func() interface{}

	NewTransportMessage() interface{}

	TransportMessageToLogic(msg interface{}) RequestMessage

	LogicMessageToTransport(msg interface{}) interface{}
}

type EndpointMessageHandlersBase struct {
	newTransportMessage func() interface{}

	transportToLogic func(interface{}) RequestMessage
	logicToTransport func(interface{}) interface{}
}

func (e *EndpointMessageHandlersBase) SetTransportToLogicMessageMapper(mapper func(interface{}) RequestMessage) {
	e.transportToLogic = mapper
}

func (e *EndpointMessageHandlersBase) GetTransportToLogicMessageMapper() func(interface{}) RequestMessage {
	return e.transportToLogic
}

func (e *EndpointMessageHandlersBase) SetLogicToTransportMessageMapper(mapper func(interface{}) interface{}) {
	e.logicToTransport = mapper
}

func (e *EndpointMessageHandlersBase) GetLogicToTransportMessageMapper() func(interface{}) interface{} {
	return e.logicToTransport
}

func (e *EndpointMessageHandlersBase) SetTransportMessageBuilder(builder func() interface{}) {
	e.newTransportMessage = builder
}

func (e *EndpointMessageHandlersBase) NewTransportMessage() interface{} {
	if e.newTransportMessage == nil {
		return nil
	}
	return e.newTransportMessage()
}

func (e *EndpointMessageHandlersBase) GetTransportMessageBuilder() func() interface{} {
	return e.newTransportMessage
}

func (e *EndpointMessageHandlersBase) TransportMessageToLogic(msg interface{}) RequestMessage {
	if e.transportToLogic == nil {
		return &RequestMessageBase{message: msg}
	}
	return e.transportToLogic(msg)
}

func (e *EndpointMessageHandlersBase) LogicMessageToTransport(msg interface{}) interface{} {
	if e.logicToTransport == nil {
		return msg
	}
	return e.logicToTransport(msg)
}

// Interface of API endpoint.
type Endpoint interface {
	api.Operation
	generic_error.ErrorsExtender
	EndpointMessageHandlers

	SetMessageHandlers(handlers EndpointMessageHandlers)

	// Handle request to server API.
	HandleRequest(request Request) error

	// Precheck request before some authorization methods
	PrecheckRequestBeforeAuth(request Request, smsMessage *string, skipSms *bool) error
}

type EndpointHandler = func(request Request)

// Base type for API endpoints.
type EndpointBase struct {
	api.Operation
	generic_error.ErrorsExtenderBase
	EndpointMessageHandlers
}

func (e *EndpointBase) Construct(op api.Operation) {
	e.Operation = op
}

func (e *EndpointBase) Init(operationName string, accessType ...access_control.AccessType) {
	e.EndpointMessageHandlers = &EndpointMessageHandlersBase{}
	e.Construct(api.NewOperation(operationName, utils.OptionalArg(access_control.Get, accessType...)))
}

func (e *EndpointBase) SetMessageHandlers(handlers EndpointMessageHandlers) {
	e.EndpointMessageHandlers = handlers
}

func (e *EndpointBase) PrecheckRequestBeforeAuth(request Request, smsMessage *string, skipSms *bool) error {
	return nil
}

type ResourceEndpointI interface {
	api.Resource
	Endpoint
	init(resourceType string, operationName string, accessType ...access_control.AccessType)
	construct(resourceType string, op api.Operation)
}

type ResourceEndpoint struct {
	api.ResourceBase
	EndpointBase
}

func (e *ResourceEndpoint) init(resourceType string, operationName string, accessType ...access_control.AccessType) {
	e.EndpointBase.Init(operationName, accessType...)
	e.ResourceBase.Init(resourceType)
}

func (e *ResourceEndpoint) construct(resourceType string, op api.Operation) {
	e.ResourceBase.Init(resourceType)
	e.EndpointBase.Construct(op)
}

func ConstructResourceEndpoint(ep ResourceEndpointI, resourceType string, op api.Operation) {
	ep.construct(resourceType, op)
	ep.AddOperation(ep)
}

func InitResourceEndpoint(ep ResourceEndpointI, resourceType string, operationName string, accessType ...access_control.AccessType) {
	ep.init(resourceType, operationName, accessType...)
	ep.AddOperation(ep)
}

// Base type for API endpoints with empty handlers.
type EndpointNoHandler struct{}

func (e *EndpointNoHandler) HandleRequest(request Request) error {
	return nil
}
