package api_server

import (
	"context"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type DynamicTableEndpoint struct {
	ResourceEndpoint
	service *DynamicTablesService
}

func NewDynamicTableEndpoint(service *DynamicTablesService) *DynamicTableEndpoint {
	ep := &DynamicTableEndpoint{service: service}
	InitResourceEndpoint(ep, "table-config", "DynamicTableConfig", access_control.Get)
	return ep
}

func (e *DynamicTableEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[Request](sctx)
	c := request.TraceInMethod("GetDynamicTable")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := ParseValidateRequest[DynamicTableQuery](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return sctx, err
	}

	// get table
	table, err := e.service.Server().DynamicTables().Table(sctx, cmd.Path)
	if err != nil {
		c.SetMessage("failed to find table for path")
		request.SetGenericErrorCode(generic_error.ErrorCodeNotFound)
		return sctx, err
	}

	// set response
	request.Response().SetMessage(table)

	// done
	return sctx, nil
}

type DynamicTablesService struct {
	ServiceBase
}

func NewDynamicTablesService(multitenancy ...bool) *DynamicTablesService {

	s := &DynamicTablesService{}

	s.Init("dynamic-tables", api.PackageName, multitenancy...)
	s.AddChild(NewDynamicTableEndpoint(s))

	return s
}
