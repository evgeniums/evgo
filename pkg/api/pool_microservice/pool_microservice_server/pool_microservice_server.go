package pool_microservice_server

import (
	"github.com/evgeniums/evgo/pkg/api/noauth_server"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/multitenancy/app_with_multitenancy"
)

type PoolMicroserviceServer struct {
	noauth_server.NoAuthServer
}

type Config = noauth_server.Config

func New(poolServiceType string, config ...Config) *PoolMicroserviceServer {
	s := &PoolMicroserviceServer{}
	s.Construct(poolServiceType, config...)
	return s
}

func (s *PoolMicroserviceServer) Construct(poolServiceType string, config ...Config) {
	if len(config) != 0 {
		cfg := config[0]
		if cfg.DefaultPoolServiceType == "" {
			cfg.DefaultPoolServiceType = poolServiceType
		}
		s.NoAuthServer.Construct(cfg)
	} else {
		cfg := Config{DefaultPoolServiceType: poolServiceType}
		s.NoAuthServer.Construct(cfg)
	}
}

func (s *PoolMicroserviceServer) Init(app app_with_multitenancy.AppWithMultitenancy, configPath ...string) error {

	err := object_config.LoadLogValidate(app.Cfg(), app.Logger(), app.Validator(), s, "microservice_server", configPath...)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load microservice server configuration", err)
	}
	err = s.NoAuthServer.Init(app, configPath...)
	if err != nil {
		return err
	}

	s.SetPropagateContextId(true)
	s.SetPropagateAuthUser(true)

	return nil
}

func (s *PoolMicroserviceServer) SetPropagateContextId(val bool) {
	s.NoAuthServer.SetPropagateContextId(val)
}

func (s *PoolMicroserviceServer) SetPropagateAuthUser(val bool) {
	s.NoAuthServer.SetPropagateAuthUser(val)
}
