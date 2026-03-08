package auth_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/evgeniums/evgo/pkg/api/bare_bones_server"
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_manager"
	"github.com/evgeniums/evgo/pkg/signature"
	"github.com/evgeniums/evgo/pkg/signature/user_pubkey"
	"github.com/evgeniums/evgo/pkg/sms"
	"github.com/evgeniums/evgo/pkg/sms/sms_provider_factory"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/user/user_default"
	"github.com/evgeniums/evgo/pkg/user/user_session_default"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/stretchr/testify/require"
)

var _, testBasePath, _, _ = runtime.Caller(0)
var testDir = filepath.Dir(testBasePath)
var assetsDir = filepath.Join(testDir, "assets")

type User = user_default.User

type UserPubKey struct {
	user_pubkey.UserPubkey
}

func dbModels() []interface{} {
	return append([]interface{}{},
		&User{},
		&user_session_default.UserSession{},
		&user_session_default.UserSessionClient{},
		&sms.SmsMessage{},
		&UserPubKey{},
		&signature.MessageSignature{},
	)
}

func initServer(t *testing.T, config ...string) (app_context.Context, *user_session_default.Users, bare_bones_server.Server) {
	app, _ := test_utils.InitAppContext(t, testDir, dbModels(), utils.OptionalArg("auth_test.jsonc", config...))

	users := user_session_default.NewUsers()
	users.Init(app.Validator())

	tenancyManager := &tenancy_manager.TenancyManager{}

	signatureManager := signature.NewSignatureManager()
	err := signatureManager.Init(app.Cfg(), app.Logger(), app.Validator())
	require.NoError(t, err)

	server := bare_bones_server.New(users, bare_bones_server.Config{SmsProviders: &sms_provider_factory.MockFactory{}, SignatureManager: signatureManager})
	require.NoErrorf(t, server.Init(app, tenancyManager), "failed to init auth server")

	return app, users, server
}

func TestInitServer(t *testing.T) {
	app, _, _ := initServer(t)
	app.Close()
}
