package auth_login_phash

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/auth/auth_session"
	"github.com/evgeniums/go-utils/pkg/common"
	"github.com/evgeniums/go-utils/pkg/config"
	"github.com/evgeniums/go-utils/pkg/config/object_config"
	"github.com/evgeniums/go-utils/pkg/crypt_utils"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/utils"
	"github.com/evgeniums/go-utils/pkg/validator"
)

const LoginProtocol = "login_phash"
const LoginName = "login"
const SaltName = "login-salt"
const PasswordHashName = "login-phash"

// TODO use token for second phase

const DelayCacheKey = "login-delay"

type LoginDelay struct {
	common.CreatedAtBase
}

type UserWithPassword interface {
	PasswordHash() string
	PasswordSalt() string
}

type User interface {
	UserWithPassword
	SetPassword(string)
}

type UserBase struct {
	PASSWORD_HASH string `json:"-" hidden:"true"`
	PASSWORD_SALT string `json:"-" hidden:"true"`
}

func (u *UserBase) PasswordHash() string {
	return u.PASSWORD_HASH
}

func (u *UserBase) PasswordSalt() string {
	return u.PASSWORD_SALT
}

func (u *UserBase) SetPassword(password string) {
	u.PASSWORD_SALT = crypt_utils.GenerateString()
	u.PASSWORD_HASH = Phash(password, u.PASSWORD_SALT)
}

type LoginHandlerConfig struct {
	THROTTLE_DELAY_SECONDS int `default:"2" validate:"gt=0"`
}

// Auth handler for login processing. The AuthTokenHandler MUST ALWAYS follow this handler in session scheme with AND conjunction.
type LoginHandler struct {
	LoginHandlerConfig
	auth.AuthHandlerBase
	users auth_session.WithAuthUserManager
}

func New(users auth_session.WithAuthUserManager) *LoginHandler {
	l := &LoginHandler{}
	l.users = users
	return l
}

func (l *LoginHandler) Config() interface{} {
	return &l.LoginHandlerConfig
}

func (l *LoginHandler) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	l.AuthHandlerBase.Init(LoginProtocol)

	path := utils.OptionalArg("auth.methods.login_phash", configPath...)
	err := object_config.LoadLogValidate(cfg, log, vld, l, path)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of auth login_phash handler", err)
	}

	return nil
}

const ErrorCodeLoginFailed = "login_failed"
const ErrorCodeCredentialsRequired = "login_credentials_required"
const ErrorCodeWaitRetry = "wait_retry"

var ErrorDescriptions = map[string]string{
	ErrorCodeLoginFailed:         "Invalid login or password",
	ErrorCodeCredentialsRequired: "Credentials hash must be provided in request",
	ErrorCodeWaitRetry:           "Retry later",
}

var ErrorProtocolCodes = map[string]int{
	ErrorCodeLoginFailed:         http.StatusUnauthorized,
	ErrorCodeCredentialsRequired: http.StatusUnauthorized,
	ErrorCodeWaitRetry:           http.StatusTooManyRequests,
}

func (l *LoginHandler) ErrorDescriptions() map[string]string {
	return ErrorDescriptions
}

func (l *LoginHandler) ErrorProtocolCodes() map[string]int {
	return ErrorProtocolCodes
}

func IsLoginError(err generic_error.Error) bool {
	if err == nil {
		return false
	}
	_, found := ErrorDescriptions[err.Code()]
	return found
}

func (l *LoginHandler) Handle(ctx auth.AuthContext) (bool, error) {

	// setup
	c := ctx.TraceInMethod("LoginHandler.Handle")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// get password hash from request
	phash := ctx.GetAuthParameter(l.Protocol(), PasswordHashName)

	// get login from request
	login := ctx.GetAuthParameter(l.Protocol(), LoginName)
	if login == "" {
		return false, nil
	}
	ctx.SetLoggerField("login", login)
	err = l.users.AuthUserManager().ValidateLogin(login)
	if err != nil {
		err = errors.New("invalid login format")
		if phash == "" {
			// forward client to second step anyway with fake salt
			ctx.SetAuthParameter(l.Protocol(), SaltName, crypt_utils.GenerateString())
			ctx.SetGenericErrorCode(ErrorCodeCredentialsRequired)
		} else {
			ctx.SetGenericErrorCode(ErrorCodeLoginFailed)
		}
		return true, err
	}

	// check delay expired
	delayCacheKey := l.delayCacheKey(login)
	delayItem := &LoginDelay{}
	found, err := ctx.Cache().Get(delayCacheKey, delayItem)
	if err != nil {
		c.SetMessage("failed to get delay item from cache")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, err
	}
	if found {
		// throttle login
		err = errors.New("wait for delay")
		ctx.SetGenericErrorCode(ErrorCodeWaitRetry)
		return true, err
	}

	// load user
	dbUser, err := l.users.AuthUserManager().FindAuthUser(ctx, login)
	if err != nil {
		err = errors.New("failed to load user")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, err
	}
	if dbUser == nil {
		err = errors.New("user not found")
		if phash == "" {
			// forward client to second step anyway with fake salt
			ctx.SetAuthParameter(l.Protocol(), SaltName, crypt_utils.GenerateString())
			ctx.SetGenericErrorCode(ErrorCodeCredentialsRequired)
		} else {
			l.setDelay(ctx, c, delayCacheKey, delayItem)
			ctx.SetGenericErrorCode(ErrorCodeLoginFailed)
		}

		return true, err
	}
	ctx.SetLoggerField("user", dbUser.Display())

	// check if user blocked
	if dbUser.IsBlocked() {
		err = errors.New("user blocked")
		ctx.SetGenericErrorCode(ErrorCodeLoginFailed)
		l.setDelay(ctx, c, delayCacheKey, delayItem)
		return true, err
	}

	// user must be of User interface
	phashUser, ok := dbUser.(UserWithPassword)
	if !ok {
		err = errors.New("user must be of auth_login_phash.User interface")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return true, err
	}

	// extract user salt
	// TODO generate salt on first step, use token for second step
	salt := phashUser.PasswordSalt()

	// check password hash
	if phash != "" {

		// check password hash
		if CheckPasswordHash(phashUser.PasswordHash(), phashUser.PasswordSalt(), phash) != nil {
			err = errors.New("invalid password hash")
			ctx.SetGenericErrorCode(ErrorCodeLoginFailed)
			l.setDelay(ctx, c, delayCacheKey, delayItem)
			return true, err
		}

		// set context user
		ctx.SetAuthUser(dbUser)

		// fill auth user
		err = l.users.AuthUserManager().FillAuthUser(ctx)
		if err != nil {
			c.SetMessage("failed to fill auth user")
			return true, err
		}

		// done
		return true, nil
	}

	// add salt to auth parameters
	ctx.SetAuthParameter(l.Protocol(), SaltName, salt)
	ctx.SetGenericErrorCode(ErrorCodeCredentialsRequired)

	// done
	err = errors.New("credentials not provided")
	return true, err
}

func (l *LoginHandler) delayCacheKey(userId string) string {
	return fmt.Sprintf("%s/%s", DelayCacheKey, userId)
}

func (l *LoginHandler) setDelay(ctx op_context.Context, c op_context.CallContext, delayCacheKey string, delayItem *LoginDelay) {
	if l.THROTTLE_DELAY_SECONDS != 0 {
		delayItem.InitCreatedAt()
		err1 := ctx.Cache().Set(delayCacheKey, delayItem, l.THROTTLE_DELAY_SECONDS)
		if err1 != nil {
			c.Logger().Error("failed to save delay item in cache", err1)
		}
	}
}

func Phash(password string, salt string) string {
	h := crypt_utils.NewHmac(password)
	return h.CalcStrStr(salt)
}

func CheckPasswordHash(password string, salt string, phash string) error {
	m := crypt_utils.NewHmac(password)
	m.CalcStr([]byte(salt))
	err := m.CheckStr(phash)
	return err
}
