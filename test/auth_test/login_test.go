package auth_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evgeniums/evgo/pkg/api/bare_bones_server"
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_login_phash"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_token"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_session_default"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initOpTest(t *testing.T, config ...string) (app_context.Context, *user_session_default.Users, bare_bones_server.Server, op_context.Context) {
	app, users, server := initServer(t, config...)

	ctx := test_utils.SimpleOpContext(app, t.Name())

	return app, users, server, ctx
}

func TestPHash(t *testing.T) {

	password := "khjliuhsd"
	salt := "aitldk"

	result := auth_login_phash.Phash(password, salt)
	t.Logf("Result: %s", result)

}

func TestLogin(t *testing.T) {
	app, users, server, opCtx := initOpTest(t)
	defer app.Close()

	// create user1
	login1 := "user1@example.com"
	password1 := "password1"
	user1, err := users.Add(opCtx, login1, password1, user.Phone("12345678", &User{}), user.Email("user1@example.com", &User{}))
	require.NoErrorf(t, err, "failed to add user")
	require.NotNil(t, user1)

	// create user2
	login2 := "user2"
	password2 := "password2"
	user2, err := users.Add(opCtx, login2, password2, user.Phone("12345679", &User{}), user.Email("user2@example.com", &User{}))
	require.NoErrorf(t, err, "failed to add user")
	require.NotNil(t, user2)

	// prepare client
	client := test_utils.PrepareHttpClient(t, test_utils.BBGinEngine(t, server))

	// login without headers
	client.Login(login1, password1, auth.ErrorCodeUnauthorized)

	// invalid user name
	client.Login("kjlkhoi43909Abc", password1, auth_login_phash.ErrorCodeLoginFailed)

	// unknown user
	client.Login("someuser", password1, auth_login_phash.ErrorCodeLoginFailed)

	// unknown user
	client.Login("someuser", password1, auth_login_phash.ErrorCodeWaitRetry)

	// invalid password
	client.Login(login1, ";oiu'oij;lkj", auth_login_phash.ErrorCodeLoginFailed)

	// invalid password
	client.Login(login1, ";oiu'oij;lkj", auth_login_phash.ErrorCodeWaitRetry)

	// invalid password with throttle
	client.Sleep(3, "login throttle")
	client.Login(login1, ";oiu'oij;lkj", auth_login_phash.ErrorCodeLoginFailed)
	client.Login(login1, password1, auth_login_phash.ErrorCodeWaitRetry)

	// good login
	client.Sleep(3, "login throttle")
	client.Login(login1, password1)

	// re-login without headers when authenticated
	client.Login(login2, password2, auth.ErrorCodeUnauthorized)

	// good re-login when authenticated
	client.Login(login2, password2)

	// invalid re-login when authenticated
	client.Login("someuser", password1, auth_login_phash.ErrorCodeLoginFailed)
}

type StrippedSession struct {
	Valid      bool
	Expiration time.Time
}

func TestSession(t *testing.T) {
	app, users, server, opCtx := initOpTest(t)
	defer app.Close()

	// create user1
	login1 := "user1"
	password1 := "password1"
	user1, err := users.Add(opCtx, login1, password1, user.Phone("12345678", &User{}), user.Email("user1@example.com", &User{}))
	require.NoErrorf(t, err, "failed to add user")
	require.NotNil(t, user1)

	// create user2
	login2 := "user2"
	password2 := "password2"
	user2, err := users.Add(opCtx, login2, password2, user.Phone("12345679", &User{}), user.Email("user2@example.com", &User{}))
	require.NoErrorf(t, err, "failed to add user")
	require.NotNil(t, user2)

	// prepare client1
	client1 := test_utils.PrepareHttpClient(t, test_utils.BBGinEngine(t, server))
	// prepare client2
	client2 := test_utils.PrepareHttpClient(t, test_utils.BBGinEngine(t, server))

	// request before login
	resp := client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Error: auth.ErrorCodeUnauthorized, HttpCode: http.StatusUnauthorized})

	// check sessions after login
	client1.Login(login1, password1)
	client2.Login(login2, password2)

	filter := &db.Filter{}
	filter.SortDirection = db.SORT_ASC
	filter.SortField = "user_login"

	sessions := make([]user_session_default.UserSession, 0)
	_, err = users.SessionManager().GetSessions(opCtx, filter, &sessions)
	require.NoError(t, err)
	require.Equal(t, 2, len(sessions))
	assert.True(t, sessions[0].IsValid())
	assert.True(t, sessions[1].IsValid())
	assert.Equal(t, user1.GetID(), sessions[0].GetUserId())
	assert.Equal(t, user2.GetID(), sessions[1].GetUserId())
	assert.Equal(t, user1.Login(), sessions[0].GetUserLogin())
	assert.Equal(t, user2.Login(), sessions[1].GetUserLogin())
	assert.Equal(t, user1.Display(), sessions[0].GetUserDisplay())
	assert.Equal(t, user2.Display(), sessions[1].GetUserDisplay())

	sessionClients := make([]user_session_default.UserSessionClient, 0)
	_, err = users.SessionManager().GetSessions(opCtx, filter, &sessionClients)
	require.NoError(t, err)
	require.Equal(t, 2, len(sessionClients))
	assert.Equal(t, user1.GetID(), sessionClients[0].GetUserId())
	assert.Equal(t, user2.GetID(), sessionClients[1].GetUserId())
	assert.Equal(t, user1.Login(), sessionClients[0].GetUserLogin())
	assert.Equal(t, user2.Login(), sessionClients[1].GetUserLogin())
	assert.Equal(t, user1.Display(), sessionClients[0].GetUserDisplay())
	assert.Equal(t, user2.Display(), sessionClients[1].GetUserDisplay())
	assert.Equal(t, sessions[0].GetID(), sessionClients[0].GetSessionId())
	assert.Equal(t, sessions[1].GetID(), sessionClients[1].GetSessionId())

	// request after login
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Message: `{"status":"success"}`})
	resp = client2.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Message: `{"status":"success"}`})

	// request after logout
	client1.Logout()
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Error: auth_token.ErrorCodeSessionExpired, HttpCode: http.StatusUnauthorized})
	resp = client2.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Message: `{"status":"success"}`})

	sessions = make([]user_session_default.UserSession, 0)
	_, err = users.SessionManager().GetSessions(opCtx, filter, &sessions)
	require.NoError(t, err)
	require.Equal(t, 2, len(sessions))
	assert.False(t, sessions[0].IsValid())
	assert.True(t, sessions[1].IsValid())

	// invalidate session
	users.SessionManager().InvalidateSession(opCtx, user2.GetID(), sessions[1].GetID())
	resp = client2.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Error: auth_token.ErrorCodeSessionExpired, HttpCode: http.StatusUnauthorized})
	sessions = make([]user_session_default.UserSession, 0)
	_, err = users.SessionManager().GetSessions(opCtx, filter, &sessions)
	require.NoError(t, err)
	require.Equal(t, 2, len(sessions))
	assert.False(t, sessions[1].IsValid())

	// invalidate all user sessions
	client1.Login(login1, password1)
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Message: `{"status":"success"}`})
	filter.SortDirection = db.SORT_DESC
	filter.SortField = "created_at"
	sessions = make([]user_session_default.UserSession, 0)
	_, err = users.SessionManager().GetSessions(opCtx, filter, &sessions)
	require.NoError(t, err)
	require.Equal(t, 3, len(sessions))
	assert.True(t, sessions[0].IsValid())
	assert.False(t, sessions[1].IsValid())
	assert.False(t, sessions[2].IsValid())
	users.SessionManager().InvalidateUserSessions(opCtx, user1.GetID())
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Error: auth_token.ErrorCodeSessionExpired, HttpCode: http.StatusUnauthorized})
	sessions = make([]user_session_default.UserSession, 0)
	_, err = users.SessionManager().GetSessions(opCtx, filter, &sessions)
	require.NoError(t, err)
	require.Equal(t, 3, len(sessions))
	assert.False(t, sessions[0].IsValid())
	assert.False(t, sessions[1].IsValid())
	assert.False(t, sessions[2].IsValid())

	// invalidate all sessions
	client1.Login(login1, password1)
	client2.Login(login1, password1)
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Message: `{"status":"success"}`})
	resp = client2.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Message: `{"status":"success"}`})
	filter.SortDirection = db.SORT_DESC
	filter.SortField = "created_at"
	sessions = make([]user_session_default.UserSession, 0)
	_, err = users.SessionManager().GetSessions(opCtx, filter, &sessions)
	require.NoError(t, err)
	require.Equal(t, 5, len(sessions))
	assert.True(t, sessions[0].IsValid())
	assert.True(t, sessions[1].IsValid())
	assert.False(t, sessions[2].IsValid())
	assert.False(t, sessions[3].IsValid())
	assert.False(t, sessions[4].IsValid())

	strippedSessions := []StrippedSession{}
	_, err = app.Db().FindWithFilter(app, filter, sessions, &strippedSessions)
	require.NoError(t, err)
	require.Equal(t, 5, len(strippedSessions))
	assert.True(t, strippedSessions[0].Valid)
	assert.True(t, strippedSessions[1].Valid)
	assert.False(t, strippedSessions[2].Valid)
	assert.False(t, strippedSessions[3].Valid)
	assert.False(t, strippedSessions[4].Valid)

	users.SessionManager().InvalidateAllSessions(opCtx)
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Error: auth_token.ErrorCodeSessionExpired, HttpCode: http.StatusUnauthorized})
	resp = client2.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Error: auth_token.ErrorCodeSessionExpired, HttpCode: http.StatusUnauthorized})
	sessions = make([]user_session_default.UserSession, 0)
	_, err = users.SessionManager().GetSessions(opCtx, filter, &sessions)
	require.NoError(t, err)
	require.Equal(t, 5, len(sessions))
	assert.False(t, sessions[0].IsValid())
	assert.False(t, sessions[1].IsValid())
	assert.False(t, sessions[2].IsValid())
	assert.False(t, sessions[3].IsValid())
	assert.False(t, sessions[4].IsValid())
}

func TestTokens(t *testing.T) {
	app, users, server, opCtx := initOpTest(t)
	defer app.Close()

	// create user1
	login1 := "user1"
	password1 := "password1"
	user1, err := users.Add(opCtx, login1, password1, user.Phone("12345678", &User{}), user.Email("user1@example.com", &User{}))
	require.NoErrorf(t, err, "failed to add user")
	require.NotNil(t, user1)

	// create user2
	login2 := "user2"
	password2 := "password2"
	user2, err := users.Add(opCtx, login2, password2, user.Phone("12345679", &User{}), user.Email("user2@example.com", &User{}))
	require.NoErrorf(t, err, "failed to add user")
	require.NotNil(t, user2)

	// prepare client1
	client1 := test_utils.PrepareHttpClient(t, test_utils.BBGinEngine(t, server))
	// prepare client2
	client2 := test_utils.PrepareHttpClient(t, test_utils.BBGinEngine(t, server))

	client1.Login(login1, password1)
	client2.Login(login1, password1)
	resp := client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Message: `{"status":"success"}`})

	t.Logf("Waiting 4 seconds for expiration of access token...")
	time.Sleep(time.Second * 4)
	client1.Get("/status/check", nil)
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Error: auth_token.ErrorCodeTokenExpired, HttpCode: http.StatusUnauthorized})

	client1.RequestRefreshToken()
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Message: `{"status":"success"}`})

	t.Logf("Waiting 2 seconds for prolongation of access token 1...")
	time.Sleep(time.Second * 2)
	client1.Get("/status/check", nil)
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Message: `{"status":"success"}`})
	t.Logf("Waiting 2 seconds for prolongation of access token 2...")
	time.Sleep(time.Second * 2)
	client1.Get("/status/check", nil)
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Message: `{"status":"success"}`})
	t.Logf("Waiting 2 seconds for prolongation of access token 3...")
	time.Sleep(time.Second * 2)
	client1.Get("/status/check", nil)
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Message: `{"status":"success"}`})

	t.Logf("Waiting 6 seconds for expiration of refresh token...")
	time.Sleep(time.Second * 6)
	client1.Get("/status/check", nil)
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Error: auth_token.ErrorCodeTokenExpired, HttpCode: http.StatusUnauthorized})

	client1.RequestRefreshToken(auth_token.ErrorCodeSessionExpired)
	resp = client1.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{Error: auth_token.ErrorCodeTokenExpired, HttpCode: http.StatusUnauthorized})
}
