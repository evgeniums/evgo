package grpc_api_server

import (
	"context"

	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/op_context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type UnaryHandler struct {
	endpoint            api_server.Endpoint
	server              *Server
	grpcUnaryServerInfo *grpc.UnaryServerInfo
}

func (u *UnaryHandler) handle(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {

	var request *Request

	msg := u.endpoint.NewTransportMessage()
	if msg != nil {
		if err := dec(msg); err != nil {
			return nil, err
		}
	}

	finalHandler := func(ctx context.Context, req interface{}) (interface{}, error) {

		var err error
		var callCtx op_context.CallContext

		handle := func() (interface{}, error) {
			request.message = u.endpoint.TransportMessageToLogic(req)

			err := u.endpoint.HandleRequest(request)
			if err != nil {
				request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			}

			var response interface{}
			if request.Response().Message() != nil {
				response = u.endpoint.LogicMessageToTransport(request.Response().Message())
			}

			return response, err
		}

		var response interface{}
		request, callCtx, err = newRequest(ctx, u.server, u.endpoint)
		if err == nil {
			handle()
		}

		if err != nil {
			callCtx.SetError(err)
		}
		request.TraceOutMethod()
		request.Close()

		return response, status.Error(request.statusCode, request.statusMessage)
	}

	if interceptor == nil {
		return finalHandler(ctx, msg)
	}

	return interceptor(ctx, msg, u.grpcUnaryServerInfo, finalHandler)
}
