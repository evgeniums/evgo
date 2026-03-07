package test_utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_login_phash"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_sms"
	"github.com/evgeniums/evgo/pkg/crypt_utils"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Expected struct {
	Error    string
	Message  string
	Details  string
	HttpCode int
}

type HttpResponse struct {
	Object  *httptest.ResponseRecorder
	Code    int
	Message string
}

func ResponseErrorCode(t *testing.T, resp *HttpResponse) string {
	if resp.Message != "" {
		errResp := generic_error.NewEmpty()
		require.NoError(t, json.Unmarshal([]byte(resp.Message), errResp))
		return errResp.Code()
	}
	return ""
}

func CheckResponse(t *testing.T, resp *HttpResponse, expected *Expected) {
	require.NotNil(t, resp)
	if expected == nil {
		assert.Contains(t, []int{http.StatusOK, http.StatusCreated}, resp.Code)
	} else {
		if expected.HttpCode != 0 {
			assert.Equal(t, expected.HttpCode, resp.Code)
		}
		if expected.Error != "" {
			assert.NotEmpty(t, resp.Message)
			if len(resp.Message) != 0 {
				errResp := generic_error.NewEmpty()
				require.NoError(t, json.Unmarshal([]byte(resp.Message), errResp))
				assert.Equal(t, expected.Error, errResp.Code())
				if expected.Details != "" {
					assert.Equal(t, expected.Details, errResp.Details())
				}
				if expected.Message != "" {
					assert.Equal(t, expected.Message, errResp.Message())
				}
			}
		} else {
			if expected.Message != "" {
				assert.Equal(t, expected.Message, resp.Message)
			}
		}
	}
}

type HttpClient struct {
	BaseUrl      string
	AccessToken  string
	RefreshToken string
	CsrfToken    string

	Gin *gin.Engine

	AutoSms bool

	T *testing.T
}

func NewHttpClient(t *testing.T, gin *gin.Engine, baseUrl ...string) *HttpClient {
	c := &HttpClient{}
	c.T = t
	c.BaseUrl = utils.OptionalArg("/api/1.0.0", baseUrl...)
	c.Gin = gin
	return c
}

func PrepareHttpClient(t *testing.T, gin *gin.Engine, baseUrl ...string) *HttpClient {
	c := &HttpClient{}
	c.T = t
	c.BaseUrl = utils.OptionalArg("/api/1.0.0", baseUrl...)
	c.Gin = gin
	c.AutoSms = true
	c.Prepare()
	return c
}

func (c *HttpClient) Url(path string) string {
	return c.BaseUrl + path
}

func (c *HttpClient) Login(user string, password string, expectedErrorCode ...string) {

	errorCode := utils.OptionalArg("", expectedErrorCode...)
	path := "/auth/login"

	if errorCode == auth.ErrorCodeUnauthorized {
		// request without headers
		resp := c.Post(path, nil)
		CheckResponse(c.T, resp, &Expected{HttpCode: http.StatusUnauthorized, Error: errorCode})
		return
	}

	// first step
	headers := map[string]string{"x-auth-login": user}
	resp := c.Post(path, nil, headers)
	if errorCode == auth_login_phash.ErrorCodeWaitRetry {
		CheckResponse(c.T, resp, &Expected{HttpCode: http.StatusTooManyRequests, Error: auth_login_phash.ErrorCodeWaitRetry})
		return
	}
	CheckResponse(c.T, resp, &Expected{HttpCode: http.StatusUnauthorized, Error: auth_login_phash.ErrorCodeCredentialsRequired})

	salt := resp.Object.Header().Get("x-auth-login-salt")
	require.NotEmpty(c.T, salt)

	// second
	phash := auth_login_phash.Phash(password, salt)
	headers["x-auth-login-phash"] = phash
	resp = c.Post(path, nil, headers)

	if errorCode != "" {
		CheckResponse(c.T, resp, &Expected{HttpCode: http.StatusUnauthorized, Error: errorCode})
	} else {
		CheckResponse(c.T, resp, &Expected{HttpCode: http.StatusOK})
		assert.Empty(c.T, resp.Message)
		c.AccessToken = resp.Object.Header().Get("x-auth-access-token")
		require.NotEmpty(c.T, c.AccessToken)
		c.RefreshToken = resp.Object.Header().Get("x-auth-refresh-token")
		require.NotEmpty(c.T, c.RefreshToken)
		assert.NotEqual(c.T, c.AccessToken, c.RefreshToken)
	}
}

func (c *HttpClient) addTokens(headers ...map[string]string) map[string]string {

	h := map[string]string{}
	if c.AccessToken != "" {
		h["x-auth-access-token"] = c.AccessToken
	}
	if c.CsrfToken != "" {
		h["x-csrf"] = c.CsrfToken
	}
	if len(headers) > 0 {
		utils.AppendMap(h, headers[0])
	}
	return h
}

func (c *HttpClient) updateToken(resp *httptest.ResponseRecorder, code int) {
	accessToken := resp.Header().Get("x-auth-access-token")
	if accessToken != "" {
		c.AccessToken = accessToken
	}
	refreshToken := resp.Header().Get("x-auth-refresh-token")
	if refreshToken != "" {
		c.RefreshToken = refreshToken
	}
	csrfToken := resp.Header().Get("x-csrf")
	if csrfToken != "" {
		c.CsrfToken = csrfToken
	}
}

func (c *HttpClient) SendSmsConfirmation(resp *HttpResponse, code string, method string, path string, cmd interface{}, headers ...map[string]string) *HttpResponse {
	c.AutoSms = false
	h := c.addTokens(headers...)
	h["x-auth-sms-code"] = code
	token := resp.Object.Header().Get("x-auth-sms-token")
	if token != "" {
		h["x-auth-sms-token"] = token
	}
	return c.RequestBody(method, path, cmd, h)
}

func (c *HttpClient) SendSmsConfirmationWithToken(resp *HttpResponse, token string, code string, method string, path string, cmd interface{}, headers ...map[string]string) *HttpResponse {
	c.AutoSms = false
	h := c.addTokens(headers...)
	h["x-auth-sms-code"] = code
	if token != "" {
		h["x-auth-sms-token"] = token
	}
	return c.RequestBody(method, path, cmd, h)
}

func (c *HttpClient) RequestBody(method string, path string, cmd interface{}, headers ...map[string]string) *HttpResponse {
	h := c.addTokens(headers...)
	c.addTokens(headers...)
	resp, code, message := HttpRequestBody(c.T, c.Gin, method, c.Url(path), cmd, h)
	c.updateToken(resp, code)
	r := &HttpResponse{resp, code, message}

	errCode := ResponseErrorCode(c.T, r)
	if c.AutoSms && errCode == auth_sms.ErrorCodeSmsConfirmationRequired {
		return c.SendSmsConfirmation(r, auth_sms.LastSmsCode, method, path, cmd, h)
	}
	c.AutoSms = true

	return r
}

func (c *HttpClient) RequestQuery(method string, path string, cmd interface{}, headers ...map[string]string) *HttpResponse {
	h := c.addTokens(headers...)
	resp, code, message := HttpRequestQuery(c.T, c.Gin, method, c.Url(path), cmd, h)
	c.updateToken(resp, code)
	return &HttpResponse{resp, code, message}
}

func (c *HttpClient) Post(path string, cmd interface{}, headers ...map[string]string) *HttpResponse {
	return c.RequestBody(http.MethodPost, path, cmd, headers...)
}

func (c *HttpClient) PostSigned(t *testing.T, signer *crypt_utils.RsaSigner, path string, cmd interface{}, headers ...map[string]string) *HttpResponse {

	content, err := json.Marshal(cmd)
	require.NoError(t, err)
	sig, err := signer.SignB64(content, http.MethodPost, path)
	h := map[string]string{"x-auth-signature": sig}
	require.NoError(t, err)
	if len(headers) > 0 {
		utils.AppendMap(h, headers[0])
	}

	return c.RequestBody(http.MethodPost, path, cmd, h)
}

func (c *HttpClient) Put(t *testing.T, path string, cmd interface{}, headers ...map[string]string) *HttpResponse {
	return c.RequestBody(http.MethodPatch, path, cmd, headers...)
}

func (c *HttpClient) Patch(path string, cmd interface{}, headers ...map[string]string) *HttpResponse {
	return c.RequestBody(http.MethodPatch, path, cmd, headers...)
}

func (c *HttpClient) Get(path string, cmd interface{}, headers ...map[string]string) *HttpResponse {
	return c.RequestQuery(http.MethodGet, path, cmd, headers...)
}

func (c *HttpClient) Delete(path string, cmd interface{}, headers ...map[string]string) *HttpResponse {
	return c.RequestQuery(http.MethodDelete, path, cmd, headers...)
}

func (c *HttpClient) Logout() {
	c.Post("/auth/logout", nil)
}

func (c *HttpClient) UpdateCsrfToken() {
	c.Get("/status/check", nil)
}

func (c *HttpClient) UpdateTokens() {
	c.UpdateCsrfToken()
	c.RequestRefreshToken()
}

func (c *HttpClient) Sleep(seconds int, message string) {
	c.T.Logf("Sleeping %d seconds for %s...", seconds, message)
	time.Sleep(time.Second * time.Duration(seconds))
	c.UpdateTokens()
}

func (c *HttpClient) RequestRefreshToken(expectedErrorCode ...string) {

	if c.RefreshToken == "" {
		return
	}

	errorCode := utils.OptionalArg("", expectedErrorCode...)

	h := map[string]string{"x-auth-refresh-token": c.RefreshToken}
	resp := c.Post("/auth/refresh", nil, h)

	if errorCode != "" {
		CheckResponse(c.T, resp, &Expected{Error: errorCode, HttpCode: http.StatusUnauthorized})
	} else {
		CheckResponse(c.T, resp, &Expected{HttpCode: http.StatusOK})
		assert.Empty(c.T, resp.Message)
		c.AccessToken = resp.Object.Header().Get("x-auth-access-token")
		require.NotEmpty(c.T, c.AccessToken)
	}
}

func (c *HttpClient) Prepare() {
	resp := c.Get("/status/check", nil)
	CheckResponse(c.T, resp, &Expected{
		HttpCode: http.StatusOK,
		Message:  `{"status":"running"}`})
}
