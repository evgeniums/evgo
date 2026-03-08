package auth_token

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/auth/auth_session"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

const TokenProtocol = "evgo_token"
const GenTokenProtocol = "evgo_token_gen"
const AccessTokenName = "access-token"
const RefreshTokenName = "refresh-token"
const TagName = "tag"
const AccessTokenExpireName = "access-expire"
const RefreshTokenExpireName = "refresh-expire"

const TokenTtlParameterKey = "token_ttl"

type AuthTokenHandlerConfig struct {
	ACCESS_TOKEN_TTL_SECONDS        int    `default:"900" validate:"gt=0"`
	REFRESH_TOKEN_TTL_SECONDS       int    `default:"43200" validate:"gt=0"`
	ACCESS_TOKEN_GEN_BEFORE_SECONDS int    `default:"60" validate:"gt=0"`
	AUTO_PROLONGATE_ACCESS          bool   `default:"true"`
	AUTO_PROLONGATE_REFRESH         bool   `default:"true"`
	REFRESH_PATH                    string `default:"/auth/refresh"`
	LOGOUT_PATH                     string `default:"/auth/logout"`

	AUTH_BEARER_HEADER bool `default:"false"`

	ACCESS_TOKEN_NAME string `default:"access-token"`
	DIRECT_TOKEN_NAME bool

	DISABLE_REFRESH bool
}

type AuthTokenHandler struct {
	auth.AuthHandlerBase
	AuthTokenHandlerConfig
	users      auth_session.WithUserSessionManager
	encryption auth.AuthParameterEncryption
}

type Token struct {
	auth.ExpireToken
	Id          string            `json:"id"`
	UserId      string            `json:"user_id"`
	UserDisplay string            `json:"user_display"`
	SessionId   string            `json:"session_id"`
	Tenancy     string            `json:"tenancy"`
	Type        string            `json:"type"`
	Parameters  map[string]string `json:"parameters"`
}

func (a *AuthTokenHandler) Config() interface{} {
	return &a.AuthTokenHandlerConfig
}

func New(users auth_session.WithUserSessionManager) *AuthTokenHandler {
	a := &AuthTokenHandler{}
	a.users = users
	return a
}

func (a *AuthTokenHandler) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	a.AuthHandlerBase.Init(TokenProtocol)

	path := utils.OptionalArg("auth.methods.token", configPath...)

	err := object_config.LoadLogValidate(cfg, log, vld, a, path)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of TOKEN handler", err)
	}

	encryption := &auth.AuthParameterEncryptionBase{}
	err = encryption.Init(cfg, log, vld, path)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of auth token encryption", err)
	}
	a.encryption = encryption

	return nil
}

const ErrorCodeTokenExpired = "auth_token_expired"
const ErrorCodeInvalidToken = "auth_token_invalid"
const ErrorCodeSessionExpired = "session_expired"
const ErrorCodeUnknownUser = "unknown_user"

const ErrorCodeRefreshDisabled = "refresh_disabled"

var ErrorDescriptions = map[string]string{
	ErrorCodeTokenExpired:    "Provided authentication token expired",
	ErrorCodeInvalidToken:    "Invalid authentication token",
	ErrorCodeSessionExpired:  "Session expired",
	ErrorCodeUnknownUser:     "Unknown user",
	ErrorCodeRefreshDisabled: "Refresh disabled",
}

var ErrorProtocolCodes = map[string]int{
	ErrorCodeTokenExpired:    http.StatusUnauthorized,
	ErrorCodeInvalidToken:    http.StatusUnauthorized,
	ErrorCodeSessionExpired:  http.StatusUnauthorized,
	ErrorCodeUnknownUser:     http.StatusUnauthorized,
	ErrorCodeRefreshDisabled: http.StatusUnauthorized,
}

func (a *AuthTokenHandler) ErrorDescriptions() map[string]string {
	return ErrorDescriptions
}

func (a *AuthTokenHandler) ErrorProtocolCodes() map[string]int {
	return ErrorProtocolCodes
}

func (a *AuthTokenHandler) Handle(sctx context.Context) (bool, error) {

	// setup
	ctx := op_context.OpContext[auth.AuthContext](sctx)
	c := ctx.TraceInMethod("AuthTokenHandler.Handle")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// chek if it is REFRESH token request or normal access
	path := ctx.GetRequestPath()
	refresh := path == a.REFRESH_PATH
	tokenName := a.ACCESS_TOKEN_NAME
	if refresh {

		if a.DISABLE_REFRESH {
			err = errors.New("refresh disabled")
			ctx.SetGenericErrorCode(ErrorCodeRefreshDisabled)
			return true, err
		}

		tokenName = RefreshTokenName
	}
	c.LoggerFields()["refresh"] = refresh

	// check token in request
	var exists bool
	prev := &Token{}
	if a.AUTH_BEARER_HEADER {
		exists, err = auth.GetAndDecodeBearer(sctx, a.encryption, prev)
	} else {
		exists, err = a.encryption.GetAuthParameter(sctx, a.Protocol(), tokenName, prev, ctx.GetAuthParameter(a.Protocol(), TagName), a.DIRECT_TOKEN_NAME)
	}
	if err != nil {
		c.SetMessage("failed to get encrypted auth parameter")
		ctx.SetGenericErrorCode(ErrorCodeInvalidToken)
		return true, err
	}
	if !exists {
		// token must be present
		return true, errors.New("token missing")
	}

	c.LoggerFields()["token"] = prev.Id
	ctx.SetLoggerField("user", prev.UserDisplay)
	ctx.SetLoggerField("session", prev.SessionId)
	if prev.Expired() {
		err = errors.New("token expired")
		if refresh {
			ctx.SetGenericErrorCode(ErrorCodeSessionExpired)
		} else {
			ctx.SetGenericErrorCode(ErrorCodeTokenExpired)
		}
		return true, err
	}
	// check token type
	if prev.Type != tokenName {
		err = errors.New("token types mismatch")
		ctx.SetGenericErrorCode(ErrorCodeInvalidToken)
		return true, err
	}

	// check tenancy
	if ctx.GetTenancy() != nil || prev.Tenancy != "" {
		if ctx.GetTenancy() == nil || prev.Tenancy != ctx.GetTenancy().GetID() {
			err = errors.New("invalid tenancy")
			c.LoggerFields()["token_tenancy"] = prev.Tenancy
			ctx.SetGenericErrorCode(ErrorCodeSessionExpired)
			return true, err
		}
	}

	// find session
	session, err := a.users.SessionManager().FindSession(sctx, prev.SessionId)
	if err != nil {
		ctx.SetGenericErrorCode(ErrorCodeSessionExpired)
		return true, err
	}

	// check if session is valid
	if !session.IsValid() {
		err = errors.New("session invalidated")
		ctx.SetGenericErrorCode(ErrorCodeSessionExpired)
		return true, err
	}
	now := time.Now()
	if now.After(session.GetExpiration()) {
		err = errors.New("session expired")
		ctx.SetGenericErrorCode(ErrorCodeSessionExpired)
		return true, err
	}

	// find user
	user, err := a.users.AuthUserManager().FindAuthUser(sctx, session.GetUserLogin())
	if err != nil {
		c.SetMessage("failed to load user")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, err
	}
	if user == nil {
		err := errors.New("user not found")
		ctx.SetGenericErrorCode(ErrorCodeUnknownUser)
		return true, err
	}
	ctx.SetLoggerField("user", user.Display())

	// check if user blocked
	if user.IsBlocked() {
		err = errors.New("user blocked")
		ctx.SetGenericErrorCode(ErrorCodeSessionExpired)
		return true, err
	}

	// set auth user
	ctx.SetAuthUser(user)

	// fill auth user
	err = a.users.AuthUserManager().FillAuthUser(sctx, true)
	if err != nil {
		c.SetMessage("failed to fill auth user")
		return true, err
	}

	// set user session
	ctx.SetSessionId(session.GetID())
	ctx.LoadSessionParameters(prev.Parameters)

	// update session client
	err = a.users.SessionManager().UpdateSessionClient(sctx)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, err
	}
	ctx.SetLoggerField("s_client", ctx.GetClientId())

	// add tokens if applicable
	if path != a.LOGOUT_PATH {

		if refresh || !refresh && a.AUTO_PROLONGATE_ACCESS {

			regenerateAccessToken := true
			if !refresh {
				tokenExpirationTime := now.Add(time.Second * time.Duration(a.ACCESS_TOKEN_GEN_BEFORE_SECONDS))
				regenerateAccessToken = tokenExpirationTime.After(prev.Exp)
			}

			if regenerateAccessToken {
				// generate access token
				_, err = a.GenAccessToken(sctx)
				if err != nil {
					ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
					return true, err
				}
			}
		}

		tokenExpirationTime := now.Add(time.Second * time.Duration(a.ACCESS_TOKEN_TTL_SECONDS))
		regenerateRefreshToken := a.AUTO_PROLONGATE_REFRESH && (refresh || tokenExpirationTime.After(session.GetExpiration()))
		if regenerateRefreshToken {
			// generate refresh token
			_, err = a.GenRefreshToken(sctx, session)
			if err != nil {
				ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
				return true, err
			}
		}

	} else {

		ctx.SetAuthUser(nil)

		// invalidate session on logout path
		err = a.users.SessionManager().InvalidateSession(sctx, session.GetUserId(), session.GetID())
		if err != nil {
			ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			return true, err
		}
	}

	// done
	return true, nil
}

func (a *AuthTokenHandler) GenAccessToken(sctx context.Context) (*Token, error) {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("AuthTokenHandler.GenAccessToken")
	defer ctx.TraceOutMethod()
	token, err := a.GenToken(sctx, a.ACCESS_TOKEN_NAME, a.ACCESS_TOKEN_TTL_SECONDS, AccessTokenExpireName)
	if err != nil {
		return nil, c.SetError(err)
	}
	return token, nil
}

func (a *AuthTokenHandler) GenRefreshToken(sctx context.Context, session auth_session.Session) (*Token, error) {

	if a.DISABLE_REFRESH {
		return nil, nil
	}

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("AuthTokenHandler.GenRefreshToken")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	expirationSeconds := a.REFRESH_TOKEN_TTL_SECONDS
	session.SetExpiration(a.SessionExpiration())
	err = a.users.SessionManager().UpdateSessionExpiration(sctx, session)
	if err != nil {
		c.SetMessage("failed to update session expiration")
		return nil, err
	}
	token, err := a.GenToken(sctx, RefreshTokenName, expirationSeconds, RefreshTokenExpireName)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (a *AuthTokenHandler) GenToken(sctx context.Context, paramName string, expirationSeconds int, expireParamName string) (*Token, error) {

	ctx := op_context.OpContext[auth.AuthContext](sctx)
	c := ctx.TraceInMethod("AuthTokenHandler.GenToken")
	defer ctx.TraceOutMethod()

	token := &Token{}
	token.Id = utils.GenerateRand64()
	token.SessionId = ctx.GetSessionId()
	token.UserDisplay = ctx.AuthUser().Display()
	token.UserId = ctx.AuthUser().GetID()
	if ctx.GetTenancy() != nil {
		token.Tenancy = ctx.GetTenancy().GetID()
	}
	token.Type = paramName
	token.SetTTL(expirationSeconds)
	token.Parameters = ctx.StoreSessionParameters()

	ctx.SetAuthParameter(a.Protocol(), TagName, a.encryption.CurrentTag())
	ctx.SetAuthParameter(a.Protocol(), expireParamName, utils.SerializeTime(token.Exp))
	err := a.encryption.SetAuthParameter(sctx, a.Protocol(), paramName, token, a.DIRECT_TOKEN_NAME)
	if err != nil {
		return nil, c.SetError(err)
	}
	return token, nil
}

func (a *AuthTokenHandler) SessionExpiration() time.Time {
	expirationSeconds := a.REFRESH_TOKEN_TTL_SECONDS
	return time.Now().Add(time.Second * time.Duration(expirationSeconds))
}

func (a *AuthTokenHandler) SetAuthManager(manager auth.AuthManager) {
	manager.Schemas().AddHandler(a)
}

type TokenSchema struct {
	auth.AuthSchema

	Token *AuthTokenHandler
}

func NewSchema(users auth_session.WithUserSessionManager) *TokenSchema {
	l := &TokenSchema{}
	l.Construct()
	l.Token = New(users)
	return l
}

func (t *TokenSchema) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	t.AuthHandlerBase.Init(TokenProtocol)

	err := t.Token.Init(cfg, log, vld, configPath...)
	if err != nil {
		return log.PushFatalStack("failed to init token handler", err)
	}

	t.AuthSchema.AppendHandlers(t.Token)
	return nil
}

func (t *TokenSchema) Handlers() []auth.AuthHandler {
	return t.AuthSchema.Handlers()
}

func (t *TokenSchema) SetAuthManager(manager auth.AuthManager) {
	manager.Schemas().AddHandler(t)
}

func ReloginRequired(code string) bool {
	return code == ErrorCodeInvalidToken || code == ErrorCodeSessionExpired || code == ErrorCodeUnknownUser
}

func RefreshRequired(code string) bool {
	return code == ErrorCodeTokenExpired
}
