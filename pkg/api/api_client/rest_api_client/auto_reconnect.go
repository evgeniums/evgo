package rest_api_client

import (
	"errors"
	"net/http"
	"sync"

	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_csrf"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_login_phash"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_token"
	"github.com/evgeniums/evgo/pkg/http_request"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type autoReconnect struct {
	client   *RestApiClientBase
	handlers api_client.AutoReconnectHandlers

	mutex   sync.RWMutex
	inLogin bool

	lastLogin    string
	lastPassword string
}

func newAutoReconnectHelper(handlers api_client.AutoReconnectHandlers) *autoReconnect {
	a := &autoReconnect{}
	a.handlers = handlers
	return a
}

func (a *autoReconnect) init() {
	token := a.handlers.GetRefreshToken()
	if token != "" {
		a.client.RefreshToken = token
	}
}

func (a *autoReconnect) resend(ctx op_context.Context, send func(opCtx op_context.Context) (Response, error), tries int) (Response, error) {
	ctx.ClearError()
	resp, err := send(ctx)
	return a.checkResponse(ctx, send, resp, err, tries-1)
}

func (a *autoReconnect) checkResponse(ctx op_context.Context, send func(opCtx op_context.Context) (Response, error), lastResp Response, lastErr error, resendTries int) (Response, error) {

	// setup
	var err error
	c := ctx.TraceInMethod("autoReconnect.checkResponse")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// check last response
	if lastResp == nil || lastResp.Error() == nil {
		// check last error
		err = lastErr
		if err != nil {
			c.SetMessage("last error")
			return lastResp, err
		}
		return lastResp, nil
	}

	// check resending tries
	if resendTries < 0 {
		err = lastErr
		if err != nil {
			c.SetMessage("error response")
			return lastResp, err
		}
		return lastResp, nil
	}

	// refresh CSRF token
	if lastResp.Code() == http.StatusForbidden && auth_csrf.IsCsrfError(lastResp.Error().Code()) {
		resp, err := a.client.UpdateCsrfToken(ctx)
		if !IsResponseOK(resp, err) {
			c.SetMessage("failed to update CSRF")
			return resp, err
		}
		c.Logger().Debug("resending after CSRF")
		resp, err = a.resend(ctx, send, utils.Min(resendTries, 2))
		if err != nil {
			c.SetMessage("failed to resend after CRSF")
			return resp, err
		}
		return resp, nil
	}

	// only unauthorized errors can be processed
	if lastResp.Code() != http.StatusUnauthorized {
		err = errors.New(lastResp.Error().Message())
		return lastResp, err
	}

	// login
	if a.client.RefreshToken == "" || auth_token.ReloginRequired(lastResp.Error().Code()) || lastResp.Error().Code() == auth_login_phash.ErrorCodeLoginFailed {

		a.mutex.RLock()
		inLogin := a.inLogin
		lastLogin := a.lastLogin
		lastPassword := a.lastPassword
		a.mutex.RUnlock()
		if inLogin {
			err = errors.New(lastResp.Error().Message())
			c.SetMessage("failed when in login")
			return lastResp, err
		}

		login, password, err := a.handlers.GetCredentials(ctx)
		if err != nil {
			c.SetMessage("failed to get credentials")
			return lastResp, err
		}
		if login == "" {
			err = errors.New("login must be specified in client credentials")
			return lastResp, err
		}
		if lastResp.Error().Code() == auth_login_phash.ErrorCodeLoginFailed && login == lastLogin && password == lastPassword {
			// relogin won't help
			err = lastResp.Error()
			c.SetMessage("login failed, credentials the same")
			return lastResp, err
		}

		a.mutex.Lock()
		a.inLogin = true
		a.lastLogin = login
		a.lastPassword = password
		a.mutex.Unlock()

		resp, err := a.client.Login(ctx, login, password)

		a.mutex.Lock()
		a.inLogin = false
		a.mutex.Unlock()

		if err != nil {
			c.SetMessage("failed to login")
			return resp, err
		}
		if resp == nil {
			err = errors.New("nil login response")
			return nil, err
		}
		ctx.ClearError()

		a.handlers.SaveRefreshToken(ctx, a.client.RefreshToken)
		resp, err = a.resend(ctx, send, utils.Min(resendTries, 1))
		if err != nil {
			return resp, err
		}
		return resp, nil
	}

	// refresh token
	if a.client.AccessToken == "" || auth_token.RefreshRequired(lastResp.Error().Code()) {

		a.mutex.RLock()
		inLogin := a.inLogin
		a.mutex.RUnlock()
		if inLogin {
			err = errors.New(lastResp.Error().Message())
			c.SetMessage("failed when in login")
			return lastResp, err
		}

		a.mutex.Lock()
		a.inLogin = true
		a.mutex.Unlock()

		resp, err := a.client.RequestRefreshToken(ctx)

		a.mutex.Lock()
		a.inLogin = false
		a.mutex.Unlock()

		if !IsResponseOK(resp, err) {
			c.SetMessage("failed to refresh auth token")
			return resp, err
		}
		a.handlers.SaveRefreshToken(ctx, a.client.RefreshToken)
		resp, err = a.resend(ctx, send, utils.Min(resendTries, 1))
		if err != nil {
			c.SetMessage("failed to resend after refreshing auth token")
			return resp, err
		}
		return resp, nil
	}

	// done
	err = lastErr
	if err != nil {
		c.SetMessage("error response")
		return lastResp, err
	}
	return lastResp, nil
}

func NewAutoReconnectRestApiClient(reconnectHandlers api_client.AutoReconnectHandlers) *RestApiClientWithConfig {

	reconnect := newAutoReconnectHelper(reconnectHandlers)
	var client *RestApiClientWithConfig

	sendWithBody := func(ctx op_context.Context, httpClient *http_request.HttpClient, method string, url string, cmd interface{}, headers ...map[string]string) (Response, error) {
		send := func(opCtx op_context.Context) (Response, error) {
			hs := client.addTokens(headers...)
			return DefaultSendWithBody(opCtx, httpClient, method, url, cmd, hs)
		}
		hs := client.addTokens(headers...)
		resp, err := DefaultSendWithBody(ctx, httpClient, method, url, cmd, hs)
		return reconnect.checkResponse(ctx, send, resp, err, 5)
	}
	sendWithQuery := func(ctx op_context.Context, httpClient *http_request.HttpClient, method string, url string, cmd interface{}, headers ...map[string]string) (Response, error) {
		send := func(opCtx op_context.Context) (Response, error) {
			return DefaultSendWithQuery(opCtx, httpClient, method, url, cmd, headers...)
		}
		resp, err := DefaultSendWithQuery(ctx, httpClient, method, url, cmd, headers...)
		return reconnect.checkResponse(ctx, send, resp, err, 5)
	}

	client = NewRestApiClientWithConfig(sendWithBody, sendWithQuery)
	reconnect.client = client.RestApiClientBase
	reconnect.init()
	return client
}
