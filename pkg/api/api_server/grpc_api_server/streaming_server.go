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
	"google.golang.org/protobuf/types/known/anypb"
)

type StreamingMessageType int

const (
	StreamingMessage      StreamingMessageType = 0
	StreamingError        StreamingMessageType = 1
	StreamingHeartBeat    StreamingMessageType = 2
	StreamingInitResponse StreamingMessageType = 3
)

func packResponse(input any) (*anypb.Any, string, error) {
	protoMsg, ok := input.(proto.Message)
	if !ok {
		return nil, "", fmt.Errorf("response is not a valid protobuf message")
	}
	anyMsg, err := anypb.New(protoMsg)
	if err != nil {
		return nil, "", err
	}
	return anyMsg, string(proto.MessageName(protoMsg)), nil
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

	err = stream.SendMsg(resp)
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

		c.Logger().Warn("failed to send message", logger.Fields{"response_type": msgType})

		return c.SetError(err)
	}

	return nil
}
