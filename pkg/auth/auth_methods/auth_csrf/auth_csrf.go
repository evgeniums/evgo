package auth_csrf

import (
	"context"
	"errors"
	"net/http"

	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

const AntiCsrfProtocol = "csrf"
const AntiCsrfTokenName = "csrf"

type AuthCsrfConfig struct {
	TOKEN_TTL_SECONDS int `default:"300" validate:"gt=0"`
	IGNORE_PATHS      []string
}

type AuthCsrf struct {
	auth.AuthHandlerBase
	AuthCsrfConfig
	Encryption auth.AuthParameterEncryption
	skipPaths  map[string]bool
}

func New() *AuthCsrf {
	a := &AuthCsrf{}
	return a
}

func (a *AuthCsrf) Config() interface{} {
	return &a.AuthCsrfConfig
}

func (a *AuthCsrf) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	a.AuthHandlerBase.Init(AntiCsrfProtocol)

	path := utils.OptionalArg("csrf", configPath...)

	err := object_config.LoadLogValidate(cfg, log, vld, a, path)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of CSRF handler", err)
	}

	encryption := &auth.AuthParameterEncryptionBase{}
	err = encryption.Init(cfg, log, vld, path)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of CSRF encryption", err)
	}
	a.Encryption = encryption

	a.skipPaths = make(map[string]bool)
	if len(a.IGNORE_PATHS) == 0 {
		// skip default status service
		a.skipPaths["/status/check"] = true
	}
	for _, path := range a.IGNORE_PATHS {
		a.skipPaths[path] = true
	}

	return nil
}

const ErrorCodeAntiCsrfRequired = "anti_csrf_token_required"
const ErrorCodeTokenExpired = "anti_csrf_token_expired"
const ErrorCodeInvalidToken = "anti_csrf_token_invalid"

func (a *AuthCsrf) ErrorDescriptions() map[string]string {
	m := map[string]string{
		ErrorCodeAntiCsrfRequired: "Request must be protected with anti-CSRF token",
		ErrorCodeTokenExpired:     "Anti-CSRF token expired",
		ErrorCodeInvalidToken:     "Invalid anti-CSRF token",
	}
	return m
}

func (a *AuthCsrf) ErrorProtocolCodes() map[string]int {
	m := map[string]int{
		ErrorCodeAntiCsrfRequired: http.StatusForbidden,
		ErrorCodeTokenExpired:     http.StatusForbidden,
		ErrorCodeInvalidToken:     http.StatusForbidden,
	}
	return m
}

func (a *AuthCsrf) Handle(sctx context.Context) (bool, error) {

	// setup
	ctx := op_context.OpContext[auth.AuthContext](sctx)
	c := ctx.TraceInMethod("AuthCsrf.Handle", logger.Fields{"path": ctx.GetRequestPath()})
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// check token in request
	_, skip := a.skipPaths[ctx.GetRequestPath()]
	if !skip {
		prev := &auth.ExpireToken{}
		exists, err := a.Encryption.GetAuthParameter(sctx, a.Protocol(), AntiCsrfTokenName, prev, "")
		if !exists {
			err = errors.New("CSRF token not found")
			c.Logger().Debug("CSRF token not found")
			ctx.SetGenericErrorCode(ErrorCodeAntiCsrfRequired)
			return false, err
		}
		if err != nil {
			c.SetMessage("failed to get encrypted auth parameter")
			ctx.SetGenericErrorCode(ErrorCodeInvalidToken)
			return true, err
		}

		if prev.Expired() {
			err = errors.New("token expired")
			ctx.SetGenericErrorCode(ErrorCodeTokenExpired)
			return true, err
		}
	}

	// set token in response
	next := &auth.ExpireToken{}
	next.SetTTL(a.TOKEN_TTL_SECONDS)
	err = a.Encryption.SetAuthParameter(sctx, a.Protocol(), AntiCsrfTokenName, next)
	if err != nil {
		c.SetMessage("failed to set encrypted auth parameter")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, err
	}

	// done
	return true, nil
}

func IsCsrfError(code string) bool {
	return code == ErrorCodeAntiCsrfRequired || code == ErrorCodeInvalidToken || code == ErrorCodeTokenExpired
}
