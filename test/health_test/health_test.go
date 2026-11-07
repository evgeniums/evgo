package health_test

import (
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/evgeniums/evgo/pkg/api/bare_bones_server"
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_manager"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/user/user_default"
	"github.com/evgeniums/evgo/pkg/user/user_session_default"
	"github.com/stretchr/testify/require"
)

var _, testBasePath, _, _ = runtime.Caller(0)
var testDir = filepath.Dir(testBasePath)

func initServer(t *testing.T) (app_context.Context, bare_bones_server.Server) {
	models := []interface{}{
		&user_default.User{},
		&user_session_default.UserSession{},
		&user_session_default.UserSessionClient{},
	}
	app, _ := test_utils.InitAppContext(t, testDir, models, "health_test.jsonc")

	users := user_session_default.NewUsers()
	users.Init(app.Validator())

	server := bare_bones_server.New(users, bare_bones_server.Config{WithHealthService: true})
	require.NoError(t, server.Init(app, &tenancy_manager.TenancyManager{}), "failed to init server")

	return app, server
}

func TestHealth(t *testing.T) {
	app, server := initServer(t)
	defer app.Close()

	client := test_utils.NewHttpClient(t, test_utils.BBGinEngine(t, server))

	// public: no token, and the config has no "noauth" method of its own
	resp := client.Get("/health/live", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{
		HttpCode: http.StatusOK,
		Message:  `{"status":"running"}`})

	resp = client.Get("/health/ready", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{
		HttpCode: http.StatusOK,
		Message:  `{"status":"ready"}`})

	// the rest of the API keeps the default schema
	resp = client.Get("/status/logged", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{
		HttpCode: http.StatusUnauthorized,
		Error:    "unauthorized"})

	// readiness is cached: killing the DB is not noticed until the TTL (1s) expires, liveness never is
	app.Db().Close()
	resp = client.Get("/health/live", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{
		HttpCode: http.StatusOK,
		Message:  `{"status":"running"}`})

	time.Sleep(1200 * time.Millisecond)
	resp = client.Get("/health/ready", nil)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{
		HttpCode: http.StatusServiceUnavailable,
		Error:    generic_error.ErrorCodeUnavailable})
}
