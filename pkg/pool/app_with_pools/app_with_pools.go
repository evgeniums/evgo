package app_with_pools

import (
	"context"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/app_context/app_default"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/op_context/default_op_context"
	"github.com/evgeniums/evgo/pkg/pool"
)

type AppWithPools interface {
	app_context.Context
	Pools() pool.PoolStore
}

type AppWithPoolsBase struct {
	*app_default.Context
	pools *pool.PoolStoreBase
}

func (a *AppWithPoolsBase) Pools() pool.PoolStore {
	return a.pools
}

type AppConfigI interface {
	app_default.AppConfigI
	GetPoolControllerBuilder() func(app app_context.Context) pool.PoolController
}

type AppConfig struct {
	app_default.AppConfig
	PoolControllerBuilder func(app app_context.Context) pool.PoolController
}

func New(buildConfig *app_context.BuildConfig, appConfig ...AppConfigI) *AppWithPoolsBase {
	a := &AppWithPoolsBase{}
	if len(appConfig) != 0 {
		cfg := appConfig[0]
		a.Context = app_default.New(buildConfig, cfg)
		if cfg.GetPoolControllerBuilder() != nil {
			pCfg := &pool.PoolStoreConfig{PoolController: cfg.GetPoolControllerBuilder()(a)}
			a.pools = pool.NewPoolStore(pCfg)
		}
	}
	if a.Context == nil {
		a.Context = app_default.New(buildConfig)
	}
	if a.pools == nil {
		a.pools = pool.NewPoolStore()
	}

	return a
}

func (a *AppWithPoolsBase) InitWithArgs(configFile string, args []string, configType ...string) (op_context.Context, context.Context, error) {

	err := a.Context.InitWithArgs(configFile, args, configType...)
	if err != nil {
		return nil, nil, err
	}

	err = a.InitDB("db")
	if err != nil {
		return nil, nil, a.Logger().PushFatalStack("failed to init database", err)
	}

	opCtx, ctx := default_op_context.NewAppInitContext(a)
	c := opCtx.TraceInMethod("AppWithPools.Init")
	defer opCtx.TraceOutMethod()

	err = a.pools.Init(ctx, "pools")
	if err != nil {
		msg := "failed to init pools"
		c.SetMessage(msg)
		return opCtx, nil, opCtx.Logger().PushFatalStack(msg, c.SetError(err))
	}

	return opCtx, ctx, nil
}

func (a *AppWithPoolsBase) Init(configFile string, configType ...string) (op_context.Context, context.Context, error) {
	return a.InitWithArgs(configFile, nil, configType...)
}
