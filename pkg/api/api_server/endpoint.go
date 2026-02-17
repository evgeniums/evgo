package api_server

import (
	"github.com/evgeniums/go-utils/pkg/access_control"
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/utils"
)

// Interface of API endpoint.
type Endpoint interface {
	api.Operation
	generic_error.ErrorsExtender

	// Handle request to server API.
	HandleRequest(request Request) error

	// Precheck request before some authorization methods
	PrecheckRequestBeforeAuth(request Request, smsMessage *string, skipSms *bool) error

	SetTransportToLogicMessageMapper(mapper func(interface{}) RequestMessage)

	SetLogicToTransportMessageMapper(mapper func(interface{}) interface{})

	SetTransportMessageBuilder(builder func() interface{})

	NewTransportMessage() interface{}

	TransportMessageToLogic(msg interface{}) RequestMessage

	LogicMessageToTransport(msg interface{}) interface{}
}

type EndpointHandler = func(request Request)

// Base type for API endpoints.
type EndpointBase struct {
	api.Operation
	generic_error.ErrorsExtenderBase

	newTransportMessage func() interface{}

	transportToLogic func(interface{}) RequestMessage
	logicToTransport func(interface{}) interface{}
}

func (e *EndpointBase) SetTransportToLogicMessageMapper(mapper func(interface{}) RequestMessage) {
	e.transportToLogic = mapper
}

func (e *EndpointBase) SetLogicToTransportMessageMapper(mapper func(interface{}) interface{}) {
	e.logicToTransport = mapper
}

func (e *EndpointBase) SetTransportMessageBuilder(builder func() interface{}) {
	e.newTransportMessage = builder
}

func (e *EndpointBase) NewTransportMessage() interface{} {
	if e.newTransportMessage == nil {
		return nil
	}
	return e.newTransportMessage
}

func (e *EndpointBase) TransportMessageToLogic(msg interface{}) RequestMessage {
	if e.transportToLogic == nil {
		return &RequestMessageBase{message: msg}
	}
	return e.transportToLogic(msg)
}

func (e *EndpointBase) LogicMessageToTransport(msg interface{}) interface{} {
	if e.logicToTransport == nil {
		return msg
	}
	return e.logicToTransport(msg)
}

func (e *EndpointBase) Construct(op api.Operation) {
	e.Operation = op
}

func (e *EndpointBase) Init(operationName string, accessType ...access_control.AccessType) {
	e.Construct(api.NewOperation(operationName, utils.OptionalArg(access_control.Get, accessType...)))
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
