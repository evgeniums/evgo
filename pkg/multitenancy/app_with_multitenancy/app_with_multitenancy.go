package app_with_multitenancy

import (
	"net/http"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/background_worker"
	"github.com/evgeniums/evgo/pkg/customer"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_manager"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/op_context/default_op_context"
	"github.com/evgeniums/evgo/pkg/pubsub/pool_pubsub"
)

type AppWithMultitenancy interface {
	pool_pubsub.AppWithPubsub
	Multitenancy() multitenancy.Multitenancy
}

type AppWithMultitenancyBase struct {
	*pool_pubsub.AppWithPubsubBase
	tenancyManager        multitenancy.Multitenancy
	tenancyManagerBuilder TenancyManagerBuilder
}

func (a *AppWithMultitenancyBase) Multitenancy() multitenancy.Multitenancy {
	return a.tenancyManager
}

type TenancyManagerBuilder = func(app pool_pubsub.AppWithPubsub, ctx op_context.Context) (multitenancy.Multitenancy, error)

type MultitenancyConfigI interface {
	GetTenancyManagerBuilder() TenancyManagerBuilder
}

type MultitenancyConfig struct {
	TenancyManagerBuilder TenancyManagerBuilder
}

func (p *MultitenancyConfig) GetTenancyManagerBuilder() TenancyManagerBuilder {
	return p.TenancyManagerBuilder
}

type AppConfigI interface {
	pool_pubsub.AppConfigI
	MultitenancyConfigI
}

type AppConfig struct {
	pool_pubsub.AppConfig
	MultitenancyConfig
}

func NewApp(buildConfig *app_context.BuildConfig, tenancyDbModels *multitenancy.TenancyDbModels, appConfig ...AppConfigI) *AppWithMultitenancyBase {
	a := &AppWithMultitenancyBase{}
	if len(appConfig) != 0 {
		cfg := appConfig[0]
		a.AppWithPubsubBase = pool_pubsub.NewApp(buildConfig, cfg)

		builder := cfg.GetTenancyManagerBuilder()
		if builder != nil {
			a.tenancyManagerBuilder = builder
		}
	}

	if a.AppWithPubsubBase == nil {
		a.AppWithPubsubBase = pool_pubsub.NewApp(buildConfig)
	}

	if a.tenancyManagerBuilder == nil {

		tenancyManager := tenancy_manager.NewTenancyManager(a.Pools(), a.Pubsub(), tenancyDbModels)

		tenancyManager.SetController(tenancy_manager.DefaultTenancyController(tenancyManager))
		tenancyManager.SetCustomerController(customer.LocalCustomerController())

		a.tenancyManagerBuilder = func(app pool_pubsub.AppWithPubsub, opCtx op_context.Context) (multitenancy.Multitenancy, error) {
			c := opCtx.TraceInMethod("AppWithMultitenancy.Init")
			defer opCtx.TraceOutMethod()

			err := tenancyManager.Init(opCtx, "multitenancy")
			if err != nil {
				msg := "failed to init multitenancy"
				c.SetMessage(msg)
				return nil, opCtx.Logger().PushFatalStack(msg, c.SetError(err))
			}

			return tenancyManager, nil
		}
	}

	return a
}

func (a *AppWithMultitenancyBase) InitWithArgs(configFile string, args []string, configType ...string) (op_context.Context, error) {

	opCtx, err := a.AppWithPubsubBase.InitWithArgs(configFile, args, configType...)
	if err != nil {
		return nil, err
	}

	a.tenancyManager, err = a.tenancyManagerBuilder(a, opCtx)
	if err != nil {
		msg := "failed to build tenancy manager"
		return nil, opCtx.Logger().PushFatalStack(msg, err)
	}

	return opCtx, nil
}

func (a *AppWithMultitenancyBase) Init(configFile string, configType ...string) (op_context.Context, error) {
	return a.InitWithArgs(configFile, nil, configType...)
}

func (a *AppWithMultitenancyBase) Close() {
	if a.tenancyManager != nil && a.tenancyManager.IsMultiTenancy() {
		a.tenancyManager.Close()
	}
	a.AppWithPoolsBase.Close()
}

func BackgroundOpContext(app app_context.Context, tenancy multitenancy.Tenancy, name string) multitenancy.TenancyContext {
	opCtx := multitenancy.NewInitContext(app, app.Logger(), app.Db())
	opCtx.SetName(name)
	errManager := &generic_error.ErrorManagerBase{}
	errManager.Init(http.StatusInternalServerError)
	opCtx.SetErrorManager(errManager)
	origin := default_op_context.NewOrigin(app)
	origin.SetUser(background_worker.ContextUser)
	origin.SetUserType(op_context.AutoUserType)
	opCtx.SetOrigin(origin)
	opCtx.SetTenancy(tenancy)
	return opCtx
}
