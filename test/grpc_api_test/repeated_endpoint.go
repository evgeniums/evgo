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

type RepeatedLogic = Repeated

type RepeatedEndpoint struct {
	api_server.EndpointBase
}

func (e *RepeatedEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("RepeatedEndpoint")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[RepeatedLogic](sctx)
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

func (e *RepeatedEndpoint) NewRequestMessage() interface{} {
	return &RepeatedLogic{}
}

func (e *RepeatedEndpoint) NewResponseMessage() interface{} {
	return &RepeatedLogic{}
}

func NewRepeatedEndpoint(opName ...string) *RepeatedEndpoint {
	ep := &RepeatedEndpoint{}
	ep.Init("Repeated", access_control.Post)
	return ep
}
