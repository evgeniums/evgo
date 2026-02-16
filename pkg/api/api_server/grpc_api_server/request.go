package grpc_api_server

import (
	"context"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/utils"
	"github.com/evgeniums/go-utils/pkg/validator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type CallContext struct {
	context.Context
}

type RequestMessage interface {
	ResourceIds() api.ResourceIds
	BinaryContent() []byte
	LogicMessage() interface{}
}

type RequestMessageBase struct {
	message interface{}
}

func NewRequestMessage() *RequestMessageBase {
	return &RequestMessageBase{}
}

func (m *RequestMessageBase) BinaryContent() []byte {
	return nil
}

func (m *RequestMessageBase) LogicMessage() any {
	return m.message
}

func (m *RequestMessageBase) ResourceIds() api.ResourceIds {
	return nil
}

type Request struct {
	api_server.RequestBase

	response *Response

	server *Server
	ctx    CallContext

	start time.Time

	clientIp          string
	forwardedOpSource string

	params map[string]any

	userAgent    string
	pseudoMethod string

	statusCode    codes.Code
	statusMessage string

	message RequestMessage
}

func HTTPToGRPC(httpCode int) codes.Code {
	switch httpCode {
	case http.StatusOK:
		return codes.OK
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusRequestTimeout:
		return codes.DeadlineExceeded
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	case http.StatusGatewayTimeout:
		return codes.DeadlineExceeded

	case http.StatusInternalServerError:
		return codes.Internal

	default:
		return codes.Unknown
	}
}
func (r *Request) Init(s *Server, ctx CallContext, fields ...logger.Fields) error {

	r.start = time.Now()
	r.server = s

	r.RequestBase.Init(s.App(), s.App().Logger(), s.App().Db(), fields...)
	r.RequestBase.SetErrorManager(s)

	r.params = make(map[string]any)

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// TODO log error
		return status.Error(codes.Unauthenticated, "metadata missing")
	}

	getHeaders := func(name string) []string {
		return md.Get(name)
	}

	getHeader := func(name string) string {
		h := getHeaders(name)
		if len(h) > 0 {
			return h[0]
		}
		return ""
	}

	p, ok := peer.FromContext(ctx)
	if ok {
		r.clientIp = p.Addr.String()
	}

	if userAgents := md.Get("user-agent"); len(userAgents) > 0 {
		r.userAgent = userAgents[0]
	}

	// TODO extract tenancy

	if s.propagateContextId {
		ctxId := getHeader(api.ForwardContext)
		if ctxId != "" {
			r.SetID(ctxId)
			r.SetLoggerField("context", ctxId)
		}
		forwardedOpSource := getHeader(api.ForwardOpSource)
		if forwardedOpSource != "" {
			r.forwardedOpSource = forwardedOpSource
			r.SetLoggerField("forwarded_op_source", forwardedOpSource)
		}
	}

	r.statusCode = codes.OK

	r.ctx = ctx
	r.response = &Response{}
	r.response.SetRequest(r)

	return nil
}

func (r *Request) Server() api_server.Server {
	return r.server
}

func (r *Request) GetParameter(key string) (any, bool) {
	value, ok := r.params[key]
	return value, ok
}

func (r *Request) SetParameter(key string, value any) {
	r.params[key] = value
}

func (r *Request) Response() api_server.Response {
	return r.response
}

func (r *Request) GetRequestMethod() string {
	return r.pseudoMethod
}

func (r *Request) GetRequestClientIp() string {
	return r.clientIp
}

func (r *Request) GetRequestUserAgent() string {
	return r.userAgent
}

func (r *Request) Close(successMessage ...string) {
	if r.GenericError() == nil {
		r.SetLoggerField("status", "success")
	} else {
		code, err := r.server.MakeResponseError(r.GenericError())
		if code < http.StatusInternalServerError {
			r.SetErrorAsWarn(true)
		}
		r.statusCode = HTTPToGRPC(code)
		r.SetLoggerField("status", err.Code())
	}

	r.RequestBase.Close("")
	// r.server.logRequest(r)
}

func (r *Request) GetRequestContent() []byte {
	if r.message == nil {
		return nil
	}
	return r.message.BinaryContent()
}

func AuthKey(key string, directKeyName ...bool) string {
	if utils.OptionalArg(false, directKeyName...) {
		return key
	}
	return utils.ConcatStrings("x-auth-", key)
}

func (r *Request) SetAuthParameter(authMethodProtocol string, key string, value string, directKeyName ...bool) {
	// handler := r.server.AuthParameterSetter(authMethodProtocol)
	// if handler != nil {
	// 	handler(r, key, value)
	// 	return
	// }
	// r.ginCtx.Header(AuthKey(key, directKeyName...), value)
}

func (r *Request) GetAuthParameter(authMethodProtocol string, key string, directKeyName ...bool) string {
	// handler := r.server.AuthParameterGetter(authMethodProtocol)
	// if handler != nil {
	// 	return handler(r, key)
	// }
	// return getHttpHeader(r.ginCtx, AuthKey(key, directKeyName...))

	return ""
}

func (r *Request) CheckRequestContent(smsMessage *string, skipSms *bool) error {
	return r.Endpoint().PrecheckRequestBeforeAuth(r, smsMessage, skipSms)
}

func (r *Request) ResourceIds() api.ResourceIds {

	if r.message == nil {
		return nil
	}

	return r.message.ResourceIds()
}

func (r *Request) GetRequestPath() string {
	return api_server.FullRequestServicePath(r)
}

func (r *Request) GetResourceId(resourceType string) api.ResourceId {

	if r.message == nil {
		return nil
	}

	return nil
}

func (r *Request) Validate(cmd interface{}) error {

	c := r.TraceInMethod("Request.Validate")
	defer r.TraceOutMethod()

	err := r.App().Validator().Validate(cmd)
	if err != nil {
		vErr, ok := err.(*validator.ValidationError)
		if ok {
			r.SetGenericError(vErr.GenericError(), true)
		}
		return c.SetError(err)
	}
	return nil
}

func (r *Request) ParseAndValidate(cmd interface{}) error {

	if cmd == nil {
		return nil
	}

	c := r.TraceInMethod("Request.ParseValidate")
	defer r.TraceOutMethod()

	err := r.Validate(cmd)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func (r *Request) FormData() map[string][]string {
	return nil
}

func (r *Request) FormFile() (*multipart.FileHeader, error) {
	return nil, nil
}

func (r *Request) GetTenancyId() string {
	// TODO Implement GetTenancy()
	return ""
}

func (r *Request) MessageFromRequest(builder func() interface{}) interface{} {
	if r.message == nil {
		return nil
	}
	return r.message.LogicMessage()
}
