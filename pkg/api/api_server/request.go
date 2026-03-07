package api_server

import (
	"mime/multipart"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context/default_op_context"
	"github.com/evgeniums/evgo/pkg/validator"
)

type MessageContent interface {
	BinaryContent() []byte
	SetBinaryContent([]byte)
	SetLogicMessage(interface{})
	LogicMessage() interface{}
	SetTransportMessage(interface{})
	TransportMessage() interface{}
}

type RequestMessage interface {
	MessageContent
}

type RequestMessageBase struct {
	logicMessage     interface{}
	transportMessage interface{}
	content          []byte
}

func NewRequestMessage() *RequestMessageBase {
	r := &RequestMessageBase{}
	return r
}

func (m *RequestMessageBase) BinaryContent() []byte {
	return m.content
}

func (m *RequestMessageBase) SetBinaryContent(content []byte) {
	m.content = content
}

func (m *RequestMessageBase) LogicMessage() any {
	return m.logicMessage
}

func (m *RequestMessageBase) TransportMessage() any {
	return m.transportMessage
}

func (m *RequestMessageBase) SetLogicMessage(msg interface{}) {
	m.logicMessage = msg
}

func (m *RequestMessageBase) SetTransportMessage(msg interface{}) {
	m.transportMessage = msg
}

// Interface of request to server API.
type Request interface {
	auth.AuthContext
	common.WithParameters

	Server() Server
	Response() Response
	Endpoint() Endpoint

	ParseAndValidate(cmd interface{}) error
	FormData() map[string][]string
	FormFile() (*multipart.FileHeader, error)

	SetMessage(msg RequestMessage)
	Message() RequestMessage
}

type RequestBase struct {
	auth.TenancyUserContext
	auth.SessionBase
	endpoint          Endpoint
	message           RequestMessage
	sessionParameters map[string]string
}

func (r *RequestBase) Init(app app_context.Context, log logger.Logger, db db.DB, fields ...logger.Fields) {
	baseCtx := default_op_context.NewContext()
	baseCtx.Init(app, log, db, fields...)
	r.Construct(baseCtx)
	r.message = &RequestMessageBase{}
	r.sessionParameters = make(map[string]string)
}

func (r *RequestBase) SetEndpoint(endpoint Endpoint) {
	r.endpoint = endpoint
}

func (r *RequestBase) Endpoint() Endpoint {
	return r.endpoint
}

func (r *RequestBase) SetMessage(msg RequestMessage) {
	r.message = msg
}

func (r *RequestBase) Message() RequestMessage {
	return r.message
}

func (r *RequestBase) LoadSessionParameters(parameters map[string]string) {
	r.sessionParameters = parameters
}

func (r *RequestBase) StoreSessionParameters() map[string]string {
	return r.sessionParameters
}

func (r *RequestBase) GetSessionParameter(key string) string {
	return r.sessionParameters[key]
}

func (r *RequestBase) SetSessionParameter(key string, value string) {
	r.sessionParameters[key] = value
}

func FullRequestPath(r Request) string {
	return r.Endpoint().Resource().BuildActualPath(r.ResourceIds())
}

func FullRequestServicePath(r Request) string {
	return r.Endpoint().Resource().BuildActualPath(r.ResourceIds(), true)
}

func ParseDbQuery(request Request, model interface{}, queryName string, cmd ...api.Query) (*db.Filter, error) {

	var q api.Query
	if len(cmd) == 0 {
		q = &api.DbQuery{}
	} else {
		q = cmd[0]
	}
	c := request.TraceInMethod("ParseDbQuery", logger.Fields{"query_name": queryName})
	defer request.TraceOutMethod()

	err := request.ParseAndValidate(q)
	if err != nil {
		c.SetMessage("failed to parse/verify query")
		return nil, c.SetError(err)
	}
	if q.Query() == "" {
		return nil, nil
	}

	filter, err := db.ParseQuery(request.Db(), q.Query(), model, queryName, db.EmptyFilterValidator(request.App().Validator()))
	if err != nil {
		vErr, ok := err.(*validator.ValidationError)
		if ok {
			request.SetGenericError(vErr.GenericError(), true)
		}
		c.SetMessage("failed to parse/validate db query")
		return nil, c.SetError(err)
	}

	return filter, nil
}

func UniqueFormData(request Request) map[string]string {

	form := request.FormData()
	res := make(map[string]string)
	for key, values := range form {
		if len(values) != 0 {
			res[key] = values[0]
		} else {
			res[key] = ""
		}
	}

	return res
}

func MessageFromRequest[T any](request Request, init ...func(*T)) (*T, error) {
	err := request.Endpoint().TransportRequestToLogic(request.Message())
	if err != nil {
		request.SetGenericErrorCode(generic_error.ErrorCodeFormat)
		return nil, err
	}
	msg, ok := request.Message().LogicMessage().(*T)
	if !ok {
		request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return nil, err
	}
	return msg, nil
}

func ParseValidateRequest[T any](request Request, init ...func(*T)) (*T, error) {
	msg, err := MessageFromRequest(request, init...)
	if err != nil {
		return nil, err
	}
	err = request.ParseAndValidate(msg)
	return msg, err
}
