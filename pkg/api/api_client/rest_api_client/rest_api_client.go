package rest_api_client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_login_phash"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/http_request"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type RestApiClient interface {
	Url(path string) string

	Login(sctx context.Context, user string, password string) (Response, error)
	Logout(sctx context.Context) (Response, error)

	UpdateTokens(sctx context.Context) (Response, error)
	UpdateCsrfToken(sctx context.Context) (Response, error)
	RequestRefreshToken(sctx context.Context) (Response, error)

	Post(sctx context.Context, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error)
	Put(sctx context.Context, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error)
	Patch(sctx context.Context, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error)
	Get(sctx context.Context, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error)
	Delete(sctx context.Context, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error)

	SendSmsConfirmation(send DoRequest, sctx context.Context, resp Response, code string, method string, url string, cmd interface{}, headers ...map[string]string) (Response, error)
}

type DoRequest = func(sctx context.Context, httpClient *http_request.HttpClient, method string, path string, cmd interface{}, headers ...map[string]string) (Response, error)

type TenancyAuth struct {
	TenancyPath string
	TenancyType string
}

type RestApiClientBase struct {
	BaseUrl   string
	UserAgent string
	Tenancy   *TenancyAuth

	AccessToken  string
	RefreshToken string
	CsrfToken    string

	SendWithBody  DoRequest
	SendWithQuery DoRequest

	HttpClient *http_request.HttpClient
}

func NewRestApiClientBase(withBodySender DoRequest, withQuerySender DoRequest) *RestApiClientBase {
	r := &RestApiClientBase{}
	r.Construct(withBodySender, withQuerySender)
	return r
}

func (r *RestApiClientBase) Init(httpClient *http_request.HttpClient, baseUrl string, userAgent string, tenancy ...*TenancyAuth) {
	r.BaseUrl = baseUrl
	r.UserAgent = userAgent
	r.HttpClient = httpClient
	if len(tenancy) != 0 {
		r.Tenancy = tenancy[0]
	}
}

func (r *RestApiClientBase) Construct(withBodySender DoRequest, withQuerySender DoRequest) {
	r.SendWithBody = withBodySender
	r.SendWithQuery = withQuerySender
}

func (r *RestApiClientBase) Url(path string) string {
	return utils.ConcatStrings(r.BaseUrl, path)
}

func (r *RestApiClientBase) AuthPath(path string) string {
	if r.Tenancy == nil {
		return path
	}
	return fmt.Sprintf("/%s/%s%s", r.Tenancy.TenancyType, r.Tenancy.TenancyPath, path)
}

func (r *RestApiClientBase) SetRefreshToken(token string) {
	r.RefreshToken = token
}

func (r *RestApiClientBase) Login(sctx context.Context, user string, password string) (Response, error) {

	path := r.AuthPath("/auth/login")

	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("RestApiClientBase.Login", logger.Fields{"user": user, "path": path})
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// first step
	headers := map[string]string{"x-evgo-login": user}
	resp, err := r.Post(sctx, path, nil, nil, headers)
	if err != nil {
		if resp.Error().Code() != auth_login_phash.ErrorCodeCredentialsRequired {
			c.SetMessage("failed to send first request")
			return resp, err
		}
	}
	ctx.ClearError()

	// second
	salt := resp.Header().Get("x-evgo-login-salt")
	phash := auth_login_phash.Phash(password, salt)
	headers["x-evgo-login-phash"] = phash
	resp, err = r.Post(sctx, path, nil, nil, headers)
	if err != nil {
		c.SetMessage("failed to send second request")
		return resp, err
	}
	if resp.Code() != http.StatusOK {
		err = errors.New("login failed")
		c.SetLoggerField("error_code", resp.Error().Code())
		return resp, err
	}

	// done
	return resp, nil
}

func (r *RestApiClientBase) addTokens(headers ...map[string]string) map[string]string {

	h := map[string]string{}
	if r.AccessToken != "" {
		h["x-evgo-token-access"] = r.AccessToken
	}
	if r.CsrfToken != "" {
		h["x-csrf-token"] = r.CsrfToken
	}
	if len(headers) > 0 {
		utils.AppendMap(h, headers[0])
	}
	return h
}

func (r *RestApiClientBase) updateTokens(resp Response) {
	accessToken := resp.Header().Get("x-evgo-token-access")
	if accessToken != "" {
		r.AccessToken = accessToken
	}
	refreshToken := resp.Header().Get("x-evgo-token-refresh")
	if refreshToken != "" {
		r.RefreshToken = refreshToken
	}
	csrfToken := resp.Header().Get("x-csrf-token")
	if csrfToken != "" {
		r.CsrfToken = csrfToken
	}
}

func (r *RestApiClientBase) SendSmsConfirmation(send DoRequest, sctx context.Context, resp Response, code string, method string, path string, cmd interface{}, headers ...map[string]string) (Response, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("RestApiClientBase.SendSmsConfirmation")
	defer ctx.TraceOutMethod()

	hs := r.addTokens(headers...)
	hs["x-auth-sms-code"] = code
	token := resp.Header().Get("x-auth-sms-token")
	if token != "" {
		hs["x-auth-sms-token"] = token
	}
	nextResp, err := r.SendRequest(send, sctx, method, path, cmd, nil, hs)
	if err != nil {
		return nil, c.SetError(err)
	}
	return nextResp, nil
}

func (r *RestApiClientBase) SendRequest(send DoRequest, sctx context.Context, method string, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error) {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("RestApiClientBase.SendRequest", logger.Fields{"method": method, "path": path})
	defer ctx.TraceOutMethod()

	// prepare tokens
	hs := r.addTokens(headers...)
	if r.UserAgent != "" {
		hs["User-Agent"] = r.UserAgent
	}

	// send request
	resp, err := send(sctx, r.HttpClient, method, r.Url(path), cmd, hs)
	if err != nil {
		c.SetMessage("failed to send")
		return resp, c.SetError(err)
	}
	r.updateTokens(resp)

	// fill good response
	if resp.Code() < http.StatusBadRequest {
		if response != nil {
			b := resp.Body()
			if len(b) == 0 {
				return nil, c.SetError(errors.New("failed to parse empty response"))
			}

			err = json.Unmarshal(b, response)
			if err != nil {
				fmt.Printf("message: %s\n", err)
				return nil, c.SetError(errors.New("failed to parse response message"))
			}
		}
	} else if resp.Error() != nil {
		ctx.SetGenericError(resp.Error())
	}

	// done
	return resp, nil
}

func (r *RestApiClientBase) RequestBody(sctx context.Context, method string, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error) {
	return r.SendRequest(r.SendWithBody, sctx, method, path, cmd, response, headers...)
}

func (r *RestApiClientBase) RequestQuery(sctx context.Context, method string, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error) {
	return r.SendRequest(r.SendWithQuery, sctx, method, path, cmd, response, headers...)
}

func (r *RestApiClientBase) Post(sctx context.Context, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error) {
	return r.RequestBody(sctx, http.MethodPost, path, cmd, response, headers...)
}

func (r *RestApiClientBase) Put(sctx context.Context, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error) {
	return r.RequestBody(sctx, http.MethodPut, path, cmd, response, headers...)
}

func (r *RestApiClientBase) Patch(sctx context.Context, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error) {
	return r.RequestBody(sctx, http.MethodPatch, path, cmd, response, headers...)
}

func (r *RestApiClientBase) Get(sctx context.Context, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error) {
	return r.RequestQuery(sctx, http.MethodGet, path, cmd, response, headers...)
}

func (r *RestApiClientBase) Delete(sctx context.Context, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error) {
	return r.RequestQuery(sctx, http.MethodDelete, path, cmd, response, headers...)
}

func (r *RestApiClientBase) Logout(sctx context.Context) (Response, error) {

	path := r.AuthPath("/auth/logout")

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("RestApiClientBase.Logout", logger.Fields{"path": path})
	defer ctx.TraceOutMethod()
	resp, err := r.Post(sctx, path, nil, nil)
	if err != nil {
		return nil, c.SetError(err)
	}
	return resp, nil
}

func (r *RestApiClientBase) UpdateCsrfToken(sctx context.Context) (Response, error) {
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("RestApiClientBase.UpdateCsrfToken")
	defer ctx.TraceOutMethod()
	resp, err := r.Get(sctx, "/status/check", nil, nil)
	if err != nil {
		return nil, c.SetError(err)
	}
	if resp.Code() != http.StatusOK {
		err = errors.New("failed to update CSRF")
		c.SetLoggerField("error_code", resp.Error().Code())
		return resp, err
	}
	return resp, nil
}

func (r *RestApiClientBase) UpdateTokens(sctx context.Context) (Response, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("RestApiClientBase.UpdateTokens")
	defer ctx.TraceOutMethod()

	resp, err := r.UpdateCsrfToken(sctx)
	if err != nil {
		return resp, c.SetError(err)
	}

	resp, err = r.RequestRefreshToken(sctx)
	if err != nil {
		return resp, c.SetError(err)
	}

	return resp, nil
}

func (r *RestApiClientBase) RequestRefreshToken(sctx context.Context) (Response, error) {

	path := r.AuthPath("/auth/refresh")

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("RestApiClientBase.RequestRefreshToken", logger.Fields{"path": path})
	defer ctx.TraceOutMethod()

	r.AccessToken = ""
	headers := map[string]string{"x-evgo-token-refresh": r.RefreshToken}
	resp, err := r.Post(sctx, path, nil, nil, headers)
	if err != nil {
		return nil, c.SetError(err)
	}
	if resp.Code() != http.StatusOK {
		err = errors.New("failed to refresh token")
		c.SetLoggerField("error_code", resp.Error().Code())
		return resp, err
	}
	return resp, nil
}

func (r *RestApiClientBase) Prepare(sctx context.Context) (Response, error) {
	return r.UpdateCsrfToken(sctx)
}

func DefaultSendWithBody(sctx context.Context, httpClient *http_request.HttpClient, method string, url string, cmd interface{}, headers ...map[string]string) (Response, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("rest_api_client.DefaultSendWithBody", logger.Fields{"method": method, "url": url})
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// prepare data
	cmdByte, err := json.Marshal(cmd)
	if err != nil {
		c.SetMessage("failed to marshal message")
		return nil, c.SetError(err)
	}

	// create request
	req, err := httpClient.NewRequest(method, url, bytes.NewBuffer(cmdByte))
	if err != nil {
		c.SetMessage("failed to create request")
		return nil, c.SetError(err)
	}
	req.NativeRequest.Header.Set("Content-Type", "application/json")
	req.NativeRequest.Header.Set("Accept", "application/json")
	http_request.HttpHeadersSet(req.NativeRequest, headers...)

	// send request
	err = req.SendRaw(sctx)
	if err != nil {
		c.SetMessage("failed to send raw")
		return nil, c.SetError(err)
	}

	// parse response
	resp, err := NewResponse(req.NativeResponse)
	if err != nil {
		c.SetMessage("failed to parse response")
		return nil, c.SetError(err)
	}

	// done
	return resp, nil
}

func DefaultSendWithQuery(sctx context.Context, httpClient *http_request.HttpClient, method string, url string, cmd interface{}, headers ...map[string]string) (Response, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("http_request.DefaultSendWithQuery", logger.Fields{"method": method, "url": url})
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// create request
	req, err := httpClient.NewRequest(method, url, nil)
	if err != nil {
		c.SetMessage("failed to create request")
		return nil, c.SetError(err)
	}

	// prepare data
	req.NativeRequest.URL.RawQuery, err = http_request.UrlEncode(cmd)
	if err != nil {
		c.SetMessage("failed to build query")
		return nil, c.SetError(err)
	}
	req.NativeRequest.Header.Set("Accept", "application/json")
	http_request.HttpHeadersSet(req.NativeRequest, headers...)

	// send request
	err = req.SendRaw(sctx)
	if err != nil {
		c.SetMessage("failed to send raw request")
		return nil, c.SetError(err)
	}

	// parse response
	resp, err := NewResponse(req.NativeResponse)
	if err != nil {
		c.SetMessage("failed to parse response")
		return nil, c.SetError(err)
	}

	// done
	return resp, nil
}

func DefaultRestApiClient(httpClient *http_request.HttpClient, baseUrl string, userAgent ...string) *RestApiClientBase {
	c := NewRestApiClientBase(DefaultSendWithBody, DefaultSendWithQuery)
	c.Init(httpClient, baseUrl, utils.OptionalArg("sed", userAgent...))
	return c
}

func fillResponseError(resp Response) error {
	b := resp.Body()
	if b != nil {
		errResp := generic_error.NewEmpty()
		err := json.Unmarshal(b, errResp)
		if err != nil {
			return err
		}
		resp.SetError(errResp)
		return nil
	}
	return nil
}
