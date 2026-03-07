package test_utils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/go-querystring/query"
	"github.com/stretchr/testify/require"
)

func HttpRequestSend(t *testing.T, g *gin.Engine, req *http.Request) (*httptest.ResponseRecorder, int, string) {
	response := httptest.NewRecorder()
	g.ServeHTTP(response, req)
	responseCode := response.Code
	responseMessage := ""
	if response.Body != nil {
		responseMessage = response.Body.String()
	}
	return response, responseCode, responseMessage
}

func HttpHeadersSet(req *http.Request, headers ...map[string]string) {
	req.Header.Set("User-Agent", "sed")
	req.RemoteAddr = "127.0.0.1:80"
	if len(headers) > 0 {
		for k, v := range headers[0] {
			req.Header.Set(k, v)
		}
	}
}

func HttpRequestBody(t *testing.T, g *gin.Engine, method string, path string, cmd interface{}, headers ...map[string]string) (*httptest.ResponseRecorder, int, string) {

	cmdStr, _ := json.Marshal(cmd)
	req, err := http.NewRequest(method, path, bytes.NewBuffer(cmdStr))
	require.NoErrorf(t, err, "failed to create request")

	HttpHeadersSet(req, headers...)

	return HttpRequestSend(t, g, req)
}

func HttpPost(t *testing.T, g *gin.Engine, path string, cmd interface{}, headers ...map[string]string) (*httptest.ResponseRecorder, int, string) {
	return HttpRequestBody(t, g, http.MethodPost, path, cmd, headers...)
}

func HttpPut(t *testing.T, g *gin.Engine, path string, cmd interface{}, headers ...map[string]string) (*httptest.ResponseRecorder, int, string) {
	return HttpRequestBody(t, g, http.MethodPut, path, cmd, headers...)
}

func HttpPatch(t *testing.T, g *gin.Engine, path string, cmd interface{}, headers ...map[string]string) (*httptest.ResponseRecorder, int, string) {
	return HttpRequestBody(t, g, http.MethodPatch, path, cmd, headers...)
}

func HttpRequestQuery(t *testing.T, g *gin.Engine, method string, path string, args interface{}, headers ...map[string]string) (*httptest.ResponseRecorder, int, string) {
	req, err := http.NewRequest(method, path, nil)
	if args != nil {
		v, _ := query.Values(args)
		req.URL.RawQuery = v.Encode()
		require.NoErrorf(t, err, "failed to create request")
	}
	HttpHeadersSet(req, headers...)

	return HttpRequestSend(t, g, req)
}

func HttpGet(t *testing.T, g *gin.Engine, path string, cmd interface{}, headers ...map[string]string) (*httptest.ResponseRecorder, int, string) {
	return HttpRequestQuery(t, g, http.MethodGet, path, cmd, headers...)
}

func HttpDelete(t *testing.T, g *gin.Engine, path string, cmd interface{}, headers ...map[string]string) (*httptest.ResponseRecorder, int, string) {
	return HttpRequestQuery(t, g, http.MethodDelete, path, cmd, headers...)
}
