package test_utils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evgeniums/evgo/pkg/api/api_client/rest_api_client"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/http_request"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/go-querystring/query"
	"github.com/stretchr/testify/require"
)

type RestApiTestResponse struct {
	Raw         *httptest.ResponseRecorder
	body        []byte
	serverError generic_error.Error
}

func NewRestApiTestResponse(raw *httptest.ResponseRecorder) (*RestApiTestResponse, error) {
	resp := &RestApiTestResponse{Raw: raw}
	if resp.Code() >= http.StatusBadRequest {
		err := fillResponseError(resp)
		if err != nil {
			return resp, err
		}
	}
	return resp, nil
}

func fillResponseError(resp *RestApiTestResponse) error {
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

func (r *RestApiTestResponse) Code() int {
	return r.Raw.Code
}

func (r *RestApiTestResponse) Header() http.Header {
	return r.Raw.Header()
}

func (r *RestApiTestResponse) GetHeader(key string) string {
	return r.Raw.Header().Get(key)
}

func (r *RestApiTestResponse) SetHeader(key string, value string) {

	r.Raw.Header().Set(key, value)
}

func (r *RestApiTestResponse) Body() []byte {
	if r.Raw.Body != nil {
		r.body = r.Raw.Body.Bytes()
	}
	return r.body
}

func (r *RestApiTestResponse) Message() string {
	return string(r.Body())
}

func (r *RestApiTestResponse) Error() generic_error.Error {
	return r.serverError
}

func (r *RestApiTestResponse) SetError(err generic_error.Error) {
	r.serverError = err
}

func HttptestSendWithBody(t *testing.T, g *gin.Engine, method string, url string, cmd interface{}, headers ...map[string]string) *RestApiTestResponse {

	// prepare data
	cmdByte, err := json.Marshal(cmd)
	require.NoErrorf(t, err, "failed to marshal message")

	// create request
	req, err := http.NewRequest(method, url, bytes.NewBuffer(cmdByte))
	require.NoErrorf(t, err, "failed to create HTTP request")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.RemoteAddr = "127.0.0.1:80"
	http_request.HttpHeadersSet(req, headers...)

	// send request
	httprec := httptest.NewRecorder()
	g.ServeHTTP(httprec, req)

	// done
	resp, err := NewRestApiTestResponse(httprec)
	require.NoError(t, err)
	return resp
}

func HttptestSendWithQuery(t *testing.T, g *gin.Engine, method string, url string, cmd interface{}, headers ...map[string]string) *RestApiTestResponse {

	// create request
	req, err := http.NewRequest(method, url, nil)
	require.NoErrorf(t, err, "failed to create request")

	// prepare data
	if cmd != nil {
		v, err := query.Values(cmd)
		require.NoErrorf(t, err, "failed to build query")
		req.URL.RawQuery = v.Encode()
	}
	req.Header.Set("Accept", "application/json")
	http_request.HttpHeadersSet(req, headers...)
	req.RemoteAddr = "127.0.0.1:80"

	// send request
	httprec := httptest.NewRecorder()
	g.ServeHTTP(httprec, req)

	// done
	resp, err := NewRestApiTestResponse(httprec)
	require.NoError(t, err)
	return resp
}

func RestApiTestClient(t *testing.T, g *gin.Engine, baseUrl string, userAgent ...string) *rest_api_client.RestApiClientBase {

	sendWithBody := func(ctx op_context.Context, httpClient *http_request.HttpClient, method string, url string, cmd interface{}, headers ...map[string]string) (rest_api_client.Response, error) {
		resp := HttptestSendWithBody(t, g, method, url, cmd, headers...)
		return resp, nil
	}
	sendWithQuery := func(ctx op_context.Context, httpClient *http_request.HttpClient, method string, url string, cmd interface{}, headers ...map[string]string) (rest_api_client.Response, error) {
		resp := HttptestSendWithQuery(t, g, method, url, cmd, headers...)
		return resp, nil
	}

	c := rest_api_client.NewRestApiClientBase(sendWithBody, sendWithQuery)
	c.Init(http_request.DefaultHttpClient(), baseUrl, utils.OptionalArg("sed", userAgent...))
	return c
}
