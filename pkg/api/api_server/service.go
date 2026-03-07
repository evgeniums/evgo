package api_server

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/utils"
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

	Package() string
	SetPackage(string)
}

type ServiceBase struct {
	common.WithNameBase
	api.ResourceBase
	generic_error.ErrorsExtenderBase
	server        Server
	dynamicTables []*DynamicTableConfig

	multitenancy bool

	packageName string
}

func (s *ServiceBase) Init(pathName string, packageName string, multitenancy ...bool) {
	s.SetName(utils.CapitalizeAscii(pathName))
	s.InitExplicit(pathName, utils.CapitalizeAscii(pathName), packageName, multitenancy...)
}

func (s *ServiceBase) InitExplicit(pathName string, serviceName string, packageName string, multitenancy ...bool) {
	s.SetName(serviceName)
	s.SetPackage(packageName)
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

func (s *ServiceBase) SetEndpointMessageHandlers(handlers map[string]MessageHandlers) {
	s.EachOperation(func(op api.Operation) error {
		ep, ok := op.(Endpoint)
		if !ok {
			return nil
		}
		epHandlers, ok1 := handlers[ep.Name()]
		if !ok1 {
			return nil
		}

		ep.SetMessageHandlers(epHandlers)
		return nil
	})
}

func (s *ServiceBase) SetPackage(value string) {
	s.packageName = value
}

func (s *ServiceBase) Package() string {
	return s.packageName
}

func ServiceName(service string, version string) string {
	if version != "" {
		return fmt.Sprintf("%s.%s", service, version)
	}
	return service
}
