package grpc_api_server

import (
	"fmt"
	"io"

	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type StreamingMessageType int

const (
	StreamingMessage      StreamingMessageType = 0
	StreamingError        StreamingMessageType = 1
	StreamingInitResponse StreamingMessageType = 2
	StreamingHeartbeat    StreamingMessageType = 3
)

// StreamHeartbeatMessageType is the StreamResponse.MessageType used for application-level
// stream heartbeats (see ServerConfig.STREAM_HEARTBEAT_PERIOD). It is liveness-only: it
// carries no payload and must never be confused with a real proto message name.
const StreamHeartbeatMessageType = "hatn.stream.heartbeat"

func packResponse(input any) (proto.Message, string, error) {
	protoMsg, ok := input.(proto.Message)
	if !ok {
		return nil, "", fmt.Errorf("response is not a valid protobuf message")
	}
	return protoMsg, string(proto.MessageName(protoMsg)), nil
}

func SendStreamingResponse(request *Request, stream grpc.ServerStream, m any, msgType StreamingMessageType) error {

	if m == nil {
		return nil
	}

	c := request.TraceInMethod("Streaming.SendMsg")
	defer request.TraceOutMethod()

	msg, name, err := packResponse(m)
	if err != nil {
		c.SetMessage("failed to cast response")
		fillResponseStatus(request, err)
		return c.SetError(err)
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		c.SetMessage("failed to marshal response")
		fillResponseStatus(request, err)
		return c.SetError(err)
	}

	resp := &StreamResponse{
		Message:     data,
		MessageType: name,
	}

	if err := sendStreamMsg(request, stream, resp, msgType); err != nil {
		return c.SetError(err)
	}
	return nil
}

// SendStreamHeartbeat sends a zero-payload liveness message on a server stream. It is not
// traced like a real response (heartbeats fire on a timer independent of application
// events, so a trace record per heartbeat would be noise) but a failed send is treated the
// same as any other stream write failure: it terminates the handler, which is the desired
// early detection of a dead client.
func SendStreamHeartbeat(request *Request, stream grpc.ServerStream) error {
	resp := &StreamResponse{
		MessageType: StreamHeartbeatMessageType,
	}
	return sendStreamMsg(request, stream, resp, StreamingHeartbeat)
}

func sendStreamMsg(request *Request, stream grpc.ServerStream, resp *StreamResponse, msgType StreamingMessageType) error {

	err := stream.SendMsg(resp)
	if err != nil {

		st, ok := status.FromError(err)
		if ok {
			request.statusCode = st.Code()
			request.SetGenericErrorCode(GRPCToGeneric(st.Code()))
		} else {
			if err == io.EOF {
				request.statusCode = codes.Aborted
				request.SetGenericErrorCode(generic_error.ErrorCodeIOAborted)
			} else {
				request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			}
		}
		fillResponseStatus(request, err)

		request.Logger().Warn("failed to send message", logger.Fields{"response_type": msgType})

		return err
	}

	return nil
}
