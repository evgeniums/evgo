package grpc_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type MapLogic = Map

type MapEndpoint struct {
	api_server.EndpointBase
}

func (e *MapEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("MapEndpoint")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[MapLogic](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return sctx, err
	}
	jsonDataPretty, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(jsonDataPretty))

	// set response
	resp := cmd
	request.Response().SetMessage(resp)

	// done
	return sctx, nil
}

func (e *MapEndpoint) NewRequestMessage() interface{} {
	return &MapLogic{}
}

func (e *MapEndpoint) NewResponseMessage() interface{} {
	return &MapLogic{}
}

func NewMapEndpoint(opName ...string) *MapEndpoint {
	ep := &MapEndpoint{}
	ep.Init("Map", access_control.Post)
	return ep
}
