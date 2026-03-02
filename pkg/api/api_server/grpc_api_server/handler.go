package grpc_api_server

import (
	"context"

	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/generic_error"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type UnaryHandler struct {
	endpoint            api_server.Endpoint
	server              *Server
	grpcUnaryServerInfo *grpc.UnaryServerInfo
}

func (u *UnaryHandler) handle(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {

	request, callCtx, err := newRequest(ctx, u.server, u.endpoint)
	if err != nil {
		request.SetGenericErrorCode(generic_error.ErrorCodeFormat)
		u.server.logRequest(callCtx.Logger(), request.start, request)
		return nil, err
	}
	nextCtx := context.WithValue(ctx, RequestContextKey, request)

	transportRequest := u.endpoint.NewTransportRequest(u.endpoint)
	if transportRequest != nil {
		if err := dec(transportRequest); err != nil {
			return nil, err
		}
	}

	finalHandler := func(ctx context.Context, transportRequest interface{}) (interface{}, error) {

		handle := func() (interface{}, error) {

			err := u.endpoint.TransportRequestToLogic(request.message)
			if err != nil {
				// TODO fill error
				return nil, err
			}

			err = u.endpoint.HandleRequest(request)
			if err != nil {
				request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			}

			respMsg := &api_server.RequestMessageBase{}
			if request.Response().Message() != nil {
				respMsg.SetLogicMessage(request.Response().Message())
				err := u.endpoint.LogicResponseToTransport(respMsg)
				if err != nil {
					//! TODO set internal server error
					return nil, err
				}
			}

			return respMsg.TransportMessage(), err
		}

		var response interface{}
		if err == nil {
			request.message = &api_server.RequestMessageBase{}
			request.message.SetTransportMessage(transportRequest)
			err = u.endpoint.TransportRequestToLogic(request.message)
			if err == nil {
				response, err = handle()
			}
		}

		if err != nil {
			callCtx.SetError(err)
		}
		request.TraceOutMethod()
		request.Close()

		return response, status.Error(request.statusCode, request.statusMessage)
	}

	if interceptor == nil {
		return finalHandler(nextCtx, transportRequest)
	}

	return interceptor(nextCtx, transportRequest, u.grpcUnaryServerInfo, finalHandler)
}
