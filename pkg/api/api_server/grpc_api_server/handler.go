package grpc_api_server

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/mem"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type RequestWrapper struct {
	request *Request
}

type RequestCodec struct {
	parent encoding.CodecV2
	server *Server
}

func (c *RequestCodec) Unmarshal(data mem.BufferSlice, v any) (err error) {

	if w, ok := v.(*RequestWrapper); ok {

		callCtx := w.request.TraceInMethod("Unmarshal")
		var err error
		defer func() {
			if r := recover(); r != nil {

				const size = 64 << 10 // 64KB
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				c.server.App().Logger().Fatal("application crashed in RequestCodec.Unmarshal", fmt.Errorf("Stack Trace:\n%s\n", buf))

				if w, ok := v.(*RequestWrapper); ok {
					request := w.request
					request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
				}
				err = status.Errorf(codes.Internal, "internal server error")
			} else {
				if err != nil {
					callCtx.SetError(err)
				}
				w.request.TraceOutMethod()
			}
		}()

		ep := w.request.Endpoint()
		w.request.payloadSize = len(data)

		// authenticate request

		// prepare buffer with content payload
		var materializedBuf *[]byte
		if len(data) == 1 {
			w.request.Message().SetBinaryContent(data[0].ReadOnlyData())
		} else {
			temp := data.Materialize()
			materializedBuf = &temp
			w.request.Message().SetBinaryContent(*materializedBuf)
		}

		// preprocess request
		w.request.sctx, err = ep.PreprocessBeforeAuth(w.request.sctx)
		if err != nil {
			callCtx.SetMessage("preprocess failed")
			return err
		}

		// dump headers
		if w.request.server.DUMP_HEADERS {
			fmt.Println("=======Dumping gRPC request headers before auth=======")
			for key, values := range w.request.requestMetadata() {
				for _, value := range values {
					fmt.Printf("%s: %s\n", key, value)
				}
			}
			fmt.Println("==========================================")
		}

		// perform auth
		err = w.request.server.Auth().HandleRequest(w.request.sctx, ep.Resource().ServicePathPrototype(), ep.AccessType())
		if err != nil {
			w.request.Message().SetBinaryContent(nil)
			if materializedBuf != nil {
				mem.DefaultBufferPool().Put(materializedBuf)
			}
			w.request.SetGenericErrorCode(auth.ErrorCodeUnauthorized)
			return err
		}

		// cleanup or copy payload if needed
		if ep.IsRequestPayloadNeeded() {
			if materializedBuf == nil {
				// We didn't materialize before, but we need it now.
				temp := data.Materialize()
				materializedBuf = &temp
				w.request.Message().SetBinaryContent(*materializedBuf)
			}
		} else {
			w.request.Message().SetBinaryContent(nil)
			if materializedBuf != nil {
				mem.DefaultBufferPool().Put(materializedBuf)
			}
		}

		// create protobuf transport message
		pb := ep.NewTransportRequest(ep)
		if pb == nil {
			// empty message, decoding not needed
			return nil
		}

		// parse pb message
		w.request.Message().SetTransportMessage(pb)
		err = c.parent.Unmarshal(data, pb)
		if err != nil {
			gerr := generic_error.NewFromErr(err, generic_error.ErrorCodeFormat)
			w.request.SetGenericError(gerr)
			return gerr
		}
		return nil
	}

	// fallback message decoding
	return c.parent.Unmarshal(data, v)
}

func (c *RequestCodec) Marshal(v any) (mem.BufferSlice, error) {
	if v == nil {
		return nil, nil
	}
	if r, ok := v.(api_server.RequestMessage); ok {
		if r.BinaryContent() != nil {
			content := r.BinaryContent()
			newBuffer := mem.NewBuffer(&content, mem.DefaultBufferPool())
			newSlice := mem.BufferSlice{newBuffer}
			r.SetBinaryContent(nil)
			return newSlice, nil
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

type Handler struct {
	endpoint            api_server.Endpoint
	server              *Server
	grpcUnaryServerInfo *grpc.UnaryServerInfo
}

func GetProtoName(i interface{}) string {
	if msg, ok := i.(proto.Message); ok {
		return string(proto.MessageName(msg))
	}
	return ""
}

func (u *Handler) handleUnary(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {

	// create request
	request, callCtx, err := newRequest(ctx, u.server, u.endpoint)
	if err != nil {
		resp := u.fillResponse(request, callCtx)
		return resp, status.Error(request.statusCode, request.statusMessage)
	}

	// invoke decoder
	w := &RequestWrapper{request: request}
	if err := dec(w); err != nil {
		resp := u.fillResponse(request, callCtx)
		st := status.Error(request.statusCode, request.statusMessage)
		return resp, st
	}

	// invoke interceptors if set
	if interceptor == nil {
		return u.handleRequest(request.sctx, request.Message().TransportMessage())
	}
	return interceptor(request.sctx, request.Message().TransportMessage(), u.grpcUnaryServerInfo, u.handleRequest)
}

type SizeInfo struct {
	value int
}
type sizeStatsHandler struct {
	stats.Handler
}

func (h *sizeStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return context.WithValue(ctx, HeaderSizeKey, &SizeInfo{})
}

func (h *sizeStatsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	if inHeader, ok := s.(*stats.InHeader); ok {
		if info, ok := ctx.Value(HeaderSizeKey).(*SizeInfo); ok {
			info.value = inHeader.WireLength
		}
	}
}

func (h *sizeStatsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}
func (h *sizeStatsHandler) HandleConn(ctx context.Context, s stats.ConnStats) {}

func (u *Handler) handleServerStream(srv interface{}, stream grpc.ServerStream) error {

	ctx := stream.Context()

	// create request
	request, callCtx, err := newRequest(ctx, u.server, u.endpoint)
	if err != nil {
		resp := u.fillResponse(request, callCtx)
		if resp != nil && resp.TransportMessage() != nil {
			err1 := stream.SendMsg(resp.TransportMessage())
			if err1 != nil {
				callCtx.Logger().Warn("failed to send error response", logger.Fields{"send_err": err1})
			}
		}
		err = status.Error(request.statusCode, request.statusMessage)
		if request != nil {
			if callCtx != nil {
				callCtx.SetError(err)
			}
			request.Close(request.sctx)
		}
	}

	defer func() {
		if err != nil {
			callCtx.SetError(err)
		}
		request.TraceOutMethod()
		request.Close(request.sctx)
	}()

	// receive initial message
	w := &RequestWrapper{request: request}
	if err1 := stream.RecvMsg(w); err1 != nil {
		resp := u.fillResponse(request, callCtx)
		if resp != nil && resp.TransportMessage() != nil {
			SendStreamingResponse(request, stream, resp.TransportMessage(), StreamingError)
		}
		err = err1
		return status.Error(request.statusCode, request.statusMessage)
	}

	// process initial message and send response
	resp, err1 := u.handleRequest(request.sctx, request.Message().TransportMessage())
	sctx := request.sctx
	// extract message queue from context here to avoid memory leaks in case of handler errors
	mq := message_queue.MqContext(ctx)
	if mq != nil {
		defer mq.Unsubscribe(ctx)
	}
	if resp != nil {
		respType := StreamingInitResponse
		if err1 != nil {
			respType = StreamingError
		}
		err2 := SendStreamingResponse(request, stream, resp, respType)
		if err1 != nil {
			err = err1
			return err1
		}
		if err2 != nil {
			err = err2
			return err2
		}
	}

	if mq == nil {
		// skip streaming
		return nil
	}

	request.OnStreamIntialized(sctx, "queue opened")

	// init heartbeat ticker
	ticker := time.NewTicker(time.Duration(request.server.HEARTBEAT_PERIOD) * time.Second)
	defer ticker.Stop()

	for {
		select {
		// SIGNAL 1: Client disconnected or timeout
		case <-sctx.Done():
			err = stream.Context().Err()
			callCtx.Logger().Warn("unexpectedly closed stream", logger.Fields{"stream_err": err})
			return err

		// SIGNAL 2: Global server shutdown
		case <-request.server.shutdown:
			return status.Error(codes.Unavailable, "server is shutting down")

		// SIGNAL 3: Heartbeat to keep connection alive
		case <-ticker.C:
			heartBeat := &HeartBeat{Timestamp: utils.ToHatnProtoDatetime(time.Now())}
			err = SendStreamingResponse(request, stream, heartBeat, StreamingHeartBeat)
			if err != nil {
				return err
			}

		// SIGNAL 4: Data from mq (using 'ok' to detect closure)
		case message, ok := <-mq.Channel():
			if !ok {
				callCtx.SetMessage("queue closed")
				return nil
			}

			msgContent := api_server.NewMessageContent()
			msgContent.SetLogicMessage(message)
			err = u.endpoint.LogicResponseToTransport(msgContent)
			if err != nil {
				callCtx.SetMessage("failed convert logic to transport")
				return status.Error(codes.Internal, "internal data error")
			}

			err = SendStreamingResponse(request, stream, msgContent.TransportMessage(), StreamingMessage)
			if err != nil {
				return err
			}

			mq.Next()
		}
	}
}

func (u *Handler) handleRequest(sctx context.Context, transportRequest interface{}) (interface{}, error) {

	request := op_context.OpContext[*Request](sctx)
	request.sctx = sctx

	c := request.TraceInMethod("handleRequest")
	defer request.TraceOutMethod()

	// dump headers
	if u.server.DUMP_HEADERS {
		fmt.Println("=======Dumping gRPC request header in final headers=======")
		for key, values := range request.requestMetadata() {
			for _, value := range values {
				fmt.Printf("%s: %s\n", key, value)
			}
		}
		fmt.Println("==========================================")
	}

	newCtx, err := u.endpoint.HandleRequest(request.sctx)
	if err != nil {
		request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		c.SetError(err)
	}
	request.sctx = newCtx

	request.sctx, err = u.endpoint.Postprocess(request.sctx)
	if err != nil {
		request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		c.SetError(err)
	}

	response := u.fillResponse(request, c)
	return response, status.Error(request.statusCode, request.statusMessage)
}

func (u *Handler) fillResponse(request *Request, callCtx op_context.CallContext) api_server.RequestMessage {

	response := &api_server.RequestMessageBase{}
	if request.Response().Payload() != nil {
		response.SetBinaryContent(request.Response().Payload())
	} else if request.Response().Message() != nil {
		response.SetLogicMessage(request.Response().Message())
		err := u.endpoint.LogicResponseToTransport(response)
		if err != nil {
			callCtx.Logger().Error("failed to convert logic message to protobuf", err)
			request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		}
	}

	// fill response headers
	// TODO put message ID to header
	md := metadata.Pairs()

	var appStatus string
	if request.GenericError() == nil {
		appStatus = "success"
		request.SetLoggerField("status", "success")
	} else {
		code, err := request.server.MakeResponseError(request.GenericError())
		if code < http.StatusInternalServerError {
			request.SetErrorAsWarn(true)
		}
		request.statusCode = HTTPToGRPC(code)
		request.statusMessage = request.GenericError().Message()
		appStatus = err.Code()
		errMsg := err.Message()
		if errMsg == "" {
			errMsg = request.statusMessage
		}
		if errMsg != "" {
			md.Append(u.server.ERROR_DESCRIPTION_HEADER, errMsg)
		}
		errDetails := err.Details()
		if errDetails != "" {
			md.Append(u.server.ERROR_DETAILS_HEADER, errDetails)
		}
		errFamily := err.Family()
		if errFamily != "" {
			md.Append(u.server.ERROR_FAMILY_HEADER, errFamily)
		}

		if err.Data() != nil {
			// TODO convert error data to protobuf and put to response message
		}
	}

	md.Append(u.server.STATUS_HEADER, appStatus)
	if response.TransportMessage() != nil {
		md.Append(u.server.MESSAGE_TYPE_HEADER, GetProtoName(response.TransportMessage()))
	}
	if err := grpc.SetHeader(request.sctx, md); err != nil {
		callCtx.Logger().Error("failed to set response headers", err)
	}

	// close request
	request.TraceOutMethod()
	request.Close(request.sctx)

	// done
	return response
}
