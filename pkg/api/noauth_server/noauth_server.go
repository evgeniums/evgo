package noauth_server

import (
	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/api/api_server/rest_api_gin_server"
	"github.com/evgeniums/go-utils/pkg/app_context"
	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/config/object_config"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/multitenancy/app_with_multitenancy"
	"github.com/evgeniums/go-utils/pkg/pool"
	"github.com/evgeniums/go-utils/pkg/pool/app_with_pools"
	"github.com/evgeniums/go-utils/pkg/utils"
)

type Server interface {
	ApiServer() api_server.Server
	Auth() auth.Auth
}

type PoolServiceConfigI interface {
	NameOrRole() string
	Type() string
	IsPublic() bool
}

type PoolServiceConfig struct {
	POOL_SERVICE_NAME   string
	POOL_SERVICE_TYPE   string
	PUBLIC_POOL_SERVICE bool
}

func (p *PoolServiceConfig) NameOrRole() string {
	return p.POOL_SERVICE_NAME
}

func (p *PoolServiceConfig) Type() string {
	return p.POOL_SERVICE_TYPE
}

func (p *PoolServiceConfig) IsPublic() bool {
	return p.PUBLIC_POOL_SERVICE
}

type NoAuthServerConfig struct {
	PoolServiceConfig
}

type NoAuthServer struct {
	auth   auth.Auth
	server api_server.Server

	config        NoAuthServerConfig
	restApiServer *rest_api_gin_server.Server

	poolService *pool.PoolServiceBinding
}

type Config struct {
	Auth                     auth.Auth
	Server                   api_server.Server
	DefaultPoolServiceName   string
	DefaultPoolServiceType   string
	DefaultPublicPoolService bool
}

func New(config ...Config) *NoAuthServer {
	s := &NoAuthServer{}
	s.Construct(config...)
	return s
}

func (s *NoAuthServer) Config() interface{} {
	return &s.config
}

func (s *NoAuthServer) Construct(config ...Config) {
	if len(config) != 0 {
		cfg := config[0]
		s.server = cfg.Server
		s.config.POOL_SERVICE_TYPE = cfg.DefaultPoolServiceType
		s.config.POOL_SERVICE_NAME = cfg.DefaultPoolServiceName
		s.config.PUBLIC_POOL_SERVICE = cfg.DefaultPublicPoolService
		s.auth = cfg.Auth
	}

	// noauth
	if s.auth == nil {
		s.auth = auth.NewNoAuth()
	}

	// create REST API server
	if s.server == nil {
		s.restApiServer = rest_api_gin_server.NewServer()
		s.server = s.restApiServer
	}
}

func InitFromPoolService(app app_context.Context, restApiServer *rest_api_gin_server.Server, cfg PoolServiceConfigI) (*pool.PoolServiceBinding, error) {

	var service *pool.PoolServiceBinding

	if cfg.NameOrRole() != "" {

		poolApp, ok := app.(app_with_pools.AppWithPools)
		if !ok {
			return nil, app.Logger().PushFatalStack("invalid application type, must be pool app", nil)
		}

		// check if app with self pool
		selfPool, err := poolApp.Pools().SelfPool()
		if err != nil {
			return nil, app.Logger().PushFatalStack("self pool must be specified for API server", err)
		}

		// find service by name
		service, err = selfPool.ServiceByName(cfg.NameOrRole())
		if err != nil {
			service, err = selfPool.Service(cfg.NameOrRole())
			if err != nil {
				return nil, app.Logger().PushFatalStack("failed to find service with specified name/tole", err, logger.Fields{"name/role": cfg.NameOrRole()})
			}
		}

		if service.TypeName() != cfg.Type() {
			return service, app.Logger().PushFatalStack("invalid service type", err, logger.Fields{"name": cfg.NameOrRole(), "service_type": cfg.Type(), "pool_service_type": service.TypeName()})
		}

		if service.Provider() != app.Application() {
			return service, app.Logger().PushFatalStack("invalid service provider", err, logger.Fields{"name": cfg.NameOrRole(), "application": app.Application(), "pool_service_provider": service.Provider()})
		}

		// load server configuration from service
		restApiServer.SetConfigFromPoolService(service, cfg.IsPublic())
	}

	return service, nil
}

func (s *NoAuthServer) Init(app app_with_multitenancy.AppWithMultitenancy, configPath ...string) error {

	path := utils.OptionalArg("server", configPath...)

	err := object_config.LoadLogValidate(app.Cfg(), app.Logger(), app.Validator(), s, path)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load server configuration", err)
	}

	// init REST API server
	if s.restApiServer != nil {

		s.poolService, err = InitFromPoolService(app, s.restApiServer, &s.config)
		if err != nil {
			return err
		}

		serverPath := object_config.Key(path, "rest_api_server")
		err := s.restApiServer.Init(app, s.auth, app.Multitenancy(), serverPath)
		if err != nil {
			return app.Logger().PushFatalStack("failed to init REST API server", err)
		}
	}

	// done
	return nil
}

func (s *NoAuthServer) SetConfigFromPoolService(service pool.PoolService, public ...bool) {
	if s.restApiServer != nil {
		s.restApiServer.SetConfigFromPoolService(service, public...)
	}
}

func (s *NoAuthServer) SetPropagateContextId(val bool) {
	if s.restApiServer != nil {
		s.restApiServer.SetPropagateContextId(val)
	}
}

func (s *NoAuthServer) SetPropagateAuthUser(val bool) {
	if s.restApiServer != nil {
		s.restApiServer.SetPropagateAuthUser(val)
	}
}

func (s *NoAuthServer) Auth() auth.Auth {
	return s.auth
}

func (s *NoAuthServer) ApiServer() api_server.Server {
	return s.server
}

func (s *NoAuthServer) PoolService() *pool.PoolServiceBinding {
	return s.poolService
}
