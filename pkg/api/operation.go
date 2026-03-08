package api

import (
	"context"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type OperationHeaders = map[string]string

type Operation interface {
	Name() string

	SetResource(resource Resource)
	Resource() Resource

	AccessType() access_control.AccessType
	Exec(sctx context.Context,
		controller interface{},
		requestMessage interface{},
		resultMessage interface{},
		headers OperationHeaders,
		tenancyArg ...multitenancy.TenancyPath) (interface{}, error)
	SetRunner(runner func(sctx context.Context,
		controller interface{},
		requestMessage interface{},
		resultMessage interface{},
		headers OperationHeaders,
		tenancyArg ...multitenancy.TenancyPath) (interface{}, error))

	TestOnly() bool
	SetTestOnly(val bool)
}

type OperationBase struct {
	name       string
	resource   Resource
	accessType access_control.AccessType
	testOnly   bool

	runner func(sctx context.Context,
		controller interface{},
		requestMessage interface{},
		resultMessage interface{},
		headers OperationHeaders,
		tenancyArg ...multitenancy.TenancyPath) (interface{}, error)
}

func NewOperation(name string, accessType access_control.AccessType, testOnly ...bool) *OperationBase {
	o := &OperationBase{}
	o.Init(name, accessType, testOnly...)
	return o
}

func (o *OperationBase) Init(name string, accessType access_control.AccessType, testOnly ...bool) {
	o.name = name
	o.accessType = accessType
	o.testOnly = utils.OptionalArg(false, testOnly...)
}

func (o *OperationBase) Name() string {
	return o.name
}

func (o *OperationBase) TestOnly() bool {
	return o.testOnly
}

func (o *OperationBase) SetTestOnly(val bool) {
	o.testOnly = val
}

func (o *OperationBase) SetResource(resource Resource) {
	o.resource = resource
}

func (o *OperationBase) Resource() Resource {
	return o.resource
}

func (o *OperationBase) AccessType() access_control.AccessType {
	return o.accessType
}

func (o *OperationBase) Exec(sctx context.Context,
	controller interface{},
	requestMessage interface{},
	resultMessage interface{},
	headers OperationHeaders,
	tenancyArg ...multitenancy.TenancyPath) (interface{}, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	ctx.TraceInMethod("Operation.Exec")
	defer ctx.TraceOutMethod()

	if o.runner == nil {
		return nil, nil
	}

	return o.runner(sctx, controller, requestMessage, resultMessage, headers, tenancyArg...)
}

func (o *OperationBase) SetRunner(runner func(sctx context.Context,
	controller interface{},
	requestMessage interface{},
	resultMessage interface{},
	headers OperationHeaders,
	tenancyArg ...multitenancy.TenancyPath) (interface{}, error)) {
	o.runner = runner
}

type OperationVisitor interface {
	Visit(operation Operation)
}

type OperationVisitors interface {
	Visit(operation Operation)
}

type OperationVisitorsBase struct {
	Visitors map[string]OperationVisitors
}

func (o *OperationVisitorsBase) Visit(operation Operation) {
	v, ok := o.Visitors[operation.Name()]
	if !ok {
		return
	}
	v.Visit(operation)
}

func VisitOperation(operation Operation, visitors ...OperationVisitors) {
	if len(visitors) > 0 {
		visitors[0].Visit(operation)
	}
}
