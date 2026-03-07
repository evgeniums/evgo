package test_utils

import (
	"net/http"
	"os"
	"testing"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/app_context/app_default"
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/op_context/default_op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/stretchr/testify/require"
)

type AppBuilder = func(t *testing.T, buildConfig *app_context.BuildConfig) app_context.Context
type AppInitializer = func(t *testing.T, app app_context.Context, configFile string, args []string, configType ...string) error

func DefaultAppBuilder(t *testing.T, buildConfig *app_context.BuildConfig) app_context.Context {
	app := app_default.New(buildConfig)
	return app
}

func DefaultAppInitializer(t *testing.T, app app_context.Context, configFile string, args []string, configType ...string) error {
	a, ok := app.(*app_default.Context)
	require.True(t, ok)
	return a.InitWithArgs(configFile, args, configType...)
}

var appBuilder = DefaultAppBuilder
var appInitializer = DefaultAppInitializer

func SetAppHandlers(builder AppBuilder, initializer AppInitializer) {
	appBuilder = builder
	appInitializer = initializer
}

func InitDefaultAppContextNoDb(t *testing.T, testDir string, config ...string) app_context.Context {

	SetTesting(t)

	configFile := utils.OptionalArg(AssetsFilePath(testDir, "test_config.json"), config...)
	if !utils.FileExists(configFile) {
		configFile = AssetsFilePath(testDir, configFile)
	}

	app := DefaultAppBuilder(t, nil)
	require.NoErrorf(t, DefaultAppInitializer(t, app, configFile, nil), "failed to init application context")

	return app
}

func InitAppContextNoDb(t *testing.T, testDir string, config ...string) app_context.Context {

	SetTesting(t)

	configFile := utils.OptionalArg(AssetsFilePath(testDir, "test_config.json"), config...)
	if !utils.FileExists(configFile) {
		configFile = AssetsFilePath(testDir, configFile)
	}

	app := appBuilder(t, nil)
	require.NoErrorf(t, appInitializer(t, app, configFile, nil), "failed to init application context")

	return app
}

func InitDbModels(t *testing.T, testDir string, dbModels []interface{}, config ...string) {

	configFile := utils.OptionalString(AssetsFilePath(testDir, "test_config.json"), config...)
	if !utils.FileExists(configFile) {
		configFile = AssetsFilePath(testDir, configFile)
	}

	SetupGormDB(t)
	dbPaths := SqliteDatabasesPath()
	t.Logf("Sqlite DB folder: %s", dbPaths)
	if dbPaths != "" && dbPaths != "/" {
		if utils.FileExists(dbPaths) {
			require.NoErrorf(t, os.RemoveAll(dbPaths), "failed to remove files from db folder")
		}
		require.NoErrorf(t, os.MkdirAll(dbPaths, os.ModePerm), "failed to create db folder")
	}

	app := DefaultAppBuilder(t, nil)
	require.NoErrorf(t, DefaultAppInitializer(t, app, configFile, nil), "failed to init application context")
	a, ok := app.(*app_default.Context)
	require.True(t, ok)
	require.NoErrorf(t, a.InitDB("db"), "failed to init database")

	CreateDbModels(t, app, dbModels)
	a.Close()
}

func InitAppContext(t *testing.T, testDir string, dbModels []interface{}, config string, newDb ...bool) app_context.Context {

	SetTesting(t)

	if utils.OptionalArg(true, newDb...) {
		InitDbModels(t, testDir, dbModels, config)
	} else {
		SetupGormDB(t)
	}

	configFile := utils.OptionalString(AssetsFilePath(testDir, "test_config.json"), config)
	if !utils.FileExists(configFile) {
		configFile = AssetsFilePath(testDir, configFile)
	}

	app := appBuilder(t, nil)
	require.NoErrorf(t, appInitializer(t, app, configFile, nil), "failed to init application context")
	a, ok := app.(app_default.WithInitGormDb)
	require.True(t, ok)
	require.NoErrorf(t, a.InitDB("db"), "failed to init database")
	// app.Db().EnableDebug(true)

	return app
}

func prepareOpContext(ctx op_context.Context, name string) {
	ctx.SetName(name)
	errManager := &generic_error.ErrorManagerBase{}
	errManager.Init(http.StatusBadRequest)
	ctx.SetErrorManager(errManager)
}

func SimpleOpContext(app app_context.Context, name string) op_context.Context {
	ctx := default_op_context.NewContext()
	ctx.Init(app, app.Logger(), app.Db())
	prepareOpContext(ctx, name)

	origin := default_op_context.NewOrigin(app)
	origin.SetUserType("simple_op_context")
	ctx.SetOrigin(origin)

	return ctx
}

func MultitenancyOpContext(app app_context.Context, name string) multitenancy.TenancyContext {
	ctx := multitenancy.NewInitContext(app, app.Logger(), app.Db())
	prepareOpContext(ctx, name)

	origin := default_op_context.NewOrigin(app)
	origin.SetUserType("multitenancy_op_context")
	ctx.SetOrigin(origin)

	return ctx
}

func UserOpContext(app app_context.Context, name string, user auth.User, tenancy ...multitenancy.Tenancy) auth.UserContext {
	ctx := &auth.TenancyUserContext{}
	baseCtx := default_op_context.NewContext()
	baseCtx.Init(app, app.Logger(), app.Db())
	ctx.Construct(baseCtx)
	prepareOpContext(ctx, name)
	ctx.SetAuthUser(user)
	t := utils.OptionalArg(nil, tenancy...)
	if t != nil {
		ctx.SetTenancy(t)
	}
	origin := default_op_context.NewOrigin(app)
	origin.SetUser(user.Display())
	ctx.SetOrigin(origin)
	return ctx
}

func SetTesting(t *testing.T) {
	app_context.Testing = t
}
