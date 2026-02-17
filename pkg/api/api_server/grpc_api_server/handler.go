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
	newProtoMessage     func() interface{}

	transportToLogic func(interface{}) RequestMessage
	logicToTransport func(interface{}) interface{}
}

func (u *UnaryHandler) SetTransportToLogicMessageMapper(mapper func(interface{}) RequestMessage) {
	u.transportToLogic = mapper
}

func (u *UnaryHandler) SetLogicToTransportMessageMapper(mapper func(interface{}) interface{}) {
	u.logicToTransport = mapper
}

func (u *UnaryHandler) SetTransportMessageBuilder(builder func() interface{}) {
	u.newProtoMessage = builder
}

func (u *UnaryHandler) handle(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {

	var request *Request

	var msg interface{}
	if u.newProtoMessage == nil {
		// TODO make message automatically
	} else {
		msg = u.newProtoMessage()
	}
	if msg != nil {
		if err := dec(msg); err != nil {
			return nil, err
		}
	}

	finalHandler := func(ctx context.Context, req interface{}) (interface{}, error) {

		var err error
		var callCtx op_context.CallContext

		handle := func() (interface{}, error) {
			if u.transportToLogic == nil {
				request.message = &RequestMessageBase{message: req}
			} else {
				request.message = u.transportToLogic(req)
			}

			err := u.endpoint.HandleRequest(request)
			if err != nil {
				request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			}

			var response interface{}
			if request.Response().Message() != nil {
				if u.logicToTransport == nil {
					response = request.Response().Message
				} else {
					response = u.logicToTransport(request.Response().Message)
				}
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
