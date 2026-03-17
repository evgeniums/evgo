package grpc_api_server

import (
	"errors"
	"sync"

	"google.golang.org/protobuf/proto"
)

type LogicToProto = func(from any) (proto.Message, error)

type LogicToProtoRegistry interface {
	Convert(name string, msg any) (proto.Message, error)
	Register(name string, converter LogicToProto)

	Marshal(name string, msg any) ([]byte, error)
}

type logicToProtoRegistry struct {
	registry map[string]LogicToProto
}

var (
	logicToProtoRegistryInst LogicToProtoRegistry
	logicToProtoOnce         sync.Once
)

func SetLogicToProtoRegistry(registry LogicToProtoRegistry) {
	logicToProtoRegistryInst = registry
}

func NewLogicToProtoRegistry() *logicToProtoRegistry {
	return &logicToProtoRegistry{
		registry: make(map[string]LogicToProto),
	}
}

func GetLogicToProtoRegistry() LogicToProtoRegistry {
	logicToProtoOnce.Do(func() {
		if logicToProtoRegistryInst == nil {
			logicToProtoRegistryInst = NewLogicToProtoRegistry()
		}
	})
	return logicToProtoRegistryInst
}

func (r *logicToProtoRegistry) Convert(name string, msg any) (proto.Message, error) {
	conv, ok := r.registry[name]
	if !ok {
		return nil, errors.New("proto converter not registered")
	}
	return conv(msg)
}

func (r *logicToProtoRegistry) Marshal(name string, msg any) ([]byte, error) {
	protoMsg, err := r.Convert(name, msg)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(protoMsg)
}

func (r *logicToProtoRegistry) Register(name string, converter LogicToProto) {
	r.registry[name] = converter
}

func ConvertLogicToProto(name string, msg any) (proto.Message, error) {
	return GetLogicToProtoRegistry().Convert(name, msg)
}

func MarshalLogicToProto(name string, msg any) ([]byte, error) {
	return GetLogicToProtoRegistry().Marshal(name, msg)
}

func RegisterLogicToProto(name string, converter LogicToProto) {
	GetLogicToProtoRegistry().Register(name, converter)
}
