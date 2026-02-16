package grpc_api_server

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/app_context"
	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/background_worker"
	"github.com/evgeniums/go-utils/pkg/config/object_config"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/pool"
	"github.com/evgeniums/go-utils/pkg/utils"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/realip"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/markphelps/optional"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var DefaultGrpcConfigSection string = "grpc"
var TenancyHeader string = "x-tenant-id"

type ServerConfig struct {
	api_server.ServerBaseConfig

	HOST                       string `default:"127.0.0.1" validate:"ip"`
	PORT                       uint16 `validate:"required"`
	PROTOCOL                   string `default:"tcp" validate:"omitempty,oneof=tcp udp"`
	TRUSTED_PROXIES            []string
	VERBOSE                    bool
	VERBOSE_BODY_MAX_LENGTH    int `default:"2048"`
	ALLOW_BLOCKED_TENANCY_PATH bool
	AUTH_FROM_TENANCY_DB       bool `default:"true"`
	SHADOW_TENANCY_PATH        bool

	TENANCY_ALLOWED_IP_LIST_TAG string
	TENANCY_ALLOWED_IP_LIST     bool

	REAL_IP_HEADER string `validate:"required" default:"X-Forwarded-For"`

	TENANCY_HEADER string `validate:"omitempty,alphanum"`
}

type GrpcServerRunner struct {
	*grpc.Server
}

func (g *GrpcServerRunner) Shutdown(ctx context.Context) error {
	g.GracefulStop()
	return nil
}

type UnaryHandler struct {
	endpoint            api_server.Endpoint
	server              *Server
	grpcUnaryServerInfo *grpc.UnaryServerInfo
	newProtoMessage     func() interface{}

	transportToLogic func(interface{}) RequestMessage
	logicToTransport func(interface{}) interface{}
}

func (u *UnaryHandler) SetTransportToLogicMessageMapper(mapper func(interface{}) RequestMessage) {
	u.transportToLogic = mapper
}

func (u *UnaryHandler) SetLogicToTransportMessageMapper(mapper func(interface{}) interface{}) {
	u.logicToTransport = mapper
}

func (u *UnaryHandler) SetTransportMessageBuilder(builder func() interface{}) {
	u.newProtoMessage = builder
}

type Server struct {
	ServerConfig
	app_context.WithAppBase
	generic_error.ErrorManagerBaseHttp
	auth.WithAuthBase

	tenancies multitenancy.Multitenancy

	configPoolService pool.PoolService

	grpcServer *GrpcServerRunner

	tenancyResource api.Resource

	dynamicTables api_server.DynamicTables

	propagateContextId bool
	propagateAuthUser  bool

	logPrefix string

	hostname string
	crashed  bool

	handlers map[string]UnaryHandler
	services map[string]api_server.Service
}

func NewServer() *Server {

	s := &Server{}

	s.TENANCY_HEADER = TenancyHeader

	return s
}

func (s *Server) ConfigPoolService() pool.PoolService {
	return s.configPoolService
}

func (s *Server) SetPropagateContextId(val bool) {
	s.propagateContextId = val
}

func (s *Server) SetPropagateAuthUser(val bool) {
	s.propagateAuthUser = val
}

func (s *Server) SetConfigFromPoolService(service pool.PoolService, public ...bool) {

	s.configPoolService = service

	pub := utils.OptionalArg(true, public...)

	s.SetName(service.Name())
	s.API_VERSION = service.ApiVersion()
	s.HOST = service.IpAddress()

	if pub {
		if s.HOST == "" {
			s.HOST = service.PublicHost()
		}
		s.PORT = service.PublicPort()
	} else {
		if s.HOST == "" {
			s.HOST = service.PrivateHost()
		}
		s.PORT = service.PrivatePort()
	}
}

func (s *Server) Config() interface{} {
	return &s.ServerConfig
}

func (s *Server) Testing() bool {
	return s.App().Testing()
}

func (s *Server) DynamicTables() api_server.DynamicTables {
	return s.dynamicTables
}

func (s *Server) TenancyManager() multitenancy.Multitenancy {
	return s.tenancies
}

func (s *Server) address() string {
	if strings.Contains(s.HOST, "::") {
		return fmt.Sprintf("[%s]:%d", s.HOST, s.PORT)
	}
	return fmt.Sprintf("%s:%d", s.HOST, s.PORT)
}

// func (s *Server) logGinRequest(log logger.Logger, path string, start time.Time, ginCtx *gin.Context, extraFields ...logger.Fields) {

// 	stop := time.Since(start)
// 	latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))
// 	statusCode := ginCtx.Writer.Status()
// 	clientIP := ginCtx.ClientIP()
// 	clientUserAgent := ginCtx.Request.UserAgent()
// 	referer := ginCtx.Request.Referer()
// 	dataLength := ginCtx.Writer.Size()
// 	if dataLength < 0 {
// 		dataLength = 0
// 	}

// 	fields := logger.Fields{
// 		"hostname":    s.hostname,
// 		"http_code":   statusCode,
// 		"latency":     latency,
// 		"client_ip":   clientIP,
// 		"method":      ginCtx.Request.Method,
// 		"path":        path,
// 		"data_length": dataLength,
// 		"user_agent":  clientUserAgent,
// 		"server_name": s.Name(),
// 	}
// 	if referer != "" {
// 		fields["referer"] = referer
// 	}
// 	logger.AppendFields(fields, extraFields...)

// 	if len(ginCtx.Errors) > 0 {
// 		log.Error(s.logPrefix, errors.New(ginCtx.Errors.ByType(gin.ErrorTypePrivate).String()), fields)
// 	} else {
// 		if statusCode >= http.StatusInternalServerError {
// 			log.Error(s.logPrefix, errors.New("internal server error"), fields)
// 		} else if statusCode >= http.StatusBadRequest {
// 			log.Warn(s.logPrefix, fields)
// 		} else {
// 			log.Info(s.logPrefix, fields)
// 		}
// 	}

// 	ginCtx.Set("logged", true)
// }

// func (s *Server) ginDefaultLogger() gin.HandlerFunc {
// 	return func(ginCtx *gin.Context) {

// 		path := ginCtx.Request.URL.Path
// 		start := time.Now()

// 		ginCtx.Next()

// 		// skip if request was already logged
// 		_, logged := ginCtx.Get("logged")
// 		if logged {
// 			return
// 		}

// 		if s.crashed {
// 			s.crashed = false
// 			s.logGinRequest(s.App().Logger(), path, start, ginCtx, logger.Fields{"status": "app_crashed"})
// 		} else {
// 			s.logGinRequest(s.App().Logger(), path, start, ginCtx, logger.Fields{"status": s.notFoundError.Code()})
// 		}
// 	}
// }

func (s *Server) IsMultitenancy() bool {
	return !s.DISABLE_MULTITENANCY && multitenancy.IsMultiTenancy(s.tenancies)
}

func (s *Server) Init(ctx app_context.Context, auth auth.Auth, tenancyManager multitenancy.Multitenancy, configPath ...string) error {

	var err error
	s.hostname = ctx.Hostname()
	ctx.Logger().Info("REST API server: init gin server", logger.Fields{"hostname": s.hostname})

	s.WithAppBase.Init(ctx)
	s.ErrorManagerBaseHttp.Init()
	s.WithAuthBase.Init(auth)
	auth.AttachToErrorManager(s)

	s.tenancies = tenancyManager

	s.handlers = map[string]UnaryHandler{}
	s.services = map[string]api_server.Service{}

	if s.IsMultitenancy() {
		ctx.Logger().Info("REST API server: enabling multitenancy mode")
		parent := api.NewResource(s.TENANCY_HEADER)
		s.tenancyResource = api.NewResource(s.TENANCY_HEADER, api.ResourceConfig{HasId: true, Tenancy: true})
		parent.AddChild(s.tenancyResource)
	} else {
		ctx.Logger().Info("REST API server: disabling multitenancy mode")
	}

	// load default configuration
	err = object_config.Load(ctx.Cfg(), s, DefaultGrpcConfigSection)
	if err != nil {
		return ctx.Logger().PushFatalStack("failed to load default server configuration", err, logger.Fields{"name": s.Name()})
	}

	// load configuration for this instance
	defaultConfigSection := "grpc_api_server"
	err = object_config.LoadLogValidate(ctx.Cfg(), ctx.Logger(), ctx.Validator(), s, defaultConfigSection, configPath...)
	if err != nil {
		return ctx.Logger().PushFatalStack("failed to load server configuration", err, logger.Fields{"name": s.Name()})
	}

	// setup trusted proxies
	trustedProxies := []netip.Prefix{}
	for _, proxy := range s.TRUSTED_PROXIES {
		trustedSubnet, err := netip.ParsePrefix(proxy)
		if err != nil {
			return ctx.Logger().PushFatalStack("invalid trusted proxy in server configuration", err, logger.Fields{"name": s.Name(), "invalid_proxy": proxy})
		}
		trustedProxies = append(trustedProxies, trustedSubnet)
	}
	realIpHeaders := []string{realip.XForwardedFor, realip.XRealIp}
	if s.REAL_IP_HEADER != "" {
		realIpHeaders = []string{s.REAL_IP_HEADER}
	}

	// setup crash recovery
	crachRecoveryFunc := func(p any) (err error) {
		s.crashed = true
		// TODO log error
		return status.Errorf(codes.Internal, "panic triggered: %v", p)
	}
	opts := []recovery.Option{
		recovery.WithRecoveryHandler(crachRecoveryFunc),
	}

	// create grpc server
	s.grpcServer = &GrpcServerRunner{
		Server: grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				realip.UnaryServerInterceptor(trustedProxies, realIpHeaders),
				recovery.UnaryServerInterceptor(opts...),
			),
			grpc.ChainStreamInterceptor(
				realip.StreamServerInterceptor(trustedProxies, realIpHeaders),
				recovery.StreamServerInterceptor(opts...),
			),
		)}

	// set server name
	name := s.Name()
	if name == "" {
		name = ctx.AppInstance()
		if name == "" {
			name = ctx.Application()
		}
		s.SetName(name)
	}
	s.logPrefix = "Served gRPC"

	// done
	return nil
}

func (s *Server) Run(fin background_worker.Finisher) {

	listener, _ := net.Listen(s.PROTOCOL, s.address())

	fin.AddRunner(s.grpcServer, &background_worker.RunnerConfig{Name: optional.NewString(s.Name())})

	go func() {
		s.App().Logger().Info("Running gRPC API server", logger.Fields{"name": s.Name(), "address": listener.Addr})
		err := s.grpcServer.Serve(listener)
		if err != nil {
			msg := "failed to run gRPC server"
			fmt.Printf("%s %s: %s\n", msg, s.Name(), err)
			s.App().Logger().Fatal(msg, err, logger.Fields{"name": s.Name()})
			app_context.AbortFatal(s.App(), msg)
		}
	}()
}

const OriginType = "grpc"

func RequestInializingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// request := &Request{}

		return nil, nil

		// // 1. Extract IP (Assumes realip interceptor ran previously)
		// if addr, ok := realip.FromContext(ctx); ok {
		// 	ri.ClientIP = addr.String()
		// }

		// // 2. Extract Metadata (Headers)
		// if md, ok := metadata.FromIncomingContext(ctx); ok {
		// 	if ids := md.Get("x-device-id"); len(ids) > 0 {
		// 		ri.DeviceID = ids[0]
		// 	}
		// 	if tokens := md.Get("authorization"); len(tokens) > 0 {
		// 		ri.AuthToken = tokens[0]
		// 	}
		// }

		// // 3. Inject into context and proceed
		// newCtx := context.WithValue(ctx, RequestContextKey, ri)
		// return handler(newCtx, req)
	}
}

// func requestHandler(s *Server, ep api_server.Endpoint) gin.HandlerFunc {
// 	return func(ginCtx *gin.Context) {

// 		var err error

// 		// create and init request
// 		request := &Request{}
// 		request.Init(s, ginCtx, ep)
// 		epName := ep.Name()
// 		request.SetName(epName)
// 		request.SetLoggerField("endpoint", ep.Resource().ServicePathPrototype())

// 		c := request.TraceInMethod("Server.RequestHandler")

// 		// dum request in verbose mode
// 		if s.VERBOSE {
// 			dumpBody := ginCtx.Request.ContentLength > 0 && int(ginCtx.Request.ContentLength) <= s.VERBOSE_BODY_MAX_LENGTH
// 			b, _ := httputil.DumpRequest(ginCtx.Request, dumpBody)
// 			c.Logger().Debug("Dump server HTTP request", logger.Fields{"request": string(b)})
// 		}

// 		// extract tenancy if applicable
// 		var tenancy multitenancy.Tenancy
// 		if s.IsMultitenancy() && ep.Resource().IsInTenancy() {
// 			tenancyInPath := request.GetResourceId(s.TENANCY_PARAMETER)
// 			request.SetLoggerField("tenancy", tenancyInPath)
// 			if s.SHADOW_TENANCY_PATH {
// 				tenancy, err = s.tenancies.TenancyByShadowPath(tenancyInPath.Value())
// 			} else {
// 				tenancy, err = s.tenancies.TenancyByPath(tenancyInPath.Value())
// 			}
// 			if err != nil {
// 				request.SetGenericErrorCode(generic_error.ErrorCodeNotFound)
// 				c.SetMessage("unknown tenancy")
// 			} else {

// 				if !tenancy.IsActive() {
// 					request.SetGenericErrorCode(generic_error.ErrorCodeNotFound)
// 					err = errors.New("tenancy is not active")
// 				} else {

// 					blocked := false
// 					if !s.ALLOW_BLOCKED_TENANCY_PATH {
// 						if s.SHADOW_TENANCY_PATH {
// 							blocked = tenancy.IsBlockedShadowPath()
// 						} else {
// 							blocked = tenancy.IsBlockedPath()
// 						}
// 					}
// 					if blocked {
// 						request.SetGenericErrorCode(generic_error.ErrorCodeNotFound)
// 						err = errors.New("tenancy path is blocked")
// 					} else {
// 						if s.AUTH_FROM_TENANCY_DB {
// 							request.SetTenancy(tenancy)
// 						}
// 					}
// 				}
// 			}
// 			if err == nil {
// 				if s.TENANCY_ALLOWED_IP_LIST {
// 					if !s.tenancies.HasIpAddressByPath(tenancyInPath.Value(), request.clientIp, s.TENANCY_ALLOWED_IP_LIST_TAG) {
// 						err = errors.New("IP address is not in whitelist")
// 						request.SetGenericErrorCode(generic_error.ErrorCodeForbidden)
// 					}
// 				}
// 				// TODO white list for non tenancy mode
// 			}
// 		}

// 		// process CSRF
// 		if err == nil {
// 			if s.csrf != nil {
// 				_, err = s.csrf.Handle(request)
// 			}
// 		}

// 		// process auth
// 		if err == nil {
// 			err = s.Auth().HandleRequest(request, ep.Resource().ServicePathPrototype(), ep.AccessType())
// 			if err != nil {
// 				request.SetGenericErrorCode(auth.ErrorCodeUnauthorized)
// 			}
// 		}
// 		if s.propagateAuthUser && (request.AuthUser() == nil || request.AuthUser().GetID() == "") {
// 			userId := ginCtx.GetHeader(api.ForwardUserId)
// 			userLogin := ginCtx.GetHeader(api.ForwardUserLogin)
// 			userDisplay := ginCtx.GetHeader(api.ForwardUserDisplay)
// 			if userId != "" || userLogin != "" || userDisplay != "" {
// 				authUser := auth.NewAuthUser(userId, userLogin, userDisplay)
// 				request.SetAuthUser(authUser)
// 			}
// 			sessionClient := ginCtx.GetHeader(api.ForwardSessionClient)
// 			if sessionClient != "" {
// 				request.SetClientId(sessionClient)
// 			}
// 		}

// 		origin := default_op_context.NewOrigin(s.App())
// 		if origin.Name() != "" {
// 			origin.SetName(utils.ConcatStrings(origin.Name(), "/", s.Name()))
// 		} else {
// 			origin.SetName(s.Name())
// 		}
// 		if request.AuthUser() != nil {
// 			origin.SetUser(auth.AuthUserDisplay(request))
// 		}
// 		originSource := request.clientIp
// 		if request.forwardedOpSource != "" {
// 			originSource = request.forwardedOpSource
// 		}
// 		origin.SetSource(originSource)
// 		origin.SetSessionClient(request.GetClientId())
// 		origin.SetUserType(s.OPLOG_USER_TYPE)
// 		request.SetOrigin(origin)

// 		// TODO process access control
// 		if err == nil {

// 		}

// 		// set tenancy
// 		if tenancy != nil && !s.AUTH_FROM_TENANCY_DB {
// 			request.SetTenancy(tenancy)
// 		}

// 		// call endpoint's request handler
// 		if err == nil {
// 			err = ep.HandleRequest(request)
// 			if err != nil {
// 				request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
// 			}
// 		}

// 		// close context with sending response to client
// 		if err != nil {
// 			c.SetError(err)
// 		}
// 		request.TraceOutMethod()
// 		request.Close()
// 	}
// }

func (s *Server) FullMethodName(service api_server.Service, ep api_server.Endpoint) string {
	return fmt.Sprintf("/%s/%s", service.Name(), ep.Name())
}

func (s *Server) GrpcUnaryHandler(service api_server.Service, ep api_server.Endpoint) *UnaryHandler {
	handler, ok := s.handlers[s.FullMethodName(service, ep)]
	if !ok {
		return nil
	}
	return &handler
}

func (s *Server) AddEndpoint(service api_server.Service, ep api_server.Endpoint, methods []grpc.MethodDesc) {

	if ep.TestOnly() && !s.Testing() {
		return
	}

	ep.AttachToErrorManager(s)
	if s.IsMultitenancy() {
		s.tenancyResource.AddChild(ep.Resource().ServiceResource())
	}

	fullMethodName := s.FullMethodName(service, ep)
	info := &grpc.UnaryServerInfo{
		Server:     s.grpcServer,
		FullMethod: fullMethodName,
	}
	handler := UnaryHandler{endpoint: ep, server: s, grpcUnaryServerInfo: info}
	s.handlers[fullMethodName] = handler

	grpcMethod := grpc.MethodDesc{
		MethodName: "CustomMethod",
		Handler:    handler.handle,
	}

	methods = append(methods, grpcMethod)
}

func (s *Server) MakeResponseError(gerr generic_error.Error) (int, generic_error.Error) {
	code := s.ErrorProtocolCode(gerr.Code())
	return code, gerr
}

type grpcContextKey string

const requestContextKey grpcContextKey = "request"

func (u *UnaryHandler) handle(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {

	request, ok := ctx.Value(requestContextKey).(*Request)
	if !ok {
		return nil, status.Errorf(codes.Internal, "invalid type of gRPC request")
	}
	request.SetEndpoint(u.endpoint)

	var msg interface{}
	if u.newProtoMessage == nil {
		// TODO make message automatically
	} else {
		msg = u.newProtoMessage()
	}
	if msg != nil {
		if err := dec(msg); err != nil {
			return nil, err
		}
	}

	finalHandler := func(ctx context.Context, req interface{}) (interface{}, error) {

		if u.transportToLogic == nil {
			request.message = &RequestMessageBase{message: req}
		} else {
			request.message = u.transportToLogic(req)
		}
		err := u.endpoint.HandleRequest(request)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to handle gRPC request")
		}

		var response interface{}
		if request.Response().Message() != nil {
			if u.logicToTransport == nil {
				response = request.Response().Message
			} else {
				response = u.logicToTransport(request.Response().Message)
			}
		}

		return response, status.Error(codes.OK, "success")
	}

	if interceptor == nil {
		return finalHandler(ctx, msg)
	}

	return interceptor(ctx, msg, u.grpcUnaryServerInfo, finalHandler)
}

func (s *Server) RegisterService(service api_server.Service) error {

	methods := []grpc.MethodDesc{}

	service.EachOperation(func(op api.Operation) error {
		ep, ok := op.(api_server.Endpoint)
		if !ok {
			return fmt.Errorf("invalid opertaion type, must be endpoint: %s", op.Name())
		}
		s.AddEndpoint(service, ep, methods)
		return nil
	})

	serviceDesc := &grpc.ServiceDesc{
		ServiceName: service.Name(),
		HandlerType: (*interface{})(nil),
		Methods:     methods,
		Streams:     []grpc.StreamDesc{},
		Metadata:    "manual-registration",
	}

	s.grpcServer.RegisterService(serviceDesc, nil)
	s.services[service.Name()] = service

	return nil
}
