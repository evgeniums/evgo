package auth_hmac

import (
	"errors"
	"net/http"

	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/config"
	"github.com/evgeniums/go-utils/pkg/config/object_config"
	"github.com/evgeniums/go-utils/pkg/crypt_utils"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/validator"
)

const HmacProtocol = "hmac"
const HmacParameter = "hmac"

type UserWithHmacSecret interface {
	HmacSecret() string
}

type AuthHmacConfig struct {
}

type AuthHmac struct {
	auth.AuthHandlerBase
	AuthHmacConfig
}

func (a *AuthHmac) Config() interface{} {
	return &a.AuthHmacConfig
}

func (a *AuthHmac) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	a.AuthHandlerBase.Init(HmacProtocol)

	err := object_config.LoadLogValidate(cfg, log, vld, a, "auth.methods.hmac", configPath...)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of HMAC auth handler", err)
	}
	return nil
}

const ErrorCodeInvalidHmac = "hmac_invalid"

func (a *AuthHmac) ErrorDescriptions() map[string]string {
	m := map[string]string{
		ErrorCodeInvalidHmac: "Invalid HMAC",
	}
	return m
}

func (a *AuthHmac) ErrorProtocolCodes() map[string]int {
	m := map[string]int{
		ErrorCodeInvalidHmac: http.StatusUnauthorized,
	}
	return m
}

// Check HMAC in request.
// Call this handler after discovering user (ctx.AuthUser() must be not nil).
// HMAC secret must be set for the user.
// HMAC string is calculated as BASE64(HMAC_SHA256(RequestMethod,RequestPath,RequestContent)), where BASE64 is calculated with padding.
func (a *AuthHmac) Handle(ctx auth.AuthContext) (bool, error) {

	// setup
	c := ctx.TraceInMethod("AuthHmac.Handle")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// get token from request
	requestHmac := ctx.GetAuthParameter(a.Protocol(), HmacParameter)
	if requestHmac == "" {
		return false, nil
	}

	// get secret from user
	if ctx.AuthUser() == nil {
		err := errors.New("unknown user")
		ctx.SetGenericErrorCode(auth.ErrorCodeUnauthorized)
		return true, err
	}
	user, ok := ctx.AuthUser().(UserWithHmacSecret)
	if !ok {
		err = errors.New("user must be of UserWithHmacSecret interface")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, err
	}

	secret := user.HmacSecret()
	if secret == "" {
		err := errors.New("hmac secret is not set for user")
		ctx.SetGenericErrorCode(auth.ErrorCodeUnauthorized)
		return true, err
	}

	// check hmac
	hmac := crypt_utils.NewHmac(secret)
	hmac.Calc([]byte(ctx.GetRequestMethod()), []byte(ctx.GetRequestPath()), ctx.GetRequestContent())
	err = hmac.CheckStr(requestHmac)
	if err != nil {
		ctx.SetGenericErrorCode(ErrorCodeInvalidHmac)
		return true, err
	}

	// done
	return true, nil
}
