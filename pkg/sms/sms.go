package sms

import (
	"errors"

	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

type ProviderResponse struct {
	ProviderMessageID string
	RawContent        string
}

type Provider interface {
	object_config.Subobject
	Send(ctx op_context.Context, message string, recipient string, smsID ...string) (*ProviderResponse, error)
}

type ProviderBase struct {
	object_config.WithProtocolBase
	common.WithNameBase
}

func (p *ProviderBase) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {
	return errors.New("incomplete provider")
}

func (p *ProviderBase) SetProtocolAndName(protocol string, name ...string) {
	p.PROTOCOL = protocol
	p.NAME = utils.OptionalArg(protocol, name...)
}

type ProviderFactory interface {
	Create(provider string) (Provider, error)
}
