package api_server

import (
	"github.com/stoewer/go-strcase"

	"github.com/evgeniums/go-utils/pkg/access_control"
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/utils"
)

type MessageBuilder func() interface{}
type MessageConverter func(msg MessageContent) error

type MessageHandlers interface {
	NewTransportRequest(ep Endpoint) interface{}
	TransportRequestToLogic(msg MessageContent) error
	LogicResponseToTransport(msg MessageContent) error
}

type MessageHandlersConfig struct {
	RequestFromTransport    MessageConverter
	ResponseToTransport     MessageConverter
	TransportRequestBuilder MessageBuilder
}

func (m *MessageHandlersConfig) NewTransportRequest(ep Endpoint) interface{} {
	if m.TransportRequestBuilder != nil {
		return m.TransportRequestBuilder()
	}
	return ep.NewRequestMessage()
}

func (m *MessageHandlersConfig) TransportRequestToLogic(msg MessageContent) error {
	if m.RequestFromTransport != nil {
		return m.RequestFromTransport(msg)
	}
	msg.SetLogicMessage(msg.TransportMessage())
	return nil
}

func (m *MessageHandlersConfig) LogicResponseToTransport(msg MessageContent) error {
	if m.ResponseToTransport != nil {
		return m.ResponseToTransport(msg)
	}
	msg.SetTransportMessage(msg.LogicMessage())
	return nil
}

// Interface of API endpoint.
type Endpoint interface {
	api.Operation
	generic_error.ErrorsExtender
	MessageHandlers

	SetMessageHandlers(handlers MessageHandlers)

	NewRequestMessage() interface{}

	// Handle request to server API.
	HandleRequest(request Request) error

	// Precheck request before some authorization methods
	PrecheckRequestBeforeAuth(request Request, smsMessage *string, skipSms *bool) error

	IsRequestPayloadNeeded() bool
}

type EndpointHandler = func(request Request)

// Base type for API endpoints.
type EndpointBase struct {
	api.Operation
	generic_error.ErrorsExtenderBase
	MessageHandlers
}

func (e *EndpointBase) Construct(op api.Operation) {
	e.Operation = op
}

func (e *EndpointBase) Init(operationName string, accessType ...access_control.AccessType) {
	e.MessageHandlers = &MessageHandlersConfig{}
	e.Construct(api.NewOperation(operationName, utils.OptionalArg(access_control.Get, accessType...)))
}

func (e *EndpointBase) SetMessageHandlers(handlers MessageHandlers) {
	e.MessageHandlers = handlers
}

func (e *EndpointBase) IsRequestPayloadNeeded() bool {
	return false
}

func (e *EndpointBase) NewRequestMessage() interface{} {
	return nil
}

func (e *EndpointBase) PrecheckRequestBeforeAuth(request Request, smsMessage *string, skipSms *bool) error {
	return nil
}

func (e *EndpointBase) HandleRequest(request Request) error {
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

func InitKebabEndpoint(ep ResourceEndpointI, operationName string, accessType ...access_control.AccessType) {
	resourceType := strcase.KebabCase(operationName)
	InitResourceEndpoint(ep, resourceType, operationName, accessType...)
}

// Base type for API endpoints with empty handlers.
type EndpointNoHandler struct{}

func (e *EndpointNoHandler) HandleRequest(request Request) error {
	return nil
}
