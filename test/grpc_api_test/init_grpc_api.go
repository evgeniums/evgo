package grpc_api

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/evgeniums/go-utils/pkg/api/api_server"
	"github.com/evgeniums/go-utils/pkg/api/api_server/grpc_api_server"
	"github.com/evgeniums/go-utils/pkg/api/bare_bones_server"
	"github.com/evgeniums/go-utils/pkg/app_context"
	"github.com/evgeniums/go-utils/pkg/multitenancy/tenancy_manager"
	"github.com/evgeniums/go-utils/pkg/signature"
	"github.com/evgeniums/go-utils/pkg/sms/sms_provider_factory"
	"github.com/evgeniums/go-utils/pkg/test_utils"
	"github.com/evgeniums/go-utils/pkg/user/user_default"
	"github.com/evgeniums/go-utils/pkg/user/user_session_default"
	"github.com/evgeniums/go-utils/pkg/utils"
	"github.com/stretchr/testify/require"
)

var _, testBasePath, _, _ = runtime.Caller(0)
var testDir = filepath.Dir(testBasePath)

type User = user_default.User

func dbModels() []interface{} {
	return append([]interface{}{},
		&User{},
		&user_session_default.UserSession{},
		&user_session_default.UserSessionClient{},
	)
}

func InitServer(t *testing.T, config ...string) (app_context.Context, *user_session_default.Users, bare_bones_server.Server) {

	// TODO use relative path
	test_utils.SqliteFolder = "/Users/user1/projects/whitemgo/workspace/test_data"

	app := test_utils.InitAppContext(t, testDir, dbModels(), utils.OptionalArg("grpc_api_server.jsonc", config...))

	users := user_session_default.NewUsers()
	users.Init(app.Validator())

	tenancyManager := &tenancy_manager.TenancyManager{}

	signatureManager := signature.NewSignatureManager()
	err := signatureManager.Init(app.Cfg(), app.Logger(), app.Validator())
	require.NoError(t, err)

	buildApiServer := func() api_server.Server {
		return grpc_api_server.NewServer()
	}

	server := bare_bones_server.New(users,
		bare_bones_server.Config{ServerBuilder: buildApiServer,
			ServerConfigPath: "grpc_api_server",
			SmsProviders:     &sms_provider_factory.MockFactory{},
			SignatureManager: signatureManager,
		})
	err = server.Init(app, tenancyManager)
	if err != nil {
		app.Logger().CheckFatalStack(app.Logger())
	}
	require.NoErrorf(t, err, "failed to init server")

	return app, users, server
}
