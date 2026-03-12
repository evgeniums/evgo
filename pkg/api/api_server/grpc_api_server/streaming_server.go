package grpc_api_server

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/logger"
	"google.golang.org/grpc"
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
		return c.SetError(err)
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		c.SetMessage("failed to marshal response")
		return c.SetError(err)
	}

	resp := &StreamResponse{
		Message:     data,
		MessageType: name,
	}

	err = stream.SendMsg(resp)
	if err != nil {
		c.Logger().Warn("failed to send message", logger.Fields{"response_type": msgType})
		return c.SetError(err)
	}

	return nil
}
