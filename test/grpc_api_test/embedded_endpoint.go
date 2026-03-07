package grpc_api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api/api_server"
)

type EmbeddedLogic = Embedded

type EmbeddedEndpoint struct {
	api_server.EndpointBase
}

func (e *EmbeddedEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("EmbeddedEndpoint")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[EmbeddedLogic](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return err
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
	return nil
}

func (e *EmbeddedEndpoint) NewRequestMessage() interface{} {
	return &EmbeddedLogic{}
}

func (e *EmbeddedEndpoint) NewResponseMessage() interface{} {
	return &EmbeddedLogic{}
}

func NewEmbeddedEndpoint(opName ...string) *EmbeddedEndpoint {
	ep := &EmbeddedEndpoint{}
	ep.Init("Embedded", access_control.Post)
	return ep
}
