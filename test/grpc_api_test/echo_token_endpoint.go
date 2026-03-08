package grpc_api

import (
	"context"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type EchoTokenEndpoint struct {
	api_server.EndpointBase
}

func (e *EchoTokenEndpoint) HandleRequest(sctx context.Context) error {
	request := op_context.OpContext[api_server.Request](sctx)
	content := request.GetRequestContent()
	request.Response().SetPayload(content)
	return nil
}

func (e *EchoTokenEndpoint) IsRequestPayloadNeeded() bool {
	return true
}

func NewEchoTokenEndpoint(opName ...string) *EchoTokenEndpoint {
	ep := &EchoTokenEndpoint{}
	ep.Init("EchoToken", access_control.Post)
	return ep
}
