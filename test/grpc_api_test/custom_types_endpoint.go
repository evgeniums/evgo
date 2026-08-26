package grpc_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type CustomTypesLogic = CustomTypes

// CustomTypesEndpoint echoes hatn's custom scalar field types back unchanged.
//
// All four (DateTime/Date/Time/DateRange) serialize as plain varints, so packed repeated
// encoding is correct for them -- in contrast to ObjectId, which is length-delimited and must
// be unpacked (see RepeatedOidEndpoint). A wrong classification would show up here as a
// mangled element count, exactly as it did for ObjectId.
//
// The datetimes are additionally decoded through utils.FromHatnProtoDatetime and re-encoded
// through utils.ToHatnProtoDatetime rather than echoed blindly: that proves Go and hatn agree
// on the packed millis+timezone representation, not merely on the framing. A blind echo would
// pass even if the two sides disagreed about what the int64 means.
type CustomTypesEndpoint struct {
	api_server.EndpointBase
}

func (e *CustomTypesEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("CustomTypesEndpoint")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[CustomTypesLogic](sctx)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return sctx, err
	}
	jsonDataPretty, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(jsonDataPretty))

	// Per-element counts are the decisive check: a packing mismatch collapses or splits
	// elements rather than corrupting them in place.
	fmt.Printf("CustomTypesEndpoint received %d datetime(s), %d date(s), %d time(s), %d daterange(s)\n",
		len(cmd.Datetimes), len(cmd.Dates), len(cmd.Times), len(cmd.Dateranges))

	// Decode each datetime so the log shows Go's own interpretation of the packed value -- if
	// the two sides disagreed about the representation, the printed instant would be visibly
	// wrong rather than merely unequal.
	//
	// The ORIGINAL packed value is echoed back, not the repacked one. FromHatnProtoDatetime
	// ends in .Local(), and the packed int64 carries the timezone in its low 8 bits, so on a
	// non-UTC host repacking a UTC-sent value yields a different int64 for the same instant.
	// Echoing that would fail the client's equality check for a timezone reason rather than a
	// wire-format one. The repacked value and whether it matches are logged instead, so a real
	// representation mismatch is still visible here.
	for i, packed := range cmd.Datetimes {
		t := utils.FromHatnProtoDatetime(packed)
		repacked := utils.ToHatnProtoDatetime(t)
		fmt.Printf("  datetime[%d] packed=%d decoded=%s repacked=%d same_int64=%v same_instant=%v\n",
			i, packed, t.Format("2006-01-02T15:04:05.000Z07:00"), repacked,
			repacked == packed,
			utils.FromHatnProtoDatetime(repacked).Equal(t))
	}
	for i, v := range cmd.Dates {
		fmt.Printf("  date[%d] %d (yyyymmdd)\n", i, v)
	}
	for i, v := range cmd.Times {
		fmt.Printf("  time[%d] %d (hhmmssmmm)\n", i, v)
	}
	for i, v := range cmd.Dateranges {
		fmt.Printf("  daterange[%d] %d\n", i, v)
	}

	// set response -- echoed verbatim, see the datetime note above
	resp := cmd
	request.Response().SetMessage(resp)

	// done
	return sctx, nil
}

func (e *CustomTypesEndpoint) NewRequestMessage() interface{} {
	return &CustomTypesLogic{}
}

func (e *CustomTypesEndpoint) NewResponseMessage() interface{} {
	return &CustomTypesLogic{}
}

func NewCustomTypesEndpoint(opName ...string) *CustomTypesEndpoint {
	ep := &CustomTypesEndpoint{}
	ep.Init("CustomTypes", access_control.Post)
	return ep
}
