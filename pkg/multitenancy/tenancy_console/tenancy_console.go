package tenancy_console

import (
	"errors"
	"fmt"
	"strings"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/app_with_multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/app_with_pools"
)

type MultitenancyAppBuilder struct {
	Models   *multitenancy.TenancyDbModels
	Config   app_with_multitenancy.AppConfigI
	App      *app_with_multitenancy.AppWithMultitenancyBase
	SetupApp *app_with_pools.AppWithPoolsBase
}

func (m *MultitenancyAppBuilder) PoolController() pool.PoolController {
	if m.SetupApp != nil {
		return m.SetupApp.Pools().PoolController()
	}
	return m.App.Pools().PoolController()
}

func (m *MultitenancyAppBuilder) NewApp(buildConfig *app_context.BuildConfig) app_context.Context {

	fmt.Println("Using app with tenancies")

	if m.Config == nil {
		m.App = app_with_multitenancy.NewApp(buildConfig, m.Models)
	} else {
		m.App = app_with_multitenancy.NewApp(buildConfig, m.Models, m.Config)
	}
	return m.App
}

func (m *MultitenancyAppBuilder) InitApp(app app_context.Context, configFile string, args []string, configType ...string) error {

	fmt.Println("Setup app with tenancies")

	a, ok := app.(*app_with_multitenancy.AppWithMultitenancyBase)
	if !ok {
		return app.Logger().PushFatalStack("invalid application type", errors.New("failed to cast app to multitenancy app"))
	}
	ctx, err := a.InitWithArgs(configFile, args, configType...)
	if ctx != nil {
		ctx.Close()
	}
	return err
}

func (m *MultitenancyAppBuilder) NewSetupApp(buildConfig *app_context.BuildConfig) app_context.Context {

	fmt.Println("Using app with pools")

	if m.Config == nil {
		m.SetupApp = app_with_pools.New(buildConfig)
	} else {
		m.SetupApp = app_with_pools.New(buildConfig, m.Config)
	}
	return m.SetupApp
}

func (m *MultitenancyAppBuilder) InitSetupApp(app app_context.Context, configFile string, args []string, configType ...string) error {

	if app == nil {
		return errors.New("unexpected nil app")
	}

	fmt.Println("Setup app with pools")

	a, ok := app.(*app_with_pools.AppWithPoolsBase)
	if !ok {
		return app.Logger().PushFatalStack("invalid application type", errors.New("failed to cast app to pools app"))
	}
	ctx, err := a.InitWithArgs(configFile, args, configType...)
	if ctx != nil {
		ctx.Close()
	}
	return err
}

func (m *MultitenancyAppBuilder) HasSetupApp() bool {
	return true
}

func (m *MultitenancyAppBuilder) Tenancy(ctx op_context.Context, id string) (multitenancy.Tenancy, error) {

	idIsDisplay := strings.Contains(id, "/")
	if !idIsDisplay {
		return m.App.Multitenancy().Tenancy(id)
	}

	tenancy, err := m.App.Multitenancy().TenancyController().Find(ctx, id, idIsDisplay)
	if err != nil {
		return nil, err
	}
	if tenancy == nil {
		return nil, errors.New("unknown tenancy")
	}

	return m.App.Multitenancy().Tenancy(tenancy.GetID())
}

type TenancyCommands struct {
	console_tool.Commands[*TenancyCommands]
	MakeController func(app app_context.Context) multitenancy.TenancyController
}

func NewTenancyCommands() *TenancyCommands {
	p := &TenancyCommands{}
	p.Construct(p, "tenancy", "Manage tenancies")
	p.LoadHandlers()
	return p
}

func (p *TenancyCommands) LoadHandlers() {
	p.AddHandlers(Add,
		Find,
		List,
		Activate,
		Deactivate,
		Customer,
		Role,
		Path,
		ShadowPath,
		ChangePoolOrDb,
		Delete,
		AddIpAddress,
		DeleteIpAddress,
		ListIpAddresses,
		DbRole,
		SetPathBlocked,
	)
}

type Handler = console_tool.Handler[*TenancyCommands]

type HandlerBase struct {
	console_tool.HandlerBase[*TenancyCommands]
}

func (b *HandlerBase) Context(data interface{}) (op_context.Context, multitenancy.TenancyController, error) {

	ctx, err := b.HandlerBase.Context(data)
	if err != nil {
		return ctx, nil, err
	}

	a, ok := ctx.App().(*app_with_multitenancy.AppWithMultitenancyBase)
	if !ok {
		panic(fmt.Errorf("invalid application type: %s", errors.New("failed to cast app to multitenancy app")))
	}

	return ctx, a.Multitenancy().TenancyController(), nil
}
