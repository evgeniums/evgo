package rest_api_gin_server

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/evgeniums/go-utils/pkg/access_control"
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/api/api_server/dynamic_table_gorm"
	"github.com/evgeniums/go-utils/pkg/app_context"
	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/auth/auth_methods/auth_csrf"
	"github.com/evgeniums/go-utils/pkg/background_worker"
	"github.com/evgeniums/go-utils/pkg/config/object_config"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/op_context/default_op_context"
	"github.com/evgeniums/go-utils/pkg/pool"
	"github.com/evgeniums/go-utils/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/markphelps/optional"
)

var DefaultHttpConfigSection string = "http"

var TenancyParameter string = "tenancy"

type ServerConfig struct {
	api_server.ServerBaseConfig

	HOST                       string `validate:"ip" default:"127.0.0.1"`
	PORT                       uint16 `validate:"required"`
	PATH_PREFIX                string `default:"/api"`
	TRUSTED_PROXIES            []string
	VERBOSE                    bool
	VERBOSE_BODY_MAX_LENGTH    int `default:"2048"`
	ALLOW_BLOCKED_TENANCY_PATH bool
	AUTH_FROM_TENANCY_DB       bool `default:"true"`
	SHADOW_TENANCY_PATH        bool

	TENANCY_ALLOWED_IP_LIST_TAG string
	TENANCY_ALLOWED_IP_LIST     bool

	TENANCY_PARAMETER string `validate:"omitempty,alphanum"`

	DEFAULT_RESPONSE_JSON string

	FORM_SINGLE_FILE_FIELD string `default:"file"`
}

type AuthParameterGetter = func(r *Request, key string) string
type AuthParameterSetter = func(r *Request, key string, value string)

type Server struct {
	ServerConfig
	app_context.WithAppBase
	generic_error.ErrorManagerBaseHttp
	auth.WithAuthBase

	tenancies multitenancy.Multitenancy

	configPoolService pool.PoolService

	ginEngine     *gin.Engine
	notFoundError generic_error.Error
	hostname      string

	authParamsGetters map[string]AuthParameterGetter
	authParamsSetters map[string]AuthParameterSetter

	csrf            *auth_csrf.AuthCsrf
	tenancyResource api.Resource

	dynamicTables api_server.DynamicTables

	propagateContextId bool
	propagateAuthUser  bool

	logPrefix string

	crashed bool
}

func getHttpHeader(g *gin.Context, name string) string {
	return g.GetHeader(name)
}

func NewServer() *Server {

	s := &Server{}

	s.dynamicTables = dynamic_table_gorm.New()

	csrfKey := func(key string) string {
		return utils.ConcatStrings("x-", key)
	}

	s.authParamsSetters = make(map[string]AuthParameterSetter, 0)
	s.authParamsSetters[auth_csrf.AntiCsrfProtocol] = func(r *Request, key string, value string) {
		r.ginCtx.Header(csrfKey(key), value)
	}

	s.authParamsGetters = make(map[string]AuthParameterGetter, 0)
	s.authParamsGetters[auth_csrf.AntiCsrfProtocol] = func(r *Request, key string) string {
		name := csrfKey(key)
		return getHttpHeader(r.ginCtx, name)
	}

	s.TENANCY_PARAMETER = TenancyParameter

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
	s.PATH_PREFIX = service.PathPrefix()

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

func (s *Server) logGinRequest(log logger.Logger, path string, start time.Time, ginCtx *gin.Context, extraFields ...logger.Fields) {

	stop := time.Since(start)
	latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))
	statusCode := ginCtx.Writer.Status()
	clientIP := ginCtx.ClientIP()
	clientUserAgent := ginCtx.Request.UserAgent()
	referer := ginCtx.Request.Referer()
	dataLength := ginCtx.Writer.Size()
	if dataLength < 0 {
		dataLength = 0
	}

	fields := logger.Fields{
		"hostname":    s.hostname,
		"http_code":   statusCode,
		"latency":     latency,
		"client_ip":   clientIP,
		"method":      ginCtx.Request.Method,
		"path":        path,
		"data_length": dataLength,
		"user_agent":  clientUserAgent,
		"server_name": s.Name(),
	}
	if referer != "" {
		fields["referer"] = referer
	}
	logger.AppendFields(fields, extraFields...)

	if len(ginCtx.Errors) > 0 {
		log.Error(s.logPrefix, errors.New(ginCtx.Errors.ByType(gin.ErrorTypePrivate).String()), fields)
	} else {
		if statusCode >= http.StatusInternalServerError {
			log.Error(s.logPrefix, errors.New("internal server error"), fields)
		} else if statusCode >= http.StatusBadRequest {
			log.Warn(s.logPrefix, fields)
		} else {
			log.Info(s.logPrefix, fields)
		}
	}

	ginCtx.Set("logged", true)
}

func (s *Server) ginDefaultLogger() gin.HandlerFunc {
	return func(ginCtx *gin.Context) {

		path := ginCtx.Request.URL.Path
		start := time.Now()

		ginCtx.Next()

		// skip if request was already logged
		_, logged := ginCtx.Get("logged")
		if logged {
			return
		}

		if s.crashed {
			s.crashed = false
			s.logGinRequest(s.App().Logger(), path, start, ginCtx, logger.Fields{"status": "app_crashed"})
		} else {
			s.logGinRequest(s.App().Logger(), path, start, ginCtx, logger.Fields{"status": s.notFoundError.Code()})
		}
	}
}

func (s *Server) crashRecovery() gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		handle := func(c *gin.Context, err any) {
			s.crashed = true
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		handler := gin.CustomRecovery(handle)
		handler(ginCtx)
	}
}

func (s *Server) NoRoute() gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		ginCtx.JSON(http.StatusNotFound, s.notFoundError)
	}
}

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

	if s.IsMultitenancy() {
		ctx.Logger().Info("REST API server: enabling multitenancy mode")
		parent := api.NewResource(s.TENANCY_PARAMETER)
		s.tenancyResource = api.NewResource(s.TENANCY_PARAMETER, api.ResourceConfig{HasId: true, Tenancy: true})
		parent.AddChild(s.tenancyResource)
	} else {
		ctx.Logger().Info("REST API server: disabling multitenancy mode")
	}

	defaultPath := "rest_api_server"

	// load default configuration
	err = object_config.Load(ctx.Cfg(), s, DefaultHttpConfigSection)
	if err != nil {
		return ctx.Logger().PushFatalStack("failed to load default server configuration", err, logger.Fields{"name": s.Name()})
	}

	// load configuration
	err = object_config.LoadLogValidate(ctx.Cfg(), ctx.Logger(), ctx.Validator(), s, defaultPath, configPath...)
	if err != nil {
		return ctx.Logger().PushFatalStack("failed to load server configuration", err, logger.Fields{"name": s.Name()})
	}

	// load CSRF configuration
	csrfKey := object_config.Key(utils.OptionalArg(defaultPath, configPath...), "csrf")
	if ctx.Cfg().IsSet(csrfKey) {
		s.csrf = auth_csrf.New()
		err = s.csrf.Init(ctx.Cfg(), ctx.Logger(), ctx.Validator(), csrfKey)
		if err != nil {
			return ctx.Logger().PushFatalStack("failed to load anti-CSRF configuration", err)
		}

		s.AddErrorDescriptions(s.csrf.ErrorDescriptions())
		s.AddErrorProtocolCodes(s.csrf.ErrorProtocolCodes())
	}

	// init gin router
	s.ginEngine = gin.New()
	// trusted proxies are needed for correct logging of client IP address
	s.ginEngine.SetTrustedProxies(s.TRUSTED_PROXIES)
	// use default logger for unhandled paths, use recovery middleware to catch panic failures
	s.ginEngine.Use(s.ginDefaultLogger(), s.crashRecovery())

	// set noroute
	e := generic_error.New("not_found", "Requested resource was not found")
	s.notFoundError = e
	s.ginEngine.NoRoute(s.NoRoute())

	name := s.Name()
	if name == "" {
		name = ctx.AppInstance()
		if name == "" {
			name = ctx.Application()
		}
		s.SetName(name)
	}
	s.logPrefix = "Served HTTP"

	// done
	return nil
}

func (s *Server) Run(fin background_worker.Finisher) {

	srv := &http.Server{Addr: s.address(), Handler: s.ginEngine}
	fin.AddRunner(srv, &background_worker.RunnerConfig{Name: optional.NewString(s.Name())})

	go func() {
		s.App().Logger().Info("Running REST API server", logger.Fields{"name": s.Name(), "address": srv.Addr})
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			msg := "failed to start HTTP server"
			fmt.Printf("%s %s: %s\n", msg, s.Name(), err)
			s.App().Logger().Fatal(msg, err, logger.Fields{"name": s.Name()})
			app_context.AbortFatal(s.App(), msg)
		}
	}()
}

const OriginType = "rest_api"

func requestHandler(s *Server, ep api_server.Endpoint) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {

		var err error

		// create and init request
		request := &Request{}
		request.Init(s, ginCtx, ep)
		epName := ep.Name()
		request.SetName(epName)
		request.SetLoggerField("endpoint", ep.Resource().ServicePathPrototype())

		c := request.TraceInMethod("Server.RequestHandler")

		// dum request in verbose mode
		if s.VERBOSE {
			dumpBody := ginCtx.Request.ContentLength > 0 && int(ginCtx.Request.ContentLength) <= s.VERBOSE_BODY_MAX_LENGTH
			b, _ := httputil.DumpRequest(ginCtx.Request, dumpBody)
			c.Logger().Debug("Dump server HTTP request", logger.Fields{"request": string(b)})
		}

		// extract tenancy if applicable
		var tenancy multitenancy.Tenancy
		if s.IsMultitenancy() && ep.Resource().IsInTenancy() {
			tenancyInPath := request.GetResourceId(s.TENANCY_PARAMETER)
			request.SetLoggerField("tenancy", tenancyInPath)
			if s.SHADOW_TENANCY_PATH {
				tenancy, err = s.tenancies.TenancyByShadowPath(tenancyInPath.Value())
			} else {
				tenancy, err = s.tenancies.TenancyByPath(tenancyInPath.Value())
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
					if !s.tenancies.HasIpAddressByPath(tenancyInPath.Value(), request.clientIp, s.TENANCY_ALLOWED_IP_LIST_TAG) {
						err = errors.New("IP address is not in whitelist")
						request.SetGenericErrorCode(generic_error.ErrorCodeForbidden)
					}
				}
				// TODO white list for non tenancy mode
			}
		}

		// process CSRF
		if err == nil {
			if s.csrf != nil {
				_, err = s.csrf.Handle(request)
			}
		}

		// process auth
		if err == nil {
			err = s.Auth().HandleRequest(request, ep.Resource().ServicePathPrototype(), ep.AccessType())
			if err != nil {
				request.SetGenericErrorCode(auth.ErrorCodeUnauthorized)
			}
		}
		if s.propagateAuthUser && (request.AuthUser() == nil || request.AuthUser().GetID() == "") {
			userId := ginCtx.GetHeader(api.ForwardUserId)
			userLogin := ginCtx.GetHeader(api.ForwardUserLogin)
			userDisplay := ginCtx.GetHeader(api.ForwardUserDisplay)
			if userId != "" || userLogin != "" || userDisplay != "" {
				authUser := auth.NewAuthUser(userId, userLogin, userDisplay)
				request.SetAuthUser(authUser)
			}
			sessionClient := ginCtx.GetHeader(api.ForwardSessionClient)
			if sessionClient != "" {
				request.SetClientId(sessionClient)
			}
		}

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

		// TODO process access control
		if err == nil {

		}

		// set tenancy
		if tenancy != nil && !s.AUTH_FROM_TENANCY_DB {
			request.SetTenancy(tenancy)
		}

		// call endpoint's request handler
		if err == nil {
			err = ep.HandleRequest(request)
			if err != nil {
				request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			}
		}

		// close context with sending response to client
		if err != nil {
			c.SetError(err)
		}
		request.TraceOutMethod()
		request.Close()
	}
}

func (s *Server) AddEndpoint(ep api_server.Endpoint, withMultitenancy ...bool) {

	if ep.TestOnly() && !s.Testing() {
		return
	}

	ep.AttachToErrorManager(s)

	method := access_control.Access2HttpMethod(ep.AccessType())
	if method == "" {
		panic(fmt.Sprintf("Invalid HTTP method in endpoint %s for access %d", ep.Name(), ep.AccessType()))
	}

	if s.IsMultitenancy() && utils.OptionalArg(false, withMultitenancy...) {
		s.tenancyResource.AddChild(ep.Resource().ServiceResource())
	}

	path := fmt.Sprintf("%s/%s%s", s.PATH_PREFIX, s.ApiVersion(), ep.Resource().FullPathPrototype())
	s.ginEngine.Handle(method, path, requestHandler(s, ep))
}

func (s *Server) MakeResponseError(gerr generic_error.Error) (int, generic_error.Error) {
	code := s.ErrorProtocolCode(gerr.Code())
	return code, gerr
}

func (s *Server) AuthParameterGetter(authMethodProtocol string) AuthParameterGetter {
	handler, ok := s.authParamsGetters[authMethodProtocol]
	if !ok {
		return nil
	}
	return handler
}

func (s *Server) AuthParameterSetter(authMethodProtocol string) AuthParameterSetter {
	handler, ok := s.authParamsSetters[authMethodProtocol]
	if !ok {
		return nil
	}
	return handler
}

func (s *Server) GinEngine() *gin.Engine {
	return s.ginEngine
}

func (s *Server) RegisterService(service api_server.Service) error {
	return service.EachOperation(func(op api.Operation) error {
		ep, ok := op.(api_server.Endpoint)
		if !ok {
			return fmt.Errorf("invalid opertaion type, must be endpoint: %s", op.Name())
		}
		s.AddEndpoint(ep, service.SupportsMultitenancy())
		return nil
	})
}
