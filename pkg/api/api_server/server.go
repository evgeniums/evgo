package api_server

import (
	"fmt"

	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/background_worker"
	"github.com/evgeniums/go-utils/pkg/common"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/pool"
)

type AuthParameterGetter = func(r *Request, key string) string
type AuthParameterSetter = func(r *Request, key string, value string)

// Interface of generic server that implements some API.
type Server interface {
	generic_error.ErrorManager
	auth.WithAuth

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
}

func AddServiceToServer(s Server, service Service) {
	err := service.AttachToServer(s)
	if err != nil {
		panic(fmt.Errorf("failed to attach service %s to server", service.Type()))
	}
	service.AttachToErrorManager(s)
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
}
