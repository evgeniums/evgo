package sms

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/crypt_utils"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

type SmsManager interface {
	generic_error.ErrorDefinitions

	Send(sctx context.Context, message string, recipient string) (string, error)
	FindSms(sctx context.Context, smsId string) (*SmsMessage, error)
}

const (
	ErrorCodeSmsSendingFailed string = "sms_sending_failed"
)

var SmsErrorDescriptions = map[string]string{
	ErrorCodeSmsSendingFailed: "Failed to send SMS",
}

var SmsErrorHttpCodes = map[string]int{
	ErrorCodeSmsSendingFailed: http.StatusInternalServerError,
}

const (
	StatusSending string = "sending"
	StatusSuccess string = "success"
	StatusFail    string = "fail"
)

type SmsMessage struct {
	common.ObjectWithMonthBase
	auth.WithUserBase
	Context     string `gorm:"index;index:,unique,composite:u_month"`
	ForeignId   string `gorm:"index"`
	Phone       string `gorm:"index"`
	Operation   string `gorm:"index"`
	Provider    string `gorm:"index"`
	Status      string `gorm:"index"`
	Tenancy     string `gorm:"index"`
	Message     string
	RawResponse string
}

type SmsManagerBaseConfig struct {
	DEFAULT_PROVIDER      string `validate:"required"`
	ENCRYPT_MESSAGE_STORE bool
	SECRET                string `mask:"true"`
	SALT                  string `mask:"true"`
}

type SmsDestinationConfig struct {
	PREFIX   string `validate:"required,number"`
	PROVIDER string `validate:"required"`
}

type SmsDestination struct {
	SmsDestinationConfig
	provider Provider
}

func (s *SmsDestination) Config() interface{} {
	return &s.SmsDestinationConfig
}

func (s *SmsDestination) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {
	err := object_config.LoadLogValidate(cfg, log, vld, s, "sms_destination", configPath...)
	if err != nil {
		return log.PushFatalStack("failed to init SMS destination", err)
	}
	return nil
}

type SmsManagerBase struct {
	SmsManagerBaseConfig
	destinations    []*SmsDestination
	cipher          *crypt_utils.AEAD
	defaultProvider Provider

	db db.DB
}

func NewSmsManager() *SmsManagerBase {
	return &SmsManagerBase{}
}

func (s *SmsManagerBase) Config() interface{} {
	return &s.SmsManagerBaseConfig
}

func (s *SmsManagerBase) Init(cfg config.Config, log logger.Logger, vld validator.Validator, factory ProviderFactory, configPath ...string) error {

	// load configuration
	path := utils.OptionalArg("sms", configPath...)
	err := object_config.LoadLogValidate(cfg, log, vld, s, path)
	if err != nil {
		return log.PushFatalStack("failed to init SMS manager", err)
	}

	// init cipher
	if s.ENCRYPT_MESSAGE_STORE {
		if s.SECRET == "" {
			return log.PushFatalStack("encryption secret must not be empty", nil)
		}
		if s.SALT == "" {
			return log.PushFatalStack("encryption salt must not be empty", nil)
		}
		s.cipher, err = crypt_utils.NewAEAD(s.SECRET, []byte(s.SALT))
		if err != nil {
			return log.PushFatalStack("failed to init cipher for SMS manager", err)
		}
	}

	// load providers
	createProvider := func(protocol string) (Provider, error) {
		return factory.Create(protocol)
	}
	providersPath := object_config.Key(path, "providers")
	providers, err := object_config.LoadLogValidateSubobjectsMap(cfg, log, vld, providersPath, createProvider)
	if err != nil {
		return log.PushFatalStack("failed to load SMS providers", err)
	}

	// load destinations
	createDestination := func() *SmsDestination {
		return &SmsDestination{}
	}
	destinationsPath := object_config.Key(path, "destinations")
	destinations, err := object_config.LoadLogValidateSubobjectsList(cfg, log, vld, destinationsPath, createDestination)
	if err != nil {
		return log.PushFatalStack("failed to load SMS destinations", err)
	}

	// set default provider
	ok := false
	s.defaultProvider, ok = providers[s.DEFAULT_PROVIDER]
	if !ok {
		return log.PushFatalStack("unknown default provider", nil, logger.Fields{"default_provider": s.DEFAULT_PROVIDER})
	}

	// set destinations
	s.destinations = make([]*SmsDestination, 0)
	for _, destination := range destinations {
		destination.provider, ok = providers[destination.PROVIDER]
		if !ok {
			return log.PushFatalStack("unknown provider for destination", nil, logger.Fields{"provider": destination.PROVIDER, "destination": destination.PREFIX})
		}
		s.destinations = append(s.destinations, destination)
	}

	// sort destinations
	sort.SliceStable(s.destinations, func(i int, j int) bool {
		return len(s.destinations[i].PREFIX) > len(s.destinations[j].PREFIX)
	})

	// done
	return nil
}

func (s *SmsManagerBase) InitDbService(sctx context.Context, poool pool.Pool, role string) error {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("SmsManagerBase.InitDbService")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// connect to database
	s.db, err = pool.ConnectDatabaseService(sctx, poool, role, "")
	if err != nil {
		c.SetMessage("faield to connect to database service in the pool")
		return err
	}

	// done
	return nil
}

func (s *SmsManagerBase) Db(ctx op_context.Context) db.DBHandlers {
	if s.db != nil {
		return s.db
	}
	return op_context.DB(ctx)
}

func (s *SmsManagerBase) Send(sctx context.Context, message string, recipient string) (string, error) {

	// setup
	ctx := op_context.OpContext[auth.UserContext](sctx)
	c := ctx.TraceInMethod("SmsManagerBase.Send", logger.Fields{"recipient": recipient})
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		} else {
			c.Logger().Info("sms sent")
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find provider for destination
	provider := s.defaultProvider
	for _, destination := range s.destinations {
		if strings.HasPrefix(recipient, destination.PREFIX) {
			provider = destination.provider
			break
		}
	}
	c.SetLoggerField("provider", provider.Name())
	c.SetLoggerField("user", ctx.AuthUser().Display())

	// keep sms
	sms := &SmsMessage{}
	sms.InitObject()
	sms.SetUser(ctx.AuthUser())
	sms.Tenancy = auth.Tenancy(ctx)
	sms.Phone = recipient
	sms.Context = ctx.ID()
	sms.Operation = ctx.Name()
	sms.Provider = provider.Name()
	sms.Status = StatusSending
	c.LoggerFields()["sms_id"] = sms.GetID()
	if s.ENCRYPT_MESSAGE_STORE {
		ciphertext, err := s.cipher.Encrypt([]byte(message))
		if err != nil {
			c.SetMessage("failed to encrypt message")
			ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			return "", err
		}
		enc := utils.Base64StringCoding{}
		sms.Message = enc.Encode(ciphertext)
	} else {
		sms.Message = message
	}
	err = s.Db(ctx).Create(sctx, sms)
	if err != nil {
		c.SetMessage("failed to save SMS in database")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return "", err
	}

	// send SMS
	resp, err := provider.Send(sctx, message, recipient, sms.GetID())
	if resp != nil {
		sms.RawResponse = resp.RawContent
		sms.ForeignId = resp.ProviderMessageID
	}
	if err != nil {
		c.SetMessage("failed to send SMS")
		sms.Status = StatusFail
	} else {
		sms.Status = StatusSuccess
	}

	// update status in database
	err1 := db.Update(s.Db(ctx), sctx, sms, db.Fields{"status": sms.Status, "raw_response": sms.RawResponse, "foreign_id": sms.ForeignId})
	if err1 != nil {
		c.LoggerFields()["status"] = sms.Status
		c.LoggerFields()["raw_response"] = sms.RawResponse
		c.Logger().Error("failed to update SMS in database", err1)
	}

	// done
	return sms.GetID(), err
}

func (s *SmsManagerBase) AttachToErrorManager(errManager generic_error.ErrorManager) {
	errManager.AddErrorDescriptions(SmsErrorDescriptions)
	errManager.AddErrorProtocolCodes(SmsErrorHttpCodes)
}

func (s *SmsManagerBase) FindSms(sctx context.Context, smsId string) (*SmsMessage, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("SmsManagerBase.FindSms", logger.Fields{"sms_id": smsId})
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	msg := &SmsMessage{}
	found, err := s.Db(ctx).FindByField(sctx, "id", smsId, msg)
	if err != nil {
		c.SetMessage("failed to find SMS in database")
		return nil, err
	}
	if !found {
		err = errors.New("SMS not found")
		return nil, err
	}

	return msg, nil
}
