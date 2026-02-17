package api_client

import (
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/op_context"
)

type Client interface {
	Exec(ctx op_context.Context, operation api.Operation, cmd interface{}, response interface{}, tenancyPath ...string) error
	Transport() interface{}
	SetPropagateAuthUser(val bool)
	SetPropagateContextId(val bool)
}

type ClientOperation interface {
	Exec(client Client, ctx op_context.Context, operation api.Operation) error
}

type TenancyClientOperation interface {
	Exec(client Client, ctx multitenancy.TenancyContext, operation api.Operation) error
}

func MakeOperationHandler(client Client, clientOperation ClientOperation) api.OperationHandler {
	return func(ctx op_context.Context, operation api.Operation) error {
		return clientOperation.Exec(client, ctx, operation)
	}
}

func MakeTenancyOperationHandler(client Client, clientOperation TenancyClientOperation) api.TenancyOperationHandler {
	return func(ctx multitenancy.TenancyContext, operation api.Operation) error {
		return clientOperation.Exec(client, ctx, operation)
	}
}

type ServiceClient struct {
	generic_error.ErrorsExtenderStub
	api.ResourceBase
	client Client
}

func (s *ServiceClient) Init(client Client, serviceName string) {
	s.client = client
	s.ResourceBase.Init(serviceName, api.ResourceConfig{Service: true})
}

func (s *ServiceClient) Client() Client {
	return s.client
}

func (s *ServiceClient) ApiClient() Client {
	return s.client
}

type Response interface {
	SetHeader(key string, value string)
	GetHeader(key string) string
}
