package grpc_api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api/api_server"
)

type BasicLogic = Basic

type BasicEndpoint struct {
	api_server.EndpointBase
}

func (e *BasicEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	c := request.TraceInMethod("BasicEndpoint")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[BasicLogic](request)
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
	// proto.Message(resp).ProtoReflect().SetUnknown(nil)
	request.Response().SetMessage(resp)

	// done
	return nil
}

func (e *BasicEndpoint) NewRequestMessage() interface{} {
	return &BasicLogic{}
}

func (e *BasicEndpoint) NewResponseMessage() interface{} {
	return &BasicLogic{}
}

func NewBasicEndpoint(opName ...string) *BasicEndpoint {
	ep := &BasicEndpoint{}
	ep.Init("Basic", access_control.Post)
	return ep
}
