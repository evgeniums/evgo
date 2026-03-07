package grpc_api_server

import (
	"context"
	"errors"
	"mime/multipart"
	"strings"
	"time"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/op_context/default_op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/mem"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type CallContext = context.Context

type Request struct {
	api_server.RequestBase

	response *Response

	server *Server
	ctx    context.Context

	start time.Time

	clientIp          string
	forwardedOpSource string

	params map[string]any

	userAgent    string
	pseudoMethod string

	statusCode    codes.Code
	statusMessage string
	err           error

	payloadSize int

	metadata metadata.MD

	resourceIds api.ResourceIds
}

func (r *Request) getHeaders(name string) []string {
	return r.metadata.Get(name)
}

func (r *Request) getHeader(name string) string {
	h := r.getHeaders(name)
	if len(h) > 0 {
		return h[0]
	}
	return ""
}

func (r *Request) Init(s *Server, ctx CallContext, fields ...logger.Fields) error {

	// basic init
	r.start = time.Now()
	r.server = s

	r.RequestBase.Init(s.App(), s.App().Logger(), s.App().Db(), fields...)
	r.RequestBase.SetErrorManager(s)

	r.params = make(map[string]any)

	r.statusCode = codes.OK
	r.ctx = ctx

	// collect data from headers
	var ok bool
	r.metadata, ok = metadata.FromIncomingContext(ctx)
	if !ok {
		// TODO log error
		return status.Error(codes.Unauthenticated, "metadata missing")
	}

	r.resourceIds = api.NewResourceIdsBase()
	for key, values := range r.metadata {
		if strings.HasPrefix(key, strings.ToLower(r.server.RESOURCE_ID_HEADER_PREFIX)) {
			for _, value := range values {
				typ := strings.TrimPrefix(key, r.server.RESOURCE_ID_HEADER_PREFIX)
				if typ != "" {
					r.resourceIds.SetId(api.NewResourceIdBase(typ, value))
				}
			}
		}
	}

	// get client address and user agent
	p, ok := peer.FromContext(ctx)
	if ok {
		r.clientIp = p.Addr.String()
	}

	if userAgents := r.metadata.Get("user-agent"); len(userAgents) > 0 {
		r.userAgent = userAgents[0]
	}

	// check if it is behind gateway whose auth context can be used
	if s.propagateContextId {
		ctxId := r.getHeader(api.ForwardContext)
		if ctxId != "" {
			r.SetID(ctxId)
			r.SetLoggerField("context", ctxId)
		}
		forwardedOpSource := r.getHeader(api.ForwardOpSource)
		if forwardedOpSource != "" {
			r.forwardedOpSource = forwardedOpSource
			r.SetLoggerField("forwarded_op_source", forwardedOpSource)
		}
	}

	// prepare response
	r.response = &Response{}
	r.response.SetRequest(r)

	// done
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
	if r.GenericError() != nil {
		r.SetLoggerField("status", r.GenericError().Code())
	}
	if r.Message().BinaryContent() != nil {
		content := r.Message().BinaryContent()
		mem.DefaultBufferPool().Put(&content)
		r.Message().SetBinaryContent(nil)
	}
	r.RequestBase.Close("")
	r.server.logRequest(r.Logger(), r.start, r, r.LoggerFields())
}

func (r *Request) GetRequestContent() []byte {
	if r.Message() == nil {
		return nil
	}
	return r.Message().BinaryContent()
}

func AuthKey(key string, directKeyName ...bool) string {
	if utils.OptionalArg(false, directKeyName...) {
		return key
	}
	return utils.ConcatStrings("x-auth-", key)
}

func (r *Request) SetAuthParameter(authMethodProtocol string, key string, value string, directKeyName ...bool) {
	header := metadata.Pairs(AuthKey(key, directKeyName...), value)
	grpc.SetHeader(r.ctx, header)
}

func (r *Request) GetAuthParameter(authMethodProtocol string, key string, directKeyName ...bool) string {
	return r.getHeader(AuthKey(key, directKeyName...))
}

func (r *Request) CheckRequestContent(smsMessage *string, skipSms *bool) error {
	return r.Endpoint().PrecheckBeforeAuth(r, smsMessage, skipSms)
}

func (r *Request) ResourceIds() api.ResourceIds {
	return r.resourceIds
}

func (r *Request) GetRequestPath() string {
	return api_server.FullRequestServicePath(r)
}

func (r *Request) GetResourceId(resourceType string) api.ResourceId {
	return r.resourceIds.GetId(resourceType)
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

func (r *Request) MessageFromRequest(builder func() interface{}) (interface{}, error) {
	if r.Message() == nil {
		return nil, nil
	}
	logicMsg := builder()
	r.Message().SetLogicMessage(logicMsg)
	err := r.Endpoint().TransportRequestToLogic(r.Message())
	if err != nil {
		return nil, err
	}
	return logicMsg, nil
}

func (r *Request) StatusCode() codes.Code {
	return r.statusCode
}

func (r *Request) StatusMessage() string {
	return r.statusMessage
}

func (r *Request) ClientIp() string {
	return r.clientIp
}

func (r *Request) UserAgent() string {
	return r.userAgent
}

func (r *Request) Method() string {
	return r.Endpoint().Name()
}

func (r *Request) Error() error {
	return r.err
}

func (r *Request) Context() context.Context {
	return r.ctx
}

func (r *Request) PayloadSize() int {
	return r.payloadSize
}

func newRequest(ctx context.Context, s *Server, ep api_server.Endpoint) (*Request, op_context.CallContext, error) {

	request := &Request{}
	request.SetEndpoint(ep)

	var err error

	// create and init request
	request.Init(s, ctx)
	epName := ep.Name()
	request.SetName(epName)
	if ep.Resource() != nil {
		request.SetLoggerField("endpoint", ep.Resource().ServicePathPrototype())
	} else if epName != "" {
		request.SetLoggerField("endpoint", epName)
	}

	// fill request origin
	origin := default_op_context.NewOrigin(s.App())
	if origin.Name() != "" {
		origin.SetName(utils.ConcatStrings(origin.Name(), "/", s.Name()))
	} else {
		origin.SetName(s.Name())
	}
	if request.AuthUser() != nil {
		origin.SetUser(auth.AuthUserDisplay(request))
	}
	originSource := request.clientIp
	if request.forwardedOpSource != "" {
		originSource = request.forwardedOpSource
	}
	origin.SetSource(originSource)
	origin.SetSessionClient(request.GetClientId())
	origin.SetUserType(s.OPLOG_USER_TYPE)
	request.SetOrigin(origin)

	c := request.TraceInMethod("Server.Handle")

	// extract tenancy if applicable
	var tenancy multitenancy.Tenancy
	if s.IsMultitenancy() && ep.Resource().IsInTenancy() {
		requestTenancy := request.GetTenancyId()
		request.SetLoggerField("tenancy", requestTenancy)
		if s.SHADOW_TENANCY_PATH {
			tenancy, err = s.tenancies.TenancyByShadowPath(requestTenancy)
		} else {
			tenancy, err = s.tenancies.TenancyByPath(requestTenancy)
		}
		if err != nil {
			request.SetGenericErrorCode(generic_error.ErrorCodeNotFound)
			c.SetMessage("unknown tenancy")
		} else {

			if !tenancy.IsActive() {
				request.SetGenericErrorCode(generic_error.ErrorCodeNotFound)
				err = errors.New("tenancy is not active")
			} else {

				blocked := false
				if !s.ALLOW_BLOCKED_TENANCY_PATH {
					if s.SHADOW_TENANCY_PATH {
						blocked = tenancy.IsBlockedShadowPath()
					} else {
						blocked = tenancy.IsBlockedPath()
					}
				}
				if blocked {
					request.SetGenericErrorCode(generic_error.ErrorCodeNotFound)
					err = errors.New("tenancy path is blocked")
				} else {
					if s.AUTH_FROM_TENANCY_DB {
						request.SetTenancy(tenancy)
					}
				}
			}
		}
		if err == nil {
			if s.TENANCY_ALLOWED_IP_LIST {
				if !s.tenancies.HasIpAddressByPath(requestTenancy, request.clientIp, s.TENANCY_ALLOWED_IP_LIST_TAG) {
					err = errors.New("IP address is not in whitelist")
					request.SetGenericErrorCode(generic_error.ErrorCodeForbidden)
				}
			}
		}
	}

	// // process auth
	// if err == nil {
	// 	err = s.Auth().HandleRequest(request, ep.Resource().ServicePathPrototype(), ep.AccessType())
	// 	if err != nil {
	// 		request.SetGenericErrorCode(auth.ErrorCodeUnauthorized)
	// 	}
	// }
	if s.propagateAuthUser && (request.AuthUser() == nil || request.AuthUser().GetID() == "") {
		userId := request.getHeader(api.ForwardUserId)
		userLogin := request.getHeader(api.ForwardUserLogin)
		userDisplay := request.getHeader(api.ForwardUserDisplay)
		if userId != "" || userLogin != "" || userDisplay != "" {
			authUser := auth.NewAuthUser(userId, userLogin, userDisplay)
			request.SetAuthUser(authUser)
		}
		sessionClient := request.getHeader(api.ForwardSessionClient)
		if sessionClient != "" {
			request.SetClientId(sessionClient)
		}
	}

	// update authenticated user in origin
	if request.AuthUser() != nil {
		origin.SetUser(auth.AuthUserDisplay(request))
	}

	// TODO process access control
	if err == nil {

	}

	// set tenancy
	if tenancy != nil && !s.AUTH_FROM_TENANCY_DB {
		request.SetTenancy(tenancy)
	}

	// done
	return request, c, err
}
