package auth_sms

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/common"
	"github.com/evgeniums/go-utils/pkg/config"
	"github.com/evgeniums/go-utils/pkg/config/object_config"
	"github.com/evgeniums/go-utils/pkg/crypt_utils"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/sms"
	"github.com/evgeniums/go-utils/pkg/utils"
	"github.com/evgeniums/go-utils/pkg/validator"
)

var LastSmsCode = ""

const SmsProtocol = "sms"

const TokenName = "sms-token"
const PhoneName = "sms-phone"
const DelayName = "sms-delay"
const CodeName = "sms-code"

const SmsDelayCacheKey = "sms-delay"
const SmsTokenCacheKey = "sms-token"

type UserWithPhone interface {
	Phone() string
}

type SmsDelay struct {
	common.CreatedAtBase
}

type SmsCacheToken struct {
	Try      int    `json:"try"`
	Code     string `json:"code"`
	Checksum string `json:"checksum"`
	SmsId    string `json:"sms_id"`
}

type SmsToken struct {
	auth.ExpireToken
	common.IDBase
}

type AuthSmsConfig struct {
	TOKEN_TTL_SECONDS int    `default:"300" validate:"gt=0"`
	SMS_DELAY_SECONDS int    `default:"30" validate:"gt=0"`
	SECRET            string `validate:"required" mask:"true"`
	MAX_TRIES         int    `default:"3" validate:"gt=1"`
	CODE_LENGTH       int    `default:"5" validate:"gte=4"`
	TESTING           bool
}

type AuthSms struct {
	auth.AuthHandlerBase
	AuthSmsConfig
	Encryption auth.AuthParameterEncryption
	smsManager sms.SmsManager

	testCodes map[string]string
}

func (a *AuthSms) Config() interface{} {
	return &a.AuthSmsConfig
}

func New(smsManager sms.SmsManager) *AuthSms {
	a := &AuthSms{}
	a.smsManager = smsManager
	return a
}

func (a *AuthSms) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	a.AuthHandlerBase.Init(SmsProtocol)

	path := utils.OptionalArg("auth.methods.sms", configPath...)

	err := object_config.LoadLogValidate(cfg, log, vld, a, path)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of auth SMS handler", err)
	}

	testCodesPath := object_config.Key(path, "test_codes")
	a.testCodes = cfg.GetStringMapString(testCodesPath)

	encryption := &auth.AuthParameterEncryptionBase{}
	err = encryption.Init(cfg, log, vld, path)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of SMS encryption", err)
	}
	a.Encryption = encryption

	return nil
}

func (a *AuthSms) SetSmsManager(smsManager sms.SmsManager) {
	a.smsManager = smsManager
}

const ErrorCodeSmsConfirmationRequired = "sms_confirmation_required"
const ErrorCodeSmsTokenRequired = "sms_token_required"
const ErrorCodeTokenExpired = "sms_token_expired"
const ErrorCodeInvalidToken = "sms_token_invalid"
const ErrorCodeInvalidSmsCode = "sms_code_invalid"
const ErrorCodeWaitDelay = "sms_wait_delay"
const ErrorCodeContentMismatch = "sms_content_mismatch"
const ErrorCodeInvalidPhone = "sms_invalid_phone"
const ErrorCodeTooManyTries = "sms_too_many_tries"

func (a *AuthSms) ErrorDescriptions() map[string]string {
	m := map[string]string{
		ErrorCodeSmsConfirmationRequired: "Request must be confirmed with SMS",
		ErrorCodeSmsTokenRequired:        "SMS token must be present in request",
		ErrorCodeTokenExpired:            "SMS token expired",
		ErrorCodeInvalidToken:            "Invalid SMS token",
		ErrorCodeInvalidSmsCode:          "Invalid SMS code",
		ErrorCodeWaitDelay:               "Wait before requesting new SMS code",
		ErrorCodeContentMismatch:         "Content of initial request and content of current request mismatch",
		ErrorCodeInvalidPhone:            "failed to send SMS confirmation because of invalid phone number",
		ErrorCodeTooManyTries:            "Too many code tries",
	}
	return m
}

func (a *AuthSms) ErrorProtocolCodes() map[string]int {
	m := map[string]int{
		ErrorCodeSmsConfirmationRequired: http.StatusUnauthorized,
		ErrorCodeSmsTokenRequired:        http.StatusUnauthorized,
		ErrorCodeTokenExpired:            http.StatusUnauthorized,
		ErrorCodeInvalidToken:            http.StatusUnauthorized,
		ErrorCodeInvalidSmsCode:          http.StatusUnauthorized,
		ErrorCodeWaitDelay:               http.StatusUnauthorized,
		ErrorCodeTooManyTries:            http.StatusUnauthorized,
		ErrorCodeContentMismatch:         http.StatusUnauthorized,
	}
	return m
}

func (a *AuthSms) Handle(ctx auth.AuthContext) (bool, error) {

	// setup
	c := ctx.TraceInMethod("AuthSms.Handle")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// precheck context
	message := ""
	var skip bool
	err = ctx.CheckRequestContent(&message, &skip)
	if err != nil {
		c.SetMessage("failed to check request content")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeFormat)
		return true, err
	}
	if skip {
		return true, nil
	}

	// check if user authenticated
	if ctx.AuthUser() == nil {
		err = errors.New("unknown user")
		ctx.SetGenericErrorCode(auth.ErrorCodeUnauthorized)
		return true, err
	}
	userId := ctx.AuthUser().GetID()

	// get SMS code from request
	code := ctx.GetAuthParameter(a.Protocol(), CodeName)

	// check if code is set in request
	if code != "" {

		// code is set in request

		// extract and check token from request
		token := &SmsToken{}
		var exists bool
		exists, err = a.Encryption.GetAuthParameter(ctx, a.Protocol(), TokenName, token)
		if !exists {
			if err == nil {
				err = errors.New("SMS token not found")
			}
			ctx.SetGenericErrorCode(ErrorCodeSmsTokenRequired)
			return false, err
		}
		if err != nil {
			c.SetMessage("failed to get encrypted SMS token")
			ctx.SetGenericErrorCode(ErrorCodeInvalidToken)
			return true, err
		}
		if token.Expired() {
			err = errors.New("token expired")
			ctx.SetGenericErrorCode(ErrorCodeTokenExpired)
			return true, err
		}

		// find corresponding cache token
		cacheToken := &SmsCacheToken{}
		oldCacheKey := a.smsTokenCacheKey(token.GetID())
		var found bool
		found, err = ctx.Cache().Get(oldCacheKey, cacheToken)
		if err != nil {
			c.SetMessage("failed to get cache token")
			ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			return true, err
		}
		if !found {
			err = errors.New("cache token expired")
			ctx.SetGenericErrorCode(ErrorCodeTokenExpired)
			return true, err
		}

		// check tries count
		if cacheToken.Try >= a.MAX_TRIES {
			ctx.Cache().Unset(oldCacheKey)
			err = errors.New("too many tries")
			ctx.SetGenericErrorCode(ErrorCodeTooManyTries)
			return true, err
		}

		// check if this is the same request as initial
		h := a.hmacOfRequest(ctx, userId)
		err = h.CheckStr(cacheToken.Checksum)
		if err != nil {
			c.SetLoggerField("hash_cache_checksum", cacheToken.Checksum)
			c.SetLoggerField("hash_actual_checksum", h.SumStr())
			c.SetLoggerField("hash_path", ctx.GetRequestPath())
			c.SetLoggerField("hash_method", ctx.GetRequestMethod())
			c.SetLoggerField("hash_content_len", len(ctx.GetRequestContent()))
			c.SetMessage("invalid request checksum")
			ctx.SetGenericErrorCode(ErrorCodeContentMismatch)
			return false, err
		}

		// check SMS code
		if code != cacheToken.Code {

			// bad SMS code
			ctx.Cache().Unset(oldCacheKey)

			// regenerate token
			token.GenerateID()
			token.SetTTL(a.TOKEN_TTL_SECONDS)

			// keep cache token with increased tries count
			cacheToken.Try += 1
			err = a.setToken(ctx, c, cacheToken, token)
			if err != nil {
				return true, err
			}

			// done
			ctx.SetGenericErrorCode(ErrorCodeInvalidSmsCode)
			err = errors.New("invalid SMS code")
			return true, err
		}

		// good SMS code

		// remove data from cache
		ctx.Cache().Unset(oldCacheKey)
		ctx.Cache().Unset(a.smsDelayCacheKey(userId))

		// done
		return true, nil
	}

	// SMS code not present in request

	// check if SMS delay expired
	delayCacheKey := a.smsDelayCacheKey(userId)
	delayItem := &SmsDelay{}
	found, err := ctx.Cache().Get(delayCacheKey, delayItem)
	if err != nil {
		c.SetMessage("failed to get delay item from cache")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, err
	}
	if found {
		// set delay parameter in response
		now := time.Now()
		diff := now.Sub(delayItem.GetCreatedAt())
		delay := int(diff.Seconds())
		if delay > a.SMS_DELAY_SECONDS {
			delay = 0
		} else {
			delay = a.SMS_DELAY_SECONDS - delay
		}
		ctx.SetAuthParameter(SmsProtocol, DelayName, fmt.Sprintf("%d", delay))

		// done
		err = errors.New("wait for delay")
		ctx.SetGenericErrorCode(ErrorCodeWaitDelay)
		return true, err
	}

	// user must be of UserWithPhone interface
	user, ok := ctx.AuthUser().(UserWithPhone)
	if !ok {
		err = errors.New("user must be of UserWithPhone interface")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, err
	}

	// get user's phone number
	phone := user.Phone()
	if phone == "" {
		err = errors.New("unknown phone number")
		ctx.SetGenericErrorCode(ErrorCodeInvalidPhone)
		return true, err
	}
	ctx.SetAuthParameter(SmsProtocol, PhoneName, utils.MaskPhone(phone))

	// prepare token
	token := &SmsToken{}
	token.GenerateID()
	cacheToken := &SmsCacheToken{}
	cacheToken.Code = a.genCode(phone)
	cacheToken.Try = 1
	h := a.hmacOfRequest(ctx, userId)
	cacheToken.Checksum = h.SumStr()

	// ctx.SetLoggerField("hash_cache_checksum", cacheToken.Checksum)
	// ctx.SetLoggerField("hash_actual_checksum", h.SumStr())
	// ctx.SetLoggerField("hash_path", ctx.GetRequestPath())
	// ctx.SetLoggerField("hash_method", ctx.GetRequestMethod())
	// ctx.SetLoggerField("hash_content_len", len(ctx.GetRequestContent()))

	if a.TESTING {
		LastSmsCode = cacheToken.Code
	}

	// send SMS
	if message == "" {
		message = "code %s"
	}
	message = fmt.Sprintf(message, cacheToken.Code)
	cacheToken.SmsId, err = a.smsManager.Send(ctx, message, phone)
	if err != nil {
		c.SetMessage("failed to send SMS")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, err
	}

	// set token
	err = a.setToken(ctx, c, cacheToken, token)
	if err != nil {
		return true, err
	}

	// save SMS delay in cache
	delayItem.InitCreatedAt()
	err1 := ctx.Cache().Set(delayCacheKey, delayItem, a.SMS_DELAY_SECONDS)
	if err1 != nil {
		c.Logger().Error("failed to save SMS delay item in cache", err1)
	}
	// set delay parameter
	ctx.SetAuthParameter(SmsProtocol, DelayName, fmt.Sprintf("%d", a.SMS_DELAY_SECONDS))

	// set response code
	ctx.SetGenericErrorCode(ErrorCodeSmsConfirmationRequired)

	// done
	return true, errors.New("sms code not found")
}

func (a *AuthSms) hmacOfRequest(ctx auth.AuthContext, userId string) *crypt_utils.Hmac {
	h := crypt_utils.NewHmac(a.SECRET)
	h.Calc([]byte(userId), []byte(ctx.GetRequestMethod()), []byte(ctx.GetRequestPath()), ctx.GetRequestContent())
	return h
}

func (a *AuthSms) smsDelayCacheKey(userId string) string {
	return fmt.Sprintf("%s/%s", SmsDelayCacheKey, userId)
}

func (a *AuthSms) smsTokenCacheKey(userId string) string {
	return fmt.Sprintf("%s/%s", SmsTokenCacheKey, userId)
}

func (a *AuthSms) genCode(phone string) string {

	if a.TESTING && a.testCodes != nil {
		code, ok := a.testCodes[phone]
		if ok {
			return code
		}
	}

	r := rand.Uint32()
	str := fmt.Sprintf("%08d", r)
	return str[len(str)-a.CODE_LENGTH:]
}

func (a *AuthSms) setToken(ctx auth.AuthContext, c op_context.CallContext, cacheToken *SmsCacheToken, requestToken *SmsToken) error {

	// keep in cache
	newCacheKey := a.smsTokenCacheKey(requestToken.GetID())
	err := ctx.Cache().Set(newCacheKey, cacheToken, a.TOKEN_TTL_SECONDS)
	if err != nil {
		c.SetMessage("failed to save token in cache")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return err
	}

	// put token to response
	requestToken.SetTTL(a.TOKEN_TTL_SECONDS)
	err = a.Encryption.SetAuthParameter(ctx, a.Protocol(), TokenName, requestToken)
	if err != nil {
		c.SetMessage("failed to put token to response")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return err
	}

	return nil
}

func (a *AuthSms) SetAuthManager(manager auth.AuthManager) {
	manager.Schemas().AddHandler(a)
}
