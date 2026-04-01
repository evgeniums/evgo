package mux_api_server

import (
	"context"
	"crypto/tls"
	"log"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/api/api_server/grpc_api_server"
	"github.com/evgeniums/evgo/pkg/api/api_server/rest_api_gin_server"
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/background_worker"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/soheilhy/cmux"
)

type MuxApiServerConfig struct {
}

type MuxApiServer struct {
	api_server.Listener

	mux cmux.CMux

	grpcServer *grpc_api_server.Server
	httpServer *rest_api_gin_server.Server
}

func NewMuxServer(grpcExtender ...grpc_api_server.ServerExtender) *MuxApiServer {

	m := &MuxApiServer{
		grpcServer: grpc_api_server.NewServer(grpcExtender...),
		httpServer: rest_api_gin_server.NewServer(),
	}

	m.grpcServer.SetListener(&m.Listener)
	m.httpServer.SetListener(&m.Listener)

	return m
}

func (m *MuxApiServer) Init(ctx app_context.Context, auth auth.Auth, tenancyManager multitenancy.Multitenancy, parentPath string, configPath ...string) error {

	cfgPath := utils.OptionalString(object_config.Key(parentPath, "api_server"), configPath...)

	err := m.Listener.Init(ctx, cfgPath)
	if err != nil {
		return err
	}

	err = m.grpcServer.Init(ctx, auth, tenancyManager, object_config.Key(cfgPath, "grpc"))
	if err != nil {
		return err
	}

	err = m.httpServer.Init(ctx, auth, tenancyManager, object_config.Key(cfgPath, "http"))
	if err != nil {
		return err
	}

	return nil
}

func (m *MuxApiServer) Run(fin background_worker.Finisher) {

	m.Listener.Run()

	listener := m.Listener.Listener()

	if !m.Listener.DISABLE_TLS {
		cert, err := tls.LoadX509KeyPair(m.Listener.TLS_CERTIFICATE_FILE, m.Listener.TLS_PRIVATE_KEY_FILE)
		if err != nil {
			log.Fatalf("failed to load keys: %v", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2", "http/1.1"}, // Crucial for gRPC (h2) and HTTP/1.1 support
		}
		listener = tls.NewListener(listener, tlsConfig)
	}

	m.mux = cmux.New(listener)

	grpcL := m.mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpL := m.mux.Match(cmux.HTTP1Fast())

	m.grpcServer.Serve(grpcL)
	m.httpServer.Serve(httpL)

	fin.AddRunner(m)

	go m.mux.Serve()
}

func (m *MuxApiServer) Shutdown(ctx context.Context) error {

	m.App().Logger().Info("shutting down server listener...")

	m.mux.Close()

	m.grpcServer.Runner().Shutdown(ctx)
	m.httpServer.Runner().Shutdown(ctx)

	return nil
}
