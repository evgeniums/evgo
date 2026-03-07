package api_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/evgeniums/evgo/pkg/admin"
	"github.com/evgeniums/evgo/pkg/admin/admin_api/admin_api_service"
	"github.com/evgeniums/evgo/pkg/api/api_client/rest_api_client"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/api/bare_bones_server"
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/app_with_multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_manager"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/sms/sms_provider_factory"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

var BaseUrl = "http://localhost/api/1.0.0"

type TestContext struct {
	ClientApp         app_context.Context
	ServerApp         app_context.Context
	RestApiClient     *rest_api_client.Client
	ClientOp          op_context.Context
	AdminOp           op_context.Context
	Server            bare_bones_server.Server
	LocalAdminManager *admin.Manager
}

func (t *TestContext) Close() {
	t.ClientOp.Close()
	t.AdminOp.Close()
	t.ClientApp.Close()
	t.ServerApp.Close()
}

func (t *TestContext) Reset() {
	t.ClientOp.Reset()
	t.AdminOp.Reset()
}

func initClient(t *testing.T, g *gin.Engine, testDir string, config string) (app_context.Context, *rest_api_client.Client) {
	app := test_utils.InitDefaultAppContextNoDb(t, testDir, config)

	opCtx := test_utils.SimpleOpContext(app, "prepare")
	restApiClient := test_utils.RestApiTestClient(t, g, BaseUrl)
	restApiClient.Prepare(opCtx)

	client := rest_api_client.New(restApiClient)
	return app, client
}

func initServer(t *testing.T, testDir string, config string, dbModels []interface{}, newDb ...bool) (app_context.Context, *admin.Manager, bare_bones_server.Server) {

	app := test_utils.InitAppContext(t, testDir, dbModels, config, newDb...)

	adminManager := admin.NewManager()
	adminManager.Init(app.Validator())

	var tenancyManager multitenancy.Multitenancy
	appWithTenancy, ok := app.(app_with_multitenancy.AppWithMultitenancy)
	if ok {
		tenancyManager = appWithTenancy.Multitenancy()
	} else {
		tenancyManager = &tenancy_manager.TenancyManager{}
	}

	server := bare_bones_server.New(adminManager, bare_bones_server.Config{SmsProviders: &sms_provider_factory.MockFactory{}})
	require.NoErrorf(t, server.Init(app, tenancyManager), "failed to init server")

	// Workaround for bug in gin engine.
	// Sometimes gin would panic because of number of path parameters cached in pool of gin contexts would mismatch
	// number of parameters in new added routes. We add stub route to ensure that number of params will be enough for most
	// useful routes later.
	ginEngine := test_utils.BBGinEngine(t, server)
	ginEngine.GET("/a/:a/b/:b/c/:c/e/:e/f/:f/g/:g/h/:h/i/:i/j/:j/k/:k")

	adminService := admin_api_service.NewAdminService(adminManager)
	api_server.AddServiceToServer(server.ApiServer(), adminService)

	return app, adminManager, server
}

func InitTest(t *testing.T, packageName string, testDir string, dbModels []interface{}, newDb ...bool) *TestContext {

	ctx := &TestContext{}

	clientConfig := fmt.Sprintf("%s_api_client.jsonc", packageName)
	serverConfig := fmt.Sprintf("%s_api_server.jsonc", packageName)

	ctx.ServerApp, ctx.LocalAdminManager, ctx.Server = initServer(t, testDir, serverConfig, dbModels, newDb...)
	ctx.ClientApp, ctx.RestApiClient = initClient(t, test_utils.BBGinEngine(t, ctx.Server), testDir, clientConfig)

	ctx.ClientOp = test_utils.SimpleOpContext(ctx.ClientApp, t.Name())
	ctx.AdminOp = test_utils.SimpleOpContext(ctx.ServerApp, t.Name())

	// add superadmin for remote admin manager login
	superadmin := "superadmin"
	superpassword := "superpassword"
	if utils.OptionalArg(true, newDb...) {
		user1, err := ctx.LocalAdminManager.Add(ctx.AdminOp, superadmin, superpassword)
		require.NoErrorf(t, err, "failed to add superadmin")
		require.NotNil(t, user1)
	}

	// login with client
	restApiClient, ok := ctx.RestApiClient.Transport().(rest_api_client.RestApiClient)
	require.True(t, ok)
	resp, err := restApiClient.Login(ctx.ClientOp, superadmin, superpassword)
	require.NoErrorf(t, err, "failed to login superadmin")
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.Code())

	ctx.AdminOp.Reset()
	ctx.ClientOp.Reset()

	return ctx
}
