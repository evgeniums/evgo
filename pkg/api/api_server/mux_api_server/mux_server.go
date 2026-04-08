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
	ENABLE_HTTP bool `default:"false"`
	ENABLE_GRPC bool `default:"true"`
}

type MuxApiServer struct {
	MuxApiServerConfig
	api_server.Listener

	mux cmux.CMux

	grpcServer *grpc_api_server.Server
	httpServer *rest_api_gin_server.Server

	grpcExtender *grpc_api_server.ServerExtender
}

func NewMuxServer(grpcExtender ...grpc_api_server.ServerExtender) *MuxApiServer {

	m := &MuxApiServer{}

	if len(grpcExtender) != 0 {
		m.grpcExtender = &grpcExtender[0]
	}

	return m
}

func (m *MuxApiServer) Config() any {
	return &m.MuxApiServerConfig
}

func (m *MuxApiServer) Init(ctx app_context.Context, auth auth.Auth, tenancyManager multitenancy.Multitenancy, parentPath string, configPath ...string) error {

	cfgPath := utils.OptionalString(object_config.Key(parentPath, "api_server"), configPath...)

	err := object_config.LoadLogValidate(ctx.Cfg(), ctx.Logger(), ctx.Validator(), m, cfgPath)
	if err != nil {
		return ctx.Logger().PushFatalStack("failed to load mux api server configuration", err)
	}

	err = m.Listener.Init(ctx, cfgPath)
	if err != nil {
		return err
	}

	if m.ENABLE_GRPC {

		if m.grpcExtender != nil {
			m.grpcServer = grpc_api_server.NewServer(*m.grpcExtender)
		}
		m.grpcServer.SetListener(&m.Listener)

		err = m.grpcServer.Init(ctx, auth, tenancyManager, object_config.Key(cfgPath, "grpc"))
		if err != nil {
			return err
		}
	}

	if m.ENABLE_HTTP {

		m.httpServer = rest_api_gin_server.NewServer()
		m.httpServer.SetListener(&m.Listener)

		err = m.httpServer.Init(ctx, auth, tenancyManager, object_config.Key(cfgPath, "http"))
		if err != nil {
			return err
		}
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

	if m.grpcServer != nil {
		grpcL := m.mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
		m.grpcServer.Serve(grpcL)
	}

	if m.httpServer != nil {
		httpL := m.mux.Match(cmux.HTTP1Fast())
		m.httpServer.Serve(httpL)
	}

	fin.AddRunner(m)

	go m.mux.Serve()
}

func (m *MuxApiServer) Shutdown(ctx context.Context) error {

	m.App().Logger().Info("shutting down server listener...")

	if m.httpServer != nil {
		m.httpServer.Runner().Shutdown(ctx)
	}

	if m.grpcServer != nil {
		m.grpcServer.Runner().Shutdown(ctx)
	}

	m.mux.Close()

	return nil
}

func (m *MuxApiServer) GrpcServer() *grpc_api_server.Server {
	return m.grpcServer
}

func (m *MuxApiServer) HttpServer() *rest_api_gin_server.Server {
	return m.httpServer
}
