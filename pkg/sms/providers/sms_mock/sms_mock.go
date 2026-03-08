package sms_mock

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/sms"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

const Protocol string = "sms_mock"

type SmsMockConfig struct {
	sms.ProviderBase
	ALWAYS_FAIL bool
}

type SmsMock struct {
	SmsMockConfig
}

func New() *SmsMock {
	return &SmsMock{}
}

func (s *SmsMock) Config() interface{} {
	return &s.SmsMockConfig
}

func (s *SmsMock) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	err := object_config.LoadLogValidate(cfg, log, vld, s, "sms.mock", configPath...)
	if err != nil {
		return log.PushFatalStack("failed to init SmsMock", err)
	}

	s.ProviderBase.SetProtocolAndName(Protocol, utils.OptionalString(Protocol, s.NAME))
	return nil
}

func (s *SmsMock) Send(sctx context.Context, message string, recipient string, smsID ...string) (*sms.ProviderResponse, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("SmsMock.Send", logger.Fields{"recipient": recipient})
	var err error
	onExit := func() {
		if err != nil {
			ctx.SetGenericErrorCode(sms.ErrorCodeSmsSendingFailed)
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// fill result
	result := &sms.ProviderResponse{}
	result.ProviderMessageID = utils.GenerateID()
	if s.ALWAYS_FAIL {
		result.RawContent = "failed"
		err = errors.New("expected failure")
	} else {
		result.RawContent = "ok"
		c.LoggerFields()["provider_sms_id"] = result.ProviderMessageID
	}

	c.Logger().Info("Send SMS", logger.Fields{"message": message})

	// return result
	return result, err
}
