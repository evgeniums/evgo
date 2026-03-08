package smsru

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/http_request"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/sms"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

const Protocol string = "smsru"

const (
	CodeOk                   = 100
	CodeInvalidRoute         = 150
	CodeInvalidPhone         = 202
	CodeForeignPhone         = 214
	CodePhoneDayLimit        = 230
	CodePhoneSameMinuteLimit = 231
	CodePhoneSameDayLimit    = 232
	CodePhoneCodeSpamLimit   = 233
)

type SmsruConfig struct {
	sms.ProviderBase
	URL    string `validate:"required,url"`
	API_ID string `validate:"required" mask:"true"`
	TEST   int
}

type Smsru struct {
	SmsruConfig
	http_request.WithHttpClient
}

func New() *Smsru {
	return &Smsru{}
}

func (s *Smsru) Config() interface{} {
	return &s.SmsruConfig
}

type request struct {
	ApiId   string `json:"api_id"`
	To      string `json:"to"`
	Message string `json:"msg"`
	Json    int    `json:"json"`
	Test    int    `json:"test"`
}

type responseItem struct {
	Status       string `json:"status"`
	StatusCode   int    `json:"status_code"`
	SmsId        string `json:"sms_id"`
	ErrorMessage string `json:"status_text"`
}

type response struct {
	Status     string                  `json:"status"`
	StatusCode int                     `json:"status_code"`
	Balance    float64                 `json:"balance"`
	Items      map[string]responseItem `json:"sms"`
}

func (s *Smsru) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	cfgPath := utils.OptionalString("sms.smsru", configPath...)
	err := object_config.LoadLogValidate(cfg, log, vld, s, cfgPath)
	if err != nil {
		return log.PushFatalStack("failed to init SmsGatewayapi", err)
	}

	s.WithHttpClient.Construct()
	err = s.WithHttpClient.Init(cfg, log, vld, cfgPath)
	if err != nil {
		return log.PushFatalStack("failed to init http client in SmsGatewayapi", err)
	}

	s.ProviderBase.SetProtocolAndName(Protocol, utils.OptionalString(Protocol, s.NAME))
	return nil
}

func (s *Smsru) Send(sctx context.Context, message string, recipient string, smsID ...string) (*sms.ProviderResponse, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("Smsru.Send", logger.Fields{"recipient": recipient})
	var err error
	onExit := func() {
		if err != nil {
			ctx.SetGenericErrorCode(sms.ErrorCodeSmsSendingFailed)
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// prepare request
	smsReq := &request{ApiId: s.API_ID, To: recipient, Message: message, Json: 1, Test: s.TEST}
	url := s.URL + "/send"
	req, err := s.HttpClient().NewGet(sctx, url, smsReq)
	if err != nil {
		return nil, err
	}
	resp := &response{}
	req.GoodResponse = resp
	req.BadResponse = resp

	// send request
	err = req.Send(sctx)
	c.LoggerFields()["response_content"] = req.ResponseContent
	c.LoggerFields()["response_status"] = req.ResponseStatus
	c.LoggerFields()["status_code"] = resp.StatusCode
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != CodeOk {
		err = errors.New("failed status code")
	}

	// fill result
	result := &sms.ProviderResponse{RawContent: req.ResponseContent}
	item, ok := resp.Items[recipient]
	if ok {
		result.ProviderMessageID = item.SmsId
		ctx.SetLoggerField("provider_sms_id", result.ProviderMessageID)
		c.LoggerFields()["sms_status_code"] = item.StatusCode
		if err == nil && item.StatusCode != CodeOk {
			err = errors.New("failed item status code")
		}
	} else {
		if err == nil {
			err = errors.New("phone not found in response")
		} else {
			c.Logger().Warn("phone not found in response")
		}
	}

	// return result
	return result, err
}
