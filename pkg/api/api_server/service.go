package api_server

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/common"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/utils"
)

type ServiceEachEndpointHandler = func(ep Endpoint)

// Interface of service that implements one or more groups of API endpoints.
type Service interface {
	generic_error.ErrorsExtender
	common.WithName
	api.Resource

	SetSupportsMultitenancy(enable bool)
	SupportsMultitenancy() bool

	Server() Server
	AttachToServer(server Server) error

	AddDynamicTables(tables ...*DynamicTableConfig)
	DynamicTables() []*DynamicTableConfig
}

type ServiceBase struct {
	common.WithNameBase
	api.ResourceBase
	generic_error.ErrorsExtenderBase
	server        Server
	dynamicTables []*DynamicTableConfig

	multitenancy bool
}

func (s *ServiceBase) Init(pathName string, multitenancy ...bool) {
	s.ResourceBase.Init(pathName, api.ResourceConfig{Service: true})
	s.dynamicTables = make([]*DynamicTableConfig, 0)
	s.multitenancy = utils.OptionalArg(false, multitenancy...)
}

func (s *ServiceBase) SetSupportsMultitenancy(enable bool) {
	s.multitenancy = enable
}

func (s *ServiceBase) SupportsMultitenancy() bool {
	return s.multitenancy
}

func (s *ServiceBase) Server() Server {
	return s.server
}

func (s *ServiceBase) DynamicTables() []*DynamicTableConfig {
	return s.dynamicTables
}

func (s *ServiceBase) AddDynamicTables(tables ...*DynamicTableConfig) {
	s.dynamicTables = append(s.dynamicTables, tables...)
}

func (s *ServiceBase) AttachToServer(server Server) error {

	s.server = server

	dynamicTables := server.DynamicTables()
	if dynamicTables != nil {
		for _, dynamicTable := range s.DynamicTables() {
			server.DynamicTables().AddTable(dynamicTable)
		}
	}
	return server.RegisterService(s)
}
