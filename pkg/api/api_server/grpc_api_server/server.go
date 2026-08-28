package grpc_api_server

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/netip"
	"runtime"
	"time"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/background_worker"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/realip"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/markphelps/optional"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

const OriginType = "grpc"
const DefaultGrpcConfigSection string = "grpc"
const HeaderSizeKey = "gu-hsize"

type ServerConfig struct {
	api_server.ServerBaseConfig

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

	TENANCY_HEADER string `validate:"omitempty,hostname_rfc1123|alphanum" default:"X-Tenancy-Id"`

	TRANSPORT_CODEC_TYPE string `validate:"required,hostname_rfc1123|alphanum" default:"proto-hatn"`

	STATUS_HEADER             string `validate:"required,hostname_rfc1123|alphanum" default:"x-hatn-status"`
	ID_HEADER                 string `validate:"omitempty,hostname_rfc1123|alphanum" default:"x-hatn-id"`
	MESSAGE_TYPE_HEADER       string `validate:"required,hostname_rfc1123|alphanum" default:"x-hatn-mtype"`
	ERROR_FAMILY_HEADER       string `validate:"omitempty,hostname_rfc1123|alphanum" default:"x-hatn-efamily"`
	ERROR_DESCRIPTION_HEADER  string `default:"x-hatn-edescription"`
	ERROR_DETAILS_HEADER      string `default:"x-hatn-edetails"`
	// See whitemdesktop/docs/error-contract.md - the terminal/retryable disposition and,
	// for DispositionRetryAfter, the delay in seconds.
	ERROR_DISPOSITION_HEADER  string `validate:"omitempty,hostname_rfc1123|alphanum" default:"x-hatn-edisposition"`
	ERROR_RETRY_AFTER_HEADER  string `validate:"omitempty,hostname_rfc1123|alphanum" default:"x-hatn-eretry-after"`
	RESOURCE_ID_HEADER_PREFIX string `default:"x-hatn-rid"`
	GRPC_CODE_HEADER          string `validate:"omitempty,hostname_rfc1123|alphanum" default:"x-grpc-code"`

	TLS_CERTIFICATE_FILE string
	TLS_PRIVATE_KEY_FILE string
	DISABLE_TLS          bool

	DUMP_HEADERS bool

	SHUTDOWN_TIMEOUT int `default:"15"`

	KEEP_ALIVE_PERIOD  int `default:"60"` // Ping the client if it is idle for KEEP_ALIVE_PERIOD
	KEEP_ALIVE_TIMEOUT int `default:"20"` // Wait KEEP_ALIVE_TIMEOUT for a ping response

	// Must be <= the smallest client keep_alive_period (15 s on both desktop and mobile).
	// If this is larger than the client's ping period, the server counts ping strikes and sends
	// GOAWAY too_many_pings on idle connections, causing the very disconnects we want to avoid.
	// 5 s matches the client's min_sent_ping_interval_without_data and leaves ample margin.
	KEEP_ALIVE_MIN_TIME             int  `default:"5"`    // Minimum time between client pings
	KEEP_ALIVE_ALLOW_WITHOUT_STREAM bool `default:"true"` // Allow pings even if no active RPCs

	// Application-level stream heartbeat: on server-streaming calls, send a
	// StreamHeartbeatMessageType message every STREAM_HEARTBEAT_PERIOD seconds to clients
	// that request it via the STREAM_HEARTBEAT_HEADER request metadata (the header value is
	// the client's own requested period in seconds; the server uses min(this, the hint)).
	// 0 disables heartbeats entirely. This is transport-liveness only: it does not touch the
	// auth timer or the message queue, and clients that don't send the header get nothing,
	// so older clients see no behavior change.
	STREAM_HEARTBEAT_PERIOD int    `default:"20"`
	STREAM_HEARTBEAT_HEADER string `validate:"omitempty,hostname_rfc1123|alphanum" default:"x-hatn-stream-hb"`
}

type GrpcServerRunner struct {
	*grpc.Server
	server *Server
}

func (g *GrpcServerRunner) Shutdown(sctx context.Context) error {

	g.server.App().Logger().Info("shutting down gRPC server...")

	// Signal all active streaming handlers to exit their select loops.
	close(g.server.shutdown)

	// Schedule a forced Stop() after the grace period. Running GracefulStop
	// synchronously here (rather than in a goroutine) ensures only one
	// grpc.stop() call is ever in flight at a time. The previous goroutine
	// approach caused a deadlock: stop(true) held s.mu inside
	// handlersWG.Wait() while stop(false) blocked trying to acquire that same
	// lock to call closeServerTransportsLocked.
	timer := time.AfterFunc(
		time.Duration(g.server.SHUTDOWN_TIMEOUT)*time.Second,
		func() {
			g.server.App().Logger().Warn("force stopping gRPC server by timeout")
			g.Stop()
		},
	)
	defer timer.Stop()

	// GracefulStop runs synchronously. It blocks until either:
	//   (a) all existing connections drain naturally (happy path), or
	//   (b) Stop() fires from the timer above, which closes all transports,
	//       cancels stream contexts, unblocks handlersWG.Wait(), and lets
	//       GracefulStop return cleanly.
	g.server.App().Logger().Info("gracefully stopping gRPC server...")
	g.GracefulStop()

	g.server.App().Logger().Info("gRPC server stopped")
	return nil
}

type ServerExtender struct {
	UnaryInterceptors        []grpc.UnaryServerInterceptor
	StreamServerInterceptors []grpc.StreamServerInterceptor
}

type Server struct {
	ServerConfig
	api_server.ServerBase
	generic_error.ErrorManagerBaseHttp
	auth.WithAuthBase
	ServerExtender

	listener *api_server.Listener

	tenancies multitenancy.Multitenancy

	configPoolService pool.PoolService

	grpcServer *GrpcServerRunner

	tenancyResource api.Resource

	dynamicTables api_server.DynamicTables

	propagateContextId bool
	propagateAuthUser  bool

	logPrefix       string
	streamLogPrefix string

	hostname string

	handlers map[string]Handler
	services map[string]api_server.Service

	shutdown chan struct{}

	externalListener bool
}

func NewServer(extender ...ServerExtender) *Server {
	s := &Server{}
	s.shutdown = make(chan struct{})
	if len(extender) > 0 {
		s.ServerExtender = extender[0]
	}
	return s
}

func (s *Server) Runner() *GrpcServerRunner {
	return s.grpcServer
}

func (s *Server) SetListener(lis *api_server.Listener) {
	s.listener = lis
	s.externalListener = true
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

func (s *Server) ensureListener() {
	if s.listener == nil {
		s.listener = &api_server.Listener{}
		s.listener.SetName(s.Name())
		s.listener.WithAppBase.Init(s.App())
	}
}

func (s *Server) SetConfigFromPoolService(service pool.PoolService, public ...bool) {

	s.configPoolService = service

	pub := utils.OptionalArg(true, public...)

	s.SetName(service.Name())
	s.API_VERSION = service.ApiVersion()

	lis := s.listener
	s.ensureListener()

	lis.HOST = service.IpAddress()

	if pub {
		if lis.HOST == "" {
			lis.HOST = service.PublicHost()
		}
		lis.PORT = service.PublicPort()
	} else {
		if lis.HOST == "" {
			lis.HOST = service.PrivateHost()
		}
		lis.PORT = service.PrivatePort()
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

func (s *Server) IsMultitenancy() bool {
	return !s.DISABLE_MULTITENANCY && multitenancy.IsMultiTenancy(s.tenancies)
}

func (s *Server) unknownHandler(srv interface{}, stream grpc.ServerStream) error {

	ctx := stream.Context()
	method, _ := grpc.MethodFromServerStream(stream)
	ep := &api_server.EndpointBase{}
	ep.Init("")

	request, _, _ := newRequest(ctx, s, ep)
	request.SetName(method)
	request.SetGenericErrorCode(generic_error.ErrorCodeUnimplemented)

	err := status.Errorf(codes.Unimplemented, "method %s is not implemented on this server", method)
	request.SetLoggerField("status", request.GenericError().Code())
	request.statusCode = status.Code(err)
	request.statusMessage = "unknown method"
	request.Close(request.sctx)

	return err
}

func (s *Server) Init(ctx app_context.Context, auth auth.Auth, tenancyManager multitenancy.Multitenancy, configPath ...string) error {

	var err error
	s.hostname = ctx.Hostname()
	ctx.Logger().Info("Grpc API server: init API server", logger.Fields{"hostname": s.hostname})

	s.WithAppBase.Init(ctx)
	s.ErrorManagerBaseHttp.Init()
	s.WithAuthBase.Init(auth)
	auth.AttachToErrorManager(s)

	s.tenancies = tenancyManager

	s.handlers = map[string]Handler{}
	s.services = map[string]api_server.Service{}

	if s.IsMultitenancy() {
		ctx.Logger().Info("Grpc API server: enabling multitenancy mode")
		parent := api.NewResource(s.TENANCY_HEADER)
		s.tenancyResource = api.NewResource(s.TENANCY_HEADER, api.ResourceConfig{HasId: true, Tenancy: true})
		parent.AddChild(s.tenancyResource)
	} else {
		ctx.Logger().Info("Grpc API server: disabling multitenancy mode")
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

	// init listener
	if s.listener == nil {
		s.ensureListener()
		err = object_config.LoadLogValidate(ctx.Cfg(), ctx.Logger(), ctx.Validator(), s.listener, defaultConfigSection, configPath...)
		if err != nil {
			return ctx.Logger().PushFatalStack("failed to load server listener configuration", err, logger.Fields{"name": s.Name()})
		}
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
	crashRecoveryFunc := func(ctx context.Context, p any) (err error) {

		const size = 64 << 10 // 64KB
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]

		s.App().Logger().Fatal("application crashed", fmt.Errorf("panic triggered: %v\nStack Trace:\n%s\n", p, buf))
		req := ctx.Value(op_context.OpContextKey{})
		err = status.Errorf(codes.Internal, "internal server error")
		if request, ok := req.(*Request); ok {
			request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			request.SetLoggerField("status", request.GenericError().Code())
			request.statusCode = status.Code(err)
			request.statusMessage = "application crashed"
			request.Close(ctx)
		}
		return
	}
	recoveryOpts := []recovery.Option{recovery.WithRecoveryHandlerContext(crashRecoveryFunc)}

	// setup unary interceptors
	unaryInterceptors := []grpc.UnaryServerInterceptor{}
	if !ctx.Testing() {
		ctx.Logger().Info("Enable crash recovery")
		unaryInterceptors = append(unaryInterceptors, recovery.UnaryServerInterceptor(recoveryOpts...))
	} else {
		ctx.Logger().Warn("Disable crash recovery in testing mode")
	}
	unaryInterceptors = append(unaryInterceptors, realip.UnaryServerInterceptor(trustedProxies, realIpHeaders))
	if len(s.UnaryInterceptors) != 0 {
		unaryInterceptors = append(unaryInterceptors, s.UnaryInterceptors...)
	}

	// setup stream interceptors
	streamInterceptors := []grpc.StreamServerInterceptor{}
	if !ctx.Testing() {
		ctx.Logger().Info("Enable streaming endpoints crash recovery")
		streamInterceptors = append(streamInterceptors, recovery.StreamServerInterceptor(recoveryOpts...))
	} else {
		ctx.Logger().Warn("Disable streaming endpoints crash recovery in testing mode")
	}
	streamInterceptors = append(streamInterceptors, realip.StreamServerInterceptor(trustedProxies, realIpHeaders))
	if len(s.StreamServerInterceptors) != 0 {
		streamInterceptors = append(streamInterceptors, s.StreamServerInterceptors...)
	}

	// create codec wrapper
	pc := encoding.GetCodecV2(proto.Name)
	codecWrapper := &RequestCodec{
		parent: pc,
		server: s,
	}

	// collect server options
	serverOpts := []grpc.ServerOption{grpc.ForceServerCodecV2(codecWrapper),
		grpc.StatsHandler(&sizeStatsHandler{}),
		grpc.UnknownServiceHandler(s.unknownHandler),
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    time.Duration(s.KEEP_ALIVE_PERIOD) * time.Second,  // Ping the client if it is idle for KEEP_ALIVE_PERIOD
			Timeout: time.Duration(s.KEEP_ALIVE_TIMEOUT) * time.Second, // Wait KEEP_ALIVE_TIMEOUT for a ping response
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             time.Duration(s.KEEP_ALIVE_MIN_TIME) * time.Second, // Minimum time between client pings
			PermitWithoutStream: s.KEEP_ALIVE_ALLOW_WITHOUT_STREAM,                  // Allow pings even if no active RPCs
		}),
	}
	if !s.externalListener && !s.DISABLE_TLS && s.TLS_PRIVATE_KEY_FILE != "" {
		creds, err := credentials.NewServerTLSFromFile(s.TLS_CERTIFICATE_FILE, s.TLS_PRIVATE_KEY_FILE)
		if err != nil {
			return ctx.Logger().PushFatalStack("failed to load TLS", err)
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}

	// create grpc server
	s.grpcServer = &GrpcServerRunner{
		Server: grpc.NewServer(serverOpts...),
		server: s,
	}

	// set server name
	name := s.Name()
	if name == "" {
		name = ctx.AppInstance()
		if name == "" {
			name = ctx.Application()
		}
		s.SetName(name)
	}
	s.logPrefix = "UNARY"

	// done
	return nil
}

func (s *Server) Run(fin background_worker.Finisher) {

	s.listener.Run(s.PROTOCOL)

	fin.AddRunner(s.grpcServer, &background_worker.RunnerConfig{Name: optional.NewString(s.Name())})
	s.Serve(s.listener.Listener())
}

func (s *Server) Serve(lis net.Listener) {

	go func() {
		s.App().Logger().Info("Running gRPC API server", logger.Fields{"name": s.Name()})
		err := s.grpcServer.Serve(lis)
		if err != nil && err.Error() != "mux: server closed" {
			msg := "failed to run gRPC server"
			fmt.Printf("%s %s: %s\n", msg, s.Name(), err)
			s.App().Logger().Fatal(msg, err, logger.Fields{"name": s.Name()})
			app_context.AbortFatal(s.App(), msg)
		}
		s.App().Logger().Info("gRPC API server stopped", logger.Fields{"name": s.Name()})
	}()
}

func (s *Server) FullMethodName(service api_server.Service, ep api_server.Endpoint) string {
	return fmt.Sprintf("/%s.%s/%s", service.Package(), service.Name(), ep.Name())
}

func (s *Server) GrpcHandler(service api_server.Service, ep api_server.Endpoint) *Handler {
	handler, ok := s.handlers[s.FullMethodName(service, ep)]
	if !ok {
		return nil
	}
	return &handler
}

func (s *Server) AddEndpoint(service api_server.Service, ep api_server.Endpoint, methods *[]grpc.MethodDesc, streams *[]grpc.StreamDesc) {

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
	handler := Handler{endpoint: ep, server: s, grpcUnaryServerInfo: info}
	_, hasEndpoint := s.handlers[fullMethodName]
	if hasEndpoint {
		s.App().Logger().Warn("Grpc API server: duplicate endpoint", logger.Fields{"method": fullMethodName})
	}
	s.handlers[fullMethodName] = handler
	if ep.IsServerStreaming() {
		grpcStreamMethod := grpc.StreamDesc{
			StreamName:    ep.Name(),
			Handler:       handler.handleServerStream,
			ServerStreams: true,
			ClientStreams: false,
		}
		*streams = append(*streams, grpcStreamMethod)
	} else {
		grpcMethod := grpc.MethodDesc{
			MethodName: ep.Name(),
			Handler:    handler.handleUnary,
		}
		*methods = append(*methods, grpcMethod)
	}

	s.App().Logger().Info("Grpc API server: register endpoint",
		logger.Fields{"method": fullMethodName, "path": ep.Resource().FullPathPrototype(), "server_stream": ep.IsServerStreaming()})
}

func (s *Server) MakeResponseError(gerr generic_error.Error) (int, generic_error.Error) {
	code := s.ErrorProtocolCode(gerr.Code())
	return code, gerr
}

func (s *Server) RegisterService(service api_server.Service) error {

	methods := []grpc.MethodDesc{}
	streams := []grpc.StreamDesc{}

	service.EachOperation(func(op api.Operation) error {
		ep, ok := op.(api_server.Endpoint)
		if !ok {
			return fmt.Errorf("invalid opertaion type, must be endpoint: %s", op.Name())
		}
		s.AddEndpoint(service, ep, &methods, &streams)
		return nil
	})

	serviceName := service.Package() + "." + service.Name()

	serviceDesc := &grpc.ServiceDesc{
		ServiceName: serviceName,
		HandlerType: (*interface{})(nil),
		Methods:     methods,
		Streams:     streams,
		Metadata:    "",
	}

	s.grpcServer.RegisterService(serviceDesc, nil)
	s.services[service.Name()] = service
	return nil
}

func (s *Server) ListEndpoints() {
	serviceInfo := s.grpcServer.GetServiceInfo()
	for serviceName, info := range serviceInfo {
		s.App().Logger().Info("Registered service", logger.Fields{"service": serviceName})
		for _, method := range info.Methods {
			s.App().Logger().Info("Registered endpoint", logger.Fields{"method": fmt.Sprintf("/%s/%s", serviceName, method.Name),
				"server_stream": method.IsServerStream})
		}
	}
}

type methodContext interface {
	StatusCode() codes.Code
	StatusMessage() string
	ClientIp() string
	UserAgent() string
	Method() string
	Error() error
	PayloadSize() int
}

func (s *Server) logRequest(sctx context.Context, log logger.Logger, start time.Time, callCtx methodContext, extraFields logger.Fields, logPrefix ...string) {

	stop := time.Since(start)
	latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))

	headerSize := 0
	sizeInfo := sctx.Value(HeaderSizeKey)
	if sizeInfo != nil {
		if info, ok := sizeInfo.(*SizeInfo); ok {
			headerSize = info.value
		}
	}

	fields := logger.Fields{
		"host":    s.hostname,
		"code":    callCtx.StatusCode(),
		"lat":     latency,
		"ip":      callCtx.ClientIp(),
		"payload": callCtx.PayloadSize(),
		"header":  headerSize,
		"agent":   callCtx.UserAgent(),
		"server":  s.Name(),
	}
	logger.AppendFields(fields, extraFields)
	// "stack" is only meaningful while pinpointing an error/debug record deep inside a call
	// chain; on the request-completion summary line it is redundant with the endpoint/op fields
	// above, so it is dropped here regardless of what extraFields carried.
	delete(fields, "stack")

	prefix := utils.OptionalString(s.logPrefix, logPrefix...)
	if StatusError(callCtx.StatusCode()) {
		log.Error(prefix, errors.New("internal server error"), fields)
	} else if StatusWarn(callCtx.StatusCode()) {
		log.Warn(prefix, fields)
	} else {
		log.Info(prefix, fields)
	}
}

func GRPCToGeneric(st codes.Code) string {
	switch st {
	case codes.OK:
		return generic_error.ErrorCodeSuccess
	case codes.Aborted:
		return generic_error.ErrorCodeIOAborted
	case codes.InvalidArgument:
		return generic_error.ErrorCodeBadRequest
	case codes.DeadlineExceeded:
		return generic_error.ErrorCodeExpired
	case codes.Canceled:
		return generic_error.ErrorCodeIOAborted
	case codes.Unimplemented:
		return generic_error.ErrorCodeUnimplemented
	case codes.PermissionDenied:
		return generic_error.ErrorCodeForbidden
	case codes.AlreadyExists:
		return generic_error.ErrorCodeConflict
	case codes.NotFound:
		return generic_error.ErrorCodeNotFound
	case codes.Unauthenticated:
		return auth.ErrorCodeUnauthorized
	case codes.ResourceExhausted:
		return generic_error.ErrorCodeResourceBusy
	case codes.Unavailable:
		return generic_error.ErrorCodeUnsupported

	}

	return generic_error.ErrorCodeExternalServiceError
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
	case generic_error.HttpStatusClientAborted:
		return codes.Aborted
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

func StatusError(status codes.Code) bool {
	return status == codes.Internal ||
		status == codes.DataLoss ||
		status == codes.Unknown ||
		status == codes.Unavailable ||
		status == codes.DeadlineExceeded
}

func StatusWarn(status codes.Code) bool {
	return status != codes.OK
}
