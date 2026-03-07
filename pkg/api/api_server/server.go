package api_server

import (
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/background_worker"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/pool"
)

type AuthParameterGetter = func(r *Request, key string) string
type AuthParameterSetter = func(r *Request, key string, value string)

// Interface of generic server that implements some API.
type Server interface {
	generic_error.ErrorManager
	auth.WithAuth
	app_context.WithApp

	Init(ctx app_context.Context, auth auth.Auth, tenancyManager multitenancy.Multitenancy, configPath ...string) error

	// Get API version.
	ApiVersion() string

	// Run server.
	Run(fin background_worker.Finisher)

	// Check if hateoas links are enabled.
	IsHateoas() bool

	// Get tenancy manager
	TenancyManager() multitenancy.Multitenancy

	// Check for testing mode.
	Testing() bool

	// Get dynamic tables store
	DynamicTables() DynamicTables

	// Load default server configuration from corresponding pool service
	SetConfigFromPoolService(service pool.PoolService, public ...bool)

	// Get pool service used for server configuration
	ConfigPoolService() pool.PoolService

	RegisterService(Service) error

	ListEndpoints()
}

func AddServiceToServer(s Server, service Service) error {
	err := service.AttachToServer(s)
	if err != nil {
		return s.App().Logger().PushFatalStack("failed to attach service to server", err, logger.Fields{"service": service.Type()})
	}
	service.AttachToErrorManager(s)
	return nil
}

type ServerBaseConfig struct {
	common.WithNameBaseConfig
	API_VERSION          string `validate:"required"`
	HATEOAS              bool
	OPLOG_USER_TYPE      string `default:"server_user"`
	DISABLE_MULTITENANCY bool
}

func (s *ServerBaseConfig) ApiVersion() string {
	return s.API_VERSION
}

func (s *ServerBaseConfig) IsHateoas() bool {
	return s.HATEOAS
}

type ServerBase struct {
	app_context.WithAppBase
}

func (s *ServerBase) ListEndpoints() {
}
