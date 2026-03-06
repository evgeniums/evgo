package auth_token

import (
	"time"

	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/auth/auth_session"
	"github.com/evgeniums/go-utils/pkg/config"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/utils"
	"github.com/evgeniums/go-utils/pkg/validator"
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

func (a *AuthNewTokenHandler) Process(ctx auth.AuthContext) (bool, *Token, error) {

	// setup
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
		session, err = a.users.SessionManager().CreateSession(ctx, a.SessionExpiration())
		if err != nil {
			ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			return true, nil, err
		}
		ctx.SetSessionId(session.GetID())
	} else {
		// find session
		session, err = a.users.SessionManager().FindSession(ctx, sessionId)
		if err != nil {
			ctx.SetGenericErrorCode(ErrorCodeSessionExpired)
			return true, nil, err
		}
	}

	// update session client
	err = a.users.SessionManager().UpdateSessionClient(ctx)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, nil, err
	}

	// generate refresh token
	_, err = a.GenRefreshToken(ctx, session)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, nil, err
	}

	// generate access token
	token, err := a.GenAccessToken(ctx)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, nil, err
	}

	// done
	return true, token, nil
}

func (a *AuthNewTokenHandler) Handle(ctx auth.AuthContext) (bool, error) {
	c := ctx.TraceInMethod("AuthNewTokenHandler.Handle")
	defer ctx.TraceOutMethod()
	found, _, err := a.Process(ctx)
	return found, c.SetError(err)
}

func GenManualToken(ctx op_context.Context, cipher auth.AuthParameterEncryption,
	tenancyID string,
	user auth.User,
	sesisonID string,
	expirationSeconds int,
	tokenType string,
	parameters ...map[string]string) (string, error) {

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

	tookenStr, err := cipher.Encrypt(ctx, token)
	if err != nil {
		c.SetMessage("failed to encrypt token")
		return "", c.SetError(err)
	}

	// done
	return tookenStr, nil
}

func AddManualSession(ctx op_context.Context, cipher auth.AuthParameterEncryption, tenancyID string, users auth_session.WithUserSessionManager, login string, ttlSeconds int, tokenName ...string) (auth_session.Session, string, error) {

	// setup
	tokenType := utils.OptionalArg(AccessTokenName, tokenName...)
	loggerFields := logger.Fields{"tenancy": tenancyID, "login": login, "token-type": tokenType}
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
	user, err := users.AuthUserManager().FindAuthUser(ctx, login)
	if err != nil {
		c.SetMessage("failed to find user")
		return nil, "", err
	}

	// create session
	now := time.Now()
	expiration := now.Add(time.Second * time.Duration(ttlSeconds))
	userCtx := auth.NewUserContext(ctx)
	userCtx.User = user
	session, err := users.SessionManager().CreateSession(userCtx, expiration)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return nil, "", err
	}

	// generate token
	token, err := GenManualToken(ctx, cipher, tenancyID, user, session.GetID(), ttlSeconds, tokenType)
	if err != nil {
		return nil, "", err
	}

	// done
	return session, token, nil
}
