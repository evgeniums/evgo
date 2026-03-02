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
	"strings"
	"time"

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
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/status"
)

const OriginType = "grpc"
const DefaultGrpcConfigSection string = "grpc"
const RequestContextKey = "gu-request"
const HeaderSizeKey = "gu-hsize"

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

	TENANCY_HEADER string `validate:"omitempty,hostname_rfc1123|alphanum" default:"X-Tenancy-Id"`

	TRANSPORT_CODEC_TYPE string `validate:"required,hostname_rfc1123|alphanum" default:"proto-hatn"`

	STATUS_HEADER            string `validate:"required,hostname_rfc1123|alphanum" default:"x-hatn-status"`
	ID_HEADER                string `validate:"omitempty,hostname_rfc1123|alphanum" default:"x-hatn-id"`
	MESSAGE_TYPE_HEADER      string `validate:"required,hostname_rfc1123|alphanum" default:"x-hatn-mtype"`
	ERROR_FAMILY_HEADER      string `validate:"omitempty,hostname_rfc1123|alphanum" default:"x-hatn-efamily"`
	ERROR_DESCRIPTION_HEADER string `default:"x-hatn-edescription"`
	ERROR_DETAILS_HEADER     string `default:"x-hatn-edetails"`
}

type GrpcServerRunner struct {
	*grpc.Server
}

func (g *GrpcServerRunner) Shutdown(ctx context.Context) error {
	g.GracefulStop()
	return nil
}

type Server struct {
	ServerConfig
	api_server.ServerBase
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

	handlers map[string]UnaryHandler
	services map[string]api_server.Service
}

func NewServer() *Server {
	s := &Server{}
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

func (s *Server) IsMultitenancy() bool {
	return !s.DISABLE_MULTITENANCY && multitenancy.IsMultiTenancy(s.tenancies)
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

	s.handlers = map[string]UnaryHandler{}
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
		req := ctx.Value(RequestContextKey)
		err = status.Errorf(codes.Internal, "internal server error")
		if request, ok := req.(*Request); ok {
			request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			request.SetLoggerField("status", request.GenericError().Code())
			request.statusCode = status.Code(err)
			request.statusMessage = "application crashed"
			request.Close()
		}
		return
	}
	opts := []recovery.Option{
		recovery.WithRecoveryHandlerContext(crashRecoveryFunc),
	}

	// create codec wrapper
	codecWrapper := &RequestCodec{
		parent: encoding.GetCodec(proto.Name),
		server: s,
	}

	// create grpc server
	s.grpcServer = &GrpcServerRunner{
		Server: grpc.NewServer(
			grpc.ForceServerCodec(codecWrapper),
			grpc.StatsHandler(&sizeStatsHandler{}),
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

	listener, err := net.Listen(s.PROTOCOL, s.address())
	if err != nil {
		msg := "TCP listening failed"
		s.App().Logger().Fatal(msg, err, logger.Fields{"name": s.Name()})
		app_context.AbortFatal(s.App(), msg)
	}

	fin.AddRunner(s.grpcServer, &background_worker.RunnerConfig{Name: optional.NewString(s.Name())})

	go func() {
		s.App().Logger().Info("Running gRPC API server", logger.Fields{"name": s.Name(), "address": listener.Addr().String()})
		err := s.grpcServer.Serve(listener)
		if err != nil {
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

func (s *Server) GrpcUnaryHandler(service api_server.Service, ep api_server.Endpoint) *UnaryHandler {
	handler, ok := s.handlers[s.FullMethodName(service, ep)]
	if !ok {
		return nil
	}
	return &handler
}

func (s *Server) AddEndpoint(service api_server.Service, ep api_server.Endpoint, methods *[]grpc.MethodDesc) {

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
	_, hasEndpoint := s.handlers[fullMethodName]
	if hasEndpoint {
		s.App().Logger().Warn("Grpc API server: duplicate endpoint", logger.Fields{"method": fullMethodName})
	}
	s.handlers[fullMethodName] = handler

	grpcMethod := grpc.MethodDesc{
		MethodName: ep.Name(),
		Handler:    handler.handle,
	}
	*methods = append(*methods, grpcMethod)

	s.App().Logger().Info("Grpc API server: register endpoint", logger.Fields{"method": fullMethodName})
}

func (s *Server) MakeResponseError(gerr generic_error.Error) (int, generic_error.Error) {
	code := s.ErrorProtocolCode(gerr.Code())
	return code, gerr
}

func (s *Server) RegisterService(service api_server.Service) error {

	methods := []grpc.MethodDesc{}

	service.EachOperation(func(op api.Operation) error {
		ep, ok := op.(api_server.Endpoint)
		if !ok {
			return fmt.Errorf("invalid opertaion type, must be endpoint: %s", op.Name())
		}
		s.AddEndpoint(service, ep, &methods)
		return nil
	})

	serviceName := service.Package() + "." + service.Name()

	serviceDesc := &grpc.ServiceDesc{
		ServiceName: serviceName,
		HandlerType: (*interface{})(nil),
		Methods:     methods,
		Streams:     []grpc.StreamDesc{},
		Metadata:    "",
	}

	s.grpcServer.RegisterService(serviceDesc, nil)
	s.services[service.Name()] = service

	/*	TODO Maybe run it after service initialization
		serviceInfo := s.grpcServer.GetServiceInfo()
		for serviceName, info := range serviceInfo {
			fmt.Printf("Service: %s\n", serviceName)
			for _, method := range info.Methods {
				// Full path format: /package.Service/Method
				fmt.Printf("  - Endpoint: /%s/%s\n", serviceName, method.Name)
			}
		}
	*/
	return nil
}

type methodContext interface {
	StatusCode() codes.Code
	StatusMessage() string
	ClientIp() string
	UserAgent() string
	Method() string
	Error() error
	PayloadSize() int
	Context() context.Context
}

func (s *Server) logRequest(log logger.Logger, start time.Time, callCtx methodContext, extraFields ...logger.Fields) {

	stop := time.Since(start)
	latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))

	headerSize := 0
	if callCtx.Context() != nil {
		if info, ok := callCtx.Context().Value(HeaderSizeKey).(*SizeInfo); ok {
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
	logger.AppendFields(fields, extraFields...)

	if StatusError(callCtx.StatusCode()) {
		log.Error(s.logPrefix, errors.New("internal server error"), fields)
	} else if StatusWarn(callCtx.StatusCode()) {
		log.Warn(s.logPrefix, fields)
	} else {
		log.Info(s.logPrefix, fields)
	}
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

func StatusError(status codes.Code) bool {
	return status == codes.Internal ||
		status == codes.DataLoss ||
		status == codes.Unknown ||
		status == codes.Unimplemented ||
		status == codes.Unavailable ||
		status == codes.DeadlineExceeded
}

func StatusWarn(status codes.Code) bool {
	return status != codes.OK
}
