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

type RepeatedOidLogic = RepeatedOid

// RepeatedOidEndpoint echoes a repeated string field back unchanged, exactly like
// RepeatedEndpoint. Its reason for existing is the client side: hatn's TYPE_OBJECT_ID must
// serialize as protobuf-unpacked `repeated string` (one tag per element), since protobuf
// permits packed encoding only for numeric scalars. A packed client encoding shows up here as
// a single element containing every id concatenated -- each preceded by a stray length byte --
// rather than one element per id, so simply echoing the field back is enough for the client
// test to assert the count and the values.
type RepeatedOidEndpoint struct {
	api_server.EndpointBase
}

func (e *RepeatedOidEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("RepeatedOidEndpoint")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[RepeatedOidLogic](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return sctx, err
	}
	jsonDataPretty, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(jsonDataPretty))

	// Explicit per-element dump: the JSON above hides the difference between one long
	// concatenated value and several clean ones behind quoting/escaping.
	fmt.Printf("RepeatedOidEndpoint received %d oid(s)\n", len(cmd.Voids))
	for i, v := range cmd.Voids {
		fmt.Printf("  [%d] len=%d value=%q\n", i, len(v), v)
	}

	// set response
	resp := cmd
	request.Response().SetMessage(resp)

	// done
	return sctx, nil
}

func (e *RepeatedOidEndpoint) NewRequestMessage() interface{} {
	return &RepeatedOidLogic{}
}

func (e *RepeatedOidEndpoint) NewResponseMessage() interface{} {
	return &RepeatedOidLogic{}
}

func NewRepeatedOidEndpoint(opName ...string) *RepeatedOidEndpoint {
	ep := &RepeatedOidEndpoint{}
	ep.Init("RepeatedOid", access_control.Post)
	return ep
}
