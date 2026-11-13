package grpc_api_server

import (
	"context"
	"net"
	"net/netip"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/realip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func addressContext(peerAddr string, headers ...string) context.Context {
	ctx := context.Background()
	if peerAddr != "" {
		addr, _ := net.ResolveTCPAddr("tcp", peerAddr)
		ctx = peer.NewContext(ctx, &peer.Peer{Addr: addr})
	}
	if len(headers) != 0 {
		ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(headers...))
	}
	return ctx
}

func TestClientAddress(t *testing.T) {

	proxies := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	server := &Server{}
	server.realIpInterceptor = realip.UnaryServerInterceptor(proxies, []string{realip.XForwardedFor})

	cases := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{"direct peer, no port", addressContext("203.0.113.7:51234"), "203.0.113.7"},
		{"direct peer ignores the header", addressContext("203.0.113.7:51234", "x-forwarded-for", "198.51.100.1"), "203.0.113.7"},
		{"trusted proxy, header address", addressContext("10.1.2.3:4000", "x-forwarded-for", "198.51.100.1"), "198.51.100.1"},
		{"trusted proxy, no header", addressContext("10.1.2.3:4000"), "10.1.2.3"},
		{"trusted proxy, rightmost entry", addressContext("10.1.2.3:4000", "x-forwarded-for", "192.0.2.9, 198.51.100.1"), "198.51.100.1"},
		{"ipv6 peer", addressContext("[2001:db8::1]:443"), "2001:db8::1"},
		{"no peer", context.Background(), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := server.clientAddress(tc.ctx); got != tc.want {
				t.Fatalf("clientAddress = %q, want %q", got, tc.want)
			}
		})
	}

	// a stream's context already carries the resolved address
	streamCtx := context.WithValue(addressContext("10.1.2.3:4000"), struct{}{}, nil)
	_, _ = server.realIpInterceptor(addressContext("10.1.2.3:4000", "x-forwarded-for", "198.51.100.1"), nil, nil,
		func(resolved context.Context, _ any) (any, error) {
			streamCtx = resolved
			return nil, nil
		})
	if got := server.clientAddress(streamCtx); got != "198.51.100.1" {
		t.Fatalf("clientAddress of a resolved context = %q", got)
	}

	// without a configured interceptor the peer address is used
	if got := (&Server{}).clientAddress(addressContext("203.0.113.7:51234")); got != "203.0.113.7" {
		t.Fatalf("clientAddress without interceptor = %q", got)
	}
}
