package sms_provider_factory

import (
	"errors"

	"github.com/evgeniums/evgo/pkg/sms"
	"github.com/evgeniums/evgo/pkg/sms/providers/gatewayapi"
	"github.com/evgeniums/evgo/pkg/sms/providers/sms_mock"
	"github.com/evgeniums/evgo/pkg/sms/providers/smsru"
)

type Builder func() sms.Provider

type DefaultFactory struct {
	builders map[string]Builder
}

func NewDefaultFactory() *DefaultFactory {
	f := &DefaultFactory{}
	f.builders = make(map[string]Builder)
	f.AddBuilder(gatewayapi.Protocol, func() sms.Provider { return gatewayapi.New() })
	f.AddBuilder(smsru.Protocol, func() sms.Provider { return smsru.New() })
	f.AddBuilder(sms_mock.Protocol, func() sms.Provider { return sms_mock.New() })
	return f
}

func (f *DefaultFactory) AddBuilder(protocol string, builder Builder) {
	f.builders[protocol] = builder
}

func (f *DefaultFactory) Create(protocol string) (sms.Provider, error) {

	builder, ok := f.builders[protocol]
	if !ok {
		return nil, errors.New("unknown SMS provider")
	}

	return builder(), nil
}

type MockFactory struct{}

func (f *MockFactory) Create(protocol string) (sms.Provider, error) {

	switch protocol {
	case sms_mock.Protocol:
		return sms_mock.New(), nil
	}

	return nil, errors.New("unknown SMS provider")
}
