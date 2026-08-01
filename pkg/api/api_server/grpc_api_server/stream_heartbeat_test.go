package grpc_api_server

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

// streamHeartbeatPeriod only reads u.server.STREAM_HEARTBEAT_PERIOD/HEADER and the incoming
// metadata carried by request.sctx, so it can be exercised directly without a live server,
// auth, or a real gRPC transport.
func TestStreamHeartbeatPeriod(t *testing.T) {

	newRequestWithHeader := func(header, value string) *Request {
		ctx := context.Background()
		if header != "" && value != "" {
			ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(header, value))
		}
		return &Request{sctx: ctx}
	}

	tests := []struct {
		name         string
		serverPeriod int
		header       string
		hintHeader   string
		hintValue    string
		want         int
	}{
		{
			name:         "disabled on server",
			serverPeriod: 0,
			header:       "x-hatn-stream-hb",
			hintHeader:   "x-hatn-stream-hb",
			hintValue:    "5",
			want:         0,
		},
		{
			name:         "client did not opt in",
			serverPeriod: 20,
			header:       "x-hatn-stream-hb",
			hintHeader:   "",
			hintValue:    "",
			want:         0,
		},
		{
			name:         "client hint smaller than server period is honored",
			serverPeriod: 20,
			header:       "x-hatn-stream-hb",
			hintHeader:   "x-hatn-stream-hb",
			hintValue:    "10",
			want:         10,
		},
		{
			name:         "client hint larger than server period is ignored, server period wins",
			serverPeriod: 20,
			header:       "x-hatn-stream-hb",
			hintHeader:   "x-hatn-stream-hb",
			hintValue:    "60",
			want:         20,
		},
		{
			name:         "client hint below the 5s floor is clamped up",
			serverPeriod: 20,
			header:       "x-hatn-stream-hb",
			hintHeader:   "x-hatn-stream-hb",
			hintValue:    "1",
			want:         5,
		},
		{
			name:         "unparseable hint falls back to server period",
			serverPeriod: 20,
			header:       "x-hatn-stream-hb",
			hintHeader:   "x-hatn-stream-hb",
			hintValue:    "not-a-number",
			want:         20,
		},
		{
			name:         "server period itself below the floor is still clamped up",
			serverPeriod: 2,
			header:       "x-hatn-stream-hb",
			hintHeader:   "x-hatn-stream-hb",
			hintValue:    "2",
			want:         5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{}
			s.STREAM_HEARTBEAT_PERIOD = tt.serverPeriod
			s.STREAM_HEARTBEAT_HEADER = tt.header

			h := &Handler{server: s}
			req := newRequestWithHeader(tt.hintHeader, tt.hintValue)

			got := h.streamHeartbeatPeriod(req)
			if got != tt.want {
				t.Errorf("streamHeartbeatPeriod() = %d, want %d", got, tt.want)
			}
		})
	}
}
