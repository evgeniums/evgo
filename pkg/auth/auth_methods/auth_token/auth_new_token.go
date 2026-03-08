package auth_token

import (
	"context"
	"time"

	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/auth/auth_session"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

const NewTokenProtocol = "new_token"

type AuthNewTokenHandler struct {
	AuthTokenHandler
}

func NewNewToken(users auth_session.WithUserSessionManager) *AuthNewTokenHandler {
	a := &AuthNewTokenHandler{}
	a.users = users
	return a
}

func (a *AuthNewTokenHandler) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	err := a.AuthTokenHandler.Init(cfg, log, vld, configPath...)
	if err != nil {
		return err
	}

	a.AuthHandlerBase.Init(NewTokenProtocol)

	return nil
}

func (a *AuthNewTokenHandler) Process(sctx context.Context) (bool, *Token, error) {

	// setup
	ctx := op_context.OpContext[auth.AuthContext](sctx)
	c := ctx.TraceInMethod("AuthNewTokenHandler.Process")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// check if user was already authenticated
	if ctx.AuthUser() == nil {
		return false, nil, nil
	}

	// user was authenticated, just create or update session client and add tokens

	sessionId := ctx.GetSessionId()
	var session auth_session.Session
	if sessionId == "" {
		// create session
		session, err = a.users.SessionManager().CreateSession(sctx, a.SessionExpiration())
		if err != nil {
			ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			return true, nil, err
		}
		ctx.SetSessionId(session.GetID())
	} else {
		// find session
		session, err = a.users.SessionManager().FindSession(sctx, sessionId)
		if err != nil {
			ctx.SetGenericErrorCode(ErrorCodeSessionExpired)
			return true, nil, err
		}
	}

	// update session client
	err = a.users.SessionManager().UpdateSessionClient(sctx)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, nil, err
	}

	// generate refresh token
	_, err = a.GenRefreshToken(sctx, session)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, nil, err
	}

	// generate access token
	token, err := a.GenAccessToken(sctx)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, nil, err
	}

	// done
	return true, token, nil
}

func (a *AuthNewTokenHandler) Handle(sctx context.Context) (bool, error) {
	ctx := op_context.OpContext[auth.AuthContext](sctx)
	c := ctx.TraceInMethod("AuthNewTokenHandler.Handle")
	defer ctx.TraceOutMethod()
	found, _, err := a.Process(sctx)
	return found, c.SetError(err)
}

func GenManualToken(sctx context.Context, cipher auth.AuthParameterEncryption,
	tenancyID string,
	user auth.User,
	sesisonID string,
	expirationSeconds int,
	tokenType string,
	parameters ...map[string]string) (string, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("GenManualToken")
	defer ctx.TraceOutMethod()

	token := &Token{}
	token.Id = utils.GenerateRand64()
	token.SessionId = sesisonID
	token.UserDisplay = user.Display()
	token.UserId = user.GetID()
	token.Tenancy = tenancyID
	token.SetTTL(expirationSeconds)
	token.Type = tokenType
	if len(parameters) > 0 {
		token.Parameters = parameters[0]
	}

	tookenStr, err := cipher.Encrypt(sctx, token)
	if err != nil {
		c.SetMessage("failed to encrypt token")
		return "", c.SetError(err)
	}

	// done
	return tookenStr, nil
}

func AddManualSession(sctx context.Context, cipher auth.AuthParameterEncryption, tenancyID string, users auth_session.WithUserSessionManager, login string, ttlSeconds int, tokenName ...string) (auth_session.Session, string, error) {

	// setup
	tokenType := utils.OptionalArg(AccessTokenName, tokenName...)
	loggerFields := logger.Fields{"tenancy": tenancyID, "login": login, "token-type": tokenType}
	ctx := op_context.OpContext[auth.AuthContext](sctx)
	c := ctx.TraceInMethod("auth_token.AddManualSession", loggerFields)
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find user
	user, err := users.AuthUserManager().FindAuthUser(sctx, login)
	if err != nil {
		c.SetMessage("failed to find user")
		return nil, "", err
	}

	// create session
	now := time.Now()
	expiration := now.Add(time.Second * time.Duration(ttlSeconds))
	userCtx, nextSctx := auth.NewUserContext(sctx)
	userCtx.User = user
	session, err := users.SessionManager().CreateSession(nextSctx, expiration)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return nil, "", err
	}

	// generate token
	token, err := GenManualToken(nextSctx, cipher, tenancyID, user, session.GetID(), ttlSeconds, tokenType)
	if err != nil {
		return nil, "", err
	}

	// done
	return session, token, nil
}
