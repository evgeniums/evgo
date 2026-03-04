package grpc_api

import (
	"github.com/evgeniums/go-utils/pkg/access_control"
	"github.com/evgeniums/go-utils/pkg/api/api_server"
)

type EchoTokenEndpoint struct {
	api_server.EndpointBase
}

func (e *EchoTokenEndpoint) HandleRequest(request api_server.Request) error {
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
