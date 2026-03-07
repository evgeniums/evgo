package gatewayapi

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/http_request"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/sms"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

const Protocol string = "gatewayapi"

type Recipient struct {
	Msisdn string `json:"msisdn"`
}

type Message struct {
	Message    string      `json:"message"`
	Recipients []Recipient `json:"recipients"`
	Usereref   string      `json:"userref"`
	Sender     string      `json:"sender"`
}

type MessageWithSender struct {
	Message
	Sender string `json:"sender"`
}

type GoodResponse struct {
	Ids []int `json:"ids"`
}

type SmsGatewayapiConfig struct {
	sms.ProviderBase
	URL    string `validate:"required,url"`
	TOKEN  string `validate:"required" mask:"true"`
	SENDER string
}

type SmsGatewayapi struct {
	SmsGatewayapiConfig
	sendUrl string

	http_request.WithHttpClient
}

func New() *SmsGatewayapi {
	return &SmsGatewayapi{}
}

func (s *SmsGatewayapi) Config() interface{} {
	return &s.SmsGatewayapiConfig
}

func (s *SmsGatewayapi) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	s.WithHttpClient.Construct()

	cfgPath := utils.OptionalString("sms.gatewayapi", configPath...)
	err := object_config.LoadLogValidate(cfg, log, vld, s, cfgPath)
	if err != nil {
		return log.PushFatalStack("failed to init SmsGatewayapi", err)
	}

	err = s.WithHttpClient.Init(cfg, log, vld, cfgPath)
	if err != nil {
		return log.PushFatalStack("failed to init http client in SmsGatewayapi", err)
	}

	s.ProviderBase.SetProtocolAndName(Protocol, utils.OptionalString(Protocol, s.NAME))

	s.sendUrl = fmt.Sprintf("%s/rest/mtsms?token=%s", s.URL, s.TOKEN)
	return nil
}

func (s *SmsGatewayapi) Send(ctx op_context.Context, message string, recipient string, smsID ...string) (*sms.ProviderResponse, error) {

	c := ctx.TraceInMethod("SmsGatewayapi.Send", logger.Fields{"recipient": recipient})
	var err error
	onExit := func() {
		if err != nil {
			ctx.SetGenericErrorCode(sms.ErrorCodeSmsSendingFailed)
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	recipients := []Recipient{{Msisdn: recipient}}
	msg := Message{Message: message, Recipients: recipients}
	if len(smsID) > 0 {
		msg.Usereref = smsID[0]
		c.LoggerFields()["sms_id"] = msg.Usereref
	}

	var obj interface{}
	if s.SENDER != "" {
		obj = &MessageWithSender{Message: msg, Sender: s.SENDER}
	} else {
		obj = &msg
	}

	request, err := s.HttpClient().NewPost(ctx, s.sendUrl, obj)
	if err != nil {
		return nil, err
	}

	response := &GoodResponse{}
	request.GoodResponse = response

	err = request.Send(ctx)
	c.LoggerFields()["response_content"] = request.ResponseContent
	c.LoggerFields()["response_status"] = request.ResponseStatus
	if err != nil {
		return nil, err
	}

	result := &sms.ProviderResponse{RawContent: request.ResponseContent}
	if len(response.Ids) > 0 {
		result.ProviderMessageID = fmt.Sprintf("%d", response.Ids[0])
		ctx.SetLoggerField("provider_sms_id", result.ProviderMessageID)
	}

	return result, nil
}
