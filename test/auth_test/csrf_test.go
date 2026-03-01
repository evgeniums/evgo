package auth_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evgeniums/go-utils/pkg/test_utils"
	"github.com/stretchr/testify/assert"
)

func TestCsrf(t *testing.T) {
	app, _, server := initServer(t)
	defer app.Close()

	client := test_utils.NewHttpClient(t, test_utils.BBGinEngine(t, server))

	resp := client.Get("/just-check-404", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{
		Error:    "not_found",
		HttpCode: http.StatusNotFound,
		Message:  "Requested resource was not found"})
	assert.Empty(t, client.CsrfToken)
	assert.Empty(t, client.AccessToken)
	assert.Empty(t, client.RefreshToken)

	resp = client.Get("/status/csrf", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{
		Error:    "anti_csrf_token_required",
		HttpCode: http.StatusForbidden,
		Message:  "Request must be protected with anti-CSRF token"})
	assert.Empty(t, client.CsrfToken)
	assert.Empty(t, client.AccessToken)
	assert.Empty(t, client.RefreshToken)

	resp = client.Get("/status/check", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{
		HttpCode: http.StatusOK,
		Message:  `{"status":"running"}`})
	assert.NotEmpty(t, client.CsrfToken)
	assert.Empty(t, client.AccessToken)
	assert.Empty(t, client.RefreshToken)
	prevToken := client.CsrfToken

	resp = client.Get("/status/csrf", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{
		HttpCode: http.StatusOK,
		Message:  `{"status":"success"}`})
	assert.NotEqual(t, prevToken, client.CsrfToken)

	t.Logf("Waiting 3 seconds for CSRF token expiration...")
	time.Sleep(time.Second * 3)

	resp = client.Get("/status/csrf", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{
		Error:    "anti_csrf_token_expired",
		HttpCode: http.StatusForbidden,
		Message:  "Anti-CSRF token expired"})

	resp = client.Get("/status/check", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{
		HttpCode: http.StatusOK,
		Message:  `{"status":"running"}`})

	resp = client.Get("/status/csrf", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{
		HttpCode: http.StatusOK,
		Message:  `{"status":"success"}`})
}
