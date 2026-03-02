package grpc_api_server

import (
	"context"
	"fmt"

	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/generic_error"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/status"
)

type RequestWrapper struct {
	request *Request
}

type RequestCodec struct {
	parent encoding.Codec
	server *Server
}

func (c *RequestCodec) Unmarshal(data []byte, v any) (err error) {

	defer func() {
		if r := recover(); r != nil {
			c.server.App().Logger().Fatal("application crashed", fmt.Errorf("panic triggered in RequestCodec.Unmarshal"))
			if w, ok := v.(*RequestWrapper); ok {
				request := w.request
				request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
				request.Close()
			}
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()

	if w, ok := v.(*RequestWrapper); ok {

		ep := w.request.Endpoint()

		// authenticate request
		w.request.Message().SetBinaryContent(data)
		err := w.request.server.Auth().HandleRequest(w.request, ep.Resource().ServicePathPrototype(), ep.AccessType())
		if err != nil {
			w.request.SetGenericErrorCode(auth.ErrorCodeUnauthorized)
			return err
		}
		w.request.Message().SetBinaryContent(nil)

		// copy payload if needed
		if ep.IsRequestPayloadNeeded() {
			payload := make([]byte, len(data))
			copy(payload, data)
			w.request.Message().SetBinaryContent(payload)
		}

		// create protobuf transport message
		pb := ep.NewTransportRequest(ep)
		if pb == nil {
			// empty message, decoding not needed
			return nil
		}

		// parse pb message
		w.request.Message().SetTransportMessage(pb)
		return c.parent.Unmarshal(data, w.request.Message().TransportMessage())
	}

	// fallback message decoding
	return c.parent.Unmarshal(data, v)
}

func (c *RequestCodec) Marshal(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	if r, ok := v.(api_server.RequestMessage); ok {
		if r.BinaryContent() != nil {
			return r.BinaryContent(), nil
		}
		if r.TransportMessage() == nil {
			return nil, nil
		}
		return c.parent.Marshal(r.TransportMessage())
	}
	return c.parent.Marshal(v)
}

func (c *RequestCodec) Name() string {
	return c.server.TRANSPORT_CODEC_TYPE
}

type UnaryHandler struct {
	endpoint            api_server.Endpoint
	server              *Server
	grpcUnaryServerInfo *grpc.UnaryServerInfo
}

func (u *UnaryHandler) handle(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {

	// create request
	request, callCtx, err := newRequest(ctx, u.server, u.endpoint)
	if err != nil {
		request.SetGenericErrorCode(generic_error.ErrorCodeFormat)
		u.server.logRequest(callCtx.Logger(), request.start, request)
		return nil, err
	}
	reqCtx := context.WithValue(ctx, RequestContextKey, request)

	// invoke decoder
	w := &RequestWrapper{request: request}
	if err := dec(w); err != nil {
		return nil, err
	}

	// define final handler
	finalHandler := func(ctx context.Context, transportRequest interface{}) (interface{}, error) {

		handle := func() (interface{}, error) {
			err = u.endpoint.HandleRequest(request)
			if err != nil {
				request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			}

			respMsg := &api_server.RequestMessageBase{}
			if request.Response().Payload() != nil {
				respMsg.SetBinaryContent(request.Response().Payload())
			} else if request.Response().Message() != nil {
				respMsg.SetLogicMessage(request.Response().Message())
				err := u.endpoint.LogicResponseToTransport(respMsg)
				if err != nil {
					callCtx.Logger().Error("failed to convert logic message to protobuf", err)
					request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
					return nil, err
				}
			}

			return respMsg, err
		}

		response, err := handle()
		if err != nil {
			callCtx.SetError(err)
		}
		request.TraceOutMethod()
		request.Close()

		return response, status.Error(request.statusCode, request.statusMessage)
	}

	// invoke interceptors if set
	if interceptor == nil {
		return finalHandler(reqCtx, request.Message().TransportMessage())
	}
	return interceptor(reqCtx, request.Message().TransportMessage(), u.grpcUnaryServerInfo, finalHandler)
}
