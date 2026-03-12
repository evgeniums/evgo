package api_server

import (
	"context"

	"github.com/stoewer/go-strcase"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/utils"
)

type MessageBuilder func() interface{}
type MessageConverter func(msg MessageContent) error

type MessageHandlers interface {
	NewTransportRequest(ep Endpoint) interface{}
	TransportRequestToLogic(msg MessageContent) error
	LogicResponseToTransport(msg MessageContent) error
}

type NamedMessageHandlers interface {
	common.WithName
	MessageHandlers
}

type MessageHandlersConfig struct {
	RequestFromTransport    MessageConverter
	ResponseToTransport     MessageConverter
	TransportRequestBuilder MessageBuilder
}

type MessageHandlersBase interface {
	common.WithNameBase
	MessageHandlersConfig
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

type EndpointExtraHandler = func(sctx context.Context) (context.Context, error)

// Interface of API endpoint.
type Endpoint interface {
	api.Operation
	generic_error.ErrorsExtender
	MessageHandlers

	SetMessageHandlers(handlers MessageHandlers)

	NewRequestMessage() interface{}

	// Handle request to server API.
	HandleRequest(sctx context.Context) (context.Context, error)

	// Precheck request before some authorization methods
	PrecheckBeforeAuth(sctx context.Context, smsMessage *string, skipSms *bool) error

	IsRequestPayloadNeeded() bool

	SetRequestPreprocessor(handler EndpointExtraHandler)
	GetRequestPreprocessor() EndpointExtraHandler
	PreprocessBeforeAuth(sctx context.Context) (context.Context, error)

	SetRequestPostprocessor(handler EndpointExtraHandler)
	GetRequestPostprocessor() EndpointExtraHandler
	Postprocess(sctx context.Context) (context.Context, error)

	IsServerStreaming() bool
}

type EndpointHandler = func(sctx context.Context) error

// Base type for API endpoints.
type EndpointBase struct {
	api.Operation
	generic_error.ErrorsExtenderBase
	MessageHandlers

	preprocessRequest  EndpointExtraHandler
	postprocessRequest EndpointExtraHandler
}

func (e *EndpointBase) Construct(op api.Operation) {
	e.Operation = op
	e.MessageHandlers = &MessageHandlersConfig{}
}

func (e *EndpointBase) Init(operationName string, accessType ...access_control.AccessType) {
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

func (e *EndpointBase) PrecheckBeforeAuth(sctx context.Context, smsMessage *string, skipSms *bool) error {
	return nil
}

func (e *EndpointBase) HandleRequest(sctx context.Context) (context.Context, error) {
	return sctx, nil
}

func (e *EndpointBase) SetRequestPreprocessor(handler EndpointExtraHandler) {
	e.preprocessRequest = handler
}

func (e *EndpointBase) GetRequestPreprocessor() EndpointExtraHandler {
	return e.preprocessRequest
}

func (e *EndpointBase) PreprocessBeforeAuth(sctx context.Context) (context.Context, error) {
	if e.preprocessRequest != nil {
		return e.preprocessRequest(sctx)
	}
	return sctx, nil
}

func (e *EndpointBase) SetRequestPostprocessor(handler EndpointExtraHandler) {
	e.postprocessRequest = handler
}

func (e *EndpointBase) GetRequestPostprocessor() EndpointExtraHandler {
	return e.postprocessRequest
}

func (e *EndpointBase) Postprocess(sctx context.Context) (context.Context, error) {
	if e.postprocessRequest != nil {
		return e.postprocessRequest(sctx)
	}
	return sctx, nil
}

func (e *EndpointBase) IsServerStreaming() bool {
	return false
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

func (e *EndpointNoHandler) HandleRequest(sctx context.Context) (context.Context, error) {
	return sctx, nil
}
