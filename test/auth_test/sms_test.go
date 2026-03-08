package auth_test

import (
	"net/http"
	"testing"

	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_sms"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Cmd struct {
	Param1 string
	Param2 string
}

func TestSms(t *testing.T) {

	t.Skip("TODO FIX SMS TEST")

	app, users, server, _, sctx := initOpTest(t)
	defer app.Close()

	// create user1
	login1 := "user1@example.com"
	password1 := "password1"
	user1, err := users.Add(sctx, login1, password1, user.Phone("12345678", &User{}), user.Email("user1@example.com", &User{}))
	require.NoErrorf(t, err, "failed to add user")
	require.NotNil(t, user1)

	// prepare client
	client := test_utils.PrepareHttpClient(t, test_utils.BBGinEngine(t, server))

	// good login
	client.Login(login1, password1)

	// send with SMS confirmation
	assert.Empty(t, auth_sms.LastSmsCode)
	resp := client.Post("/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusOK})
	assert.NotEmpty(t, auth_sms.LastSmsCode)

	// send altered data or path
	t.Logf("Check mismatched data")
	cmd1 := &Cmd{Param1: "value1_1", Param2: "value1_2"}
	cmd2 := &Cmd{Param1: "value2_1", Param2: "value2_2"}
	client.AutoSms = false
	resp = client.Post("/status/sms", cmd1)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeSmsConfirmationRequired})
	smsToken := resp.Object.Header().Get("x-auth-sms-token")
	smsDelay := resp.Object.Header().Get("x-auth-sms-delay")
	t.Logf("Current delay: %s", smsDelay)
	resp = client.SendSmsConfirmation(resp, auth_sms.LastSmsCode, http.MethodPost, "/status/sms", cmd2)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeContentMismatch})
	headers := map[string]string{"x-auth-sms-token": smsToken}
	resp = client.SendSmsConfirmation(resp, auth_sms.LastSmsCode, http.MethodPost, "/status/sms-alt", cmd1, headers)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeContentMismatch})
	resp = client.SendSmsConfirmation(resp, auth_sms.LastSmsCode, http.MethodPut, "/status/sms", cmd1, headers)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeContentMismatch})
	resp = client.SendSmsConfirmation(resp, auth_sms.LastSmsCode, http.MethodPost, "/status/sms", cmd1, headers)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusOK})

	// check too many tries of invalid code
	client.AutoSms = false
	resp = client.Post("/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeSmsConfirmationRequired})
	resp = client.SendSmsConfirmation(resp, "0000", http.MethodPost, "/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeInvalidSmsCode})
	overridenToken := resp.Object.Header().Get("x-auth-sms-token")
	resp = client.SendSmsConfirmation(resp, "1111", http.MethodPost, "/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeInvalidSmsCode})
	actualToken := resp.Object.Header().Get("x-auth-sms-token")
	resp = client.SendSmsConfirmationWithToken(resp, overridenToken, "2222", http.MethodPost, "/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeTokenExpired})
	resp = client.SendSmsConfirmationWithToken(resp, actualToken, "2222", http.MethodPost, "/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeTooManyTries})

	// check delay
	client.AutoSms = false
	resp = client.Post("/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeWaitDelay})
	smsDelay = resp.Object.Header().Get("x-auth-sms-delay")
	t.Logf("Current delay: %s", smsDelay)
	assert.Equal(t, "2", smsDelay)
	client.Sleep(1, "SMS delay")
	client.AutoSms = false
	resp = client.Post("/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeWaitDelay})
	smsDelay = resp.Object.Header().Get("x-auth-sms-delay")
	t.Logf("Current delay: %s", smsDelay)
	assert.Equal(t, "1", smsDelay)
	client.Sleep(2, "SMS delay")
	client.AutoSms = false
	resp = client.Post("/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeSmsConfirmationRequired})
	resp = client.SendSmsConfirmation(resp, auth_sms.LastSmsCode, http.MethodPost, "/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusOK})

	// check token expiration
	client.AutoSms = false
	resp = client.Post("/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeSmsConfirmationRequired})
	client.Sleep(4, "SMS token expiration")
	resp = client.SendSmsConfirmation(resp, auth_sms.LastSmsCode, http.MethodPost, "/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeTokenExpired})

	// check no SMS token
	client.AutoSms = false
	resp = client.Post("/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeSmsConfirmationRequired})
	headers = map[string]string{"x-auth-sms-code": auth_sms.LastSmsCode}
	resp = client.Post("/status/sms", nil, headers)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeSmsTokenRequired})

	// check invalid SMS token
	client.Sleep(3, "SMS delay")
	client.AutoSms = false
	resp = client.Post("/status/sms", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeSmsConfirmationRequired})
	headers = map[string]string{"x-auth-sms-code": auth_sms.LastSmsCode, "x-auth-sms-token": "blabla"}
	resp = client.Post("/status/sms", nil, headers)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth_sms.ErrorCodeInvalidToken})
}
