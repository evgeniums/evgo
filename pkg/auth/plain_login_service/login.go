package plain_login_service

import (
	"crypto/subtle"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_login_phash"
	"github.com/evgeniums/evgo/pkg/generic_error"
)

type UserWithPlainPassword interface {
	PlainPassword() string
}

type LoginCmd struct {
	Username string `json:"username" validate:"required,alphanum_" vmessage:"Invalid login"`
	Password string `json:"password" validate:"omitempty,max=64" vmessage:"Password too big"`
}

type LoginResponse struct {
	api.ResponseStub
	Token string `json:"token"`
}

type LoginEndpoint struct {
	api_server.ResourceEndpoint
	service *PlainLoginService
}

type requestWrapper struct {
	api_server.Request

	token string
}

func (r *requestWrapper) SetAuthParameter(authMethodProtocol string, key string, value string, directKeyName ...bool) {
	r.token = value
}

func (e *LoginEndpoint) HandleRequest(request api_server.Request) error {

	// setup
	var err error
	c := request.TraceInMethod("auth.PLainLogin")
	defer request.TraceOutMethod()

	// parse command
	cmd, err := api_server.ParseValidateRequest[LoginCmd](request)
	if err != nil {
		c.SetMessage("failed to parse/validate command")
		return c.SetError(err)
	}

	// find user
	user, err := e.service.users.AuthUserManager().FindAuthUser(request, cmd.Username)
	if err != nil {
		c.SetMessage("user not found")
		request.SetGenericErrorCode(auth_login_phash.ErrorCodeLoginFailed)
		return c.SetError(err)
	}

	userWithPassword, ok := user.(UserWithPlainPassword)
	if !ok {
		request.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return c.SetErrorStr("user is not of UserWithPlainPassword type")
	}

	// check password
	if subtle.ConstantTimeCompare([]byte(userWithPassword.PlainPassword()), []byte(cmd.Password)) != 1 {
		request.SetGenericErrorCode(auth_login_phash.ErrorCodeLoginFailed)
		return c.SetErrorStr("invalid login or password")
	}

	// set auth user
	request.SetAuthUser(user)

	// fill auth user
	err = e.service.users.AuthUserManager().FillAuthUser(request)
	if err != nil {
		c.SetMessage("failed to fill auth user")
		return err
	}

	// generate session token
	requestWrapper := &requestWrapper{Request: request}
	_, _, err = e.service.tokenHandler.Process(requestWrapper)
	if err != nil {
		c.SetMessage("failed to process token")
		return c.SetError(err)
	}

	// prepare response
	resp := &LoginResponse{Token: requestWrapper.token}

	// set response
	request.Response().SetMessage(resp)

	// done
	return nil
}

func NewLoginEndpoint(service *PlainLoginService) *LoginEndpoint {
	ep := &LoginEndpoint{service: service}
	api_server.InitResourceEndpoint(ep, "login", "Login", access_control.Post)
	return ep
}
