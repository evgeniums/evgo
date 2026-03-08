package pool_pubsub

import (
	"context"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool/app_with_pools"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_providers/pubsub_factory"
)

type AppWithPubsub interface {
	app_with_pools.AppWithPools
	Pubsub() PoolPubsub
}

type AppWithPubsubBase struct {
	*app_with_pools.AppWithPoolsBase
	pubsub *PoolPubsubBase
}

func (a *AppWithPubsubBase) Pubsub() PoolPubsub {
	return a.pubsub
}

type PoolPubsubConfigI interface {
	GetPubsubFactory() pubsub_factory.PubsubFactory
}

type PoolPubsubConfig struct {
	PubsubFactory pubsub_factory.PubsubFactory
}

func (p *PoolPubsubConfig) GetPubsubFactory() pubsub_factory.PubsubFactory {
	return p.PubsubFactory
}

type AppConfigI interface {
	app_with_pools.AppConfigI
	PoolPubsubConfigI
}

type AppConfig struct {
	app_with_pools.AppConfig
	PoolPubsubConfig
}

func NewApp(buildConfig *app_context.BuildConfig, appConfig ...AppConfigI) *AppWithPubsubBase {
	a := &AppWithPubsubBase{}
	if len(appConfig) != 0 {
		cfg := appConfig[0]
		a.AppWithPoolsBase = app_with_pools.New(buildConfig, cfg)

		if cfg.GetPubsubFactory() != nil {
			a.pubsub = NewPubsub(cfg.GetPubsubFactory())
		}
	}

	if a.AppWithPoolsBase == nil {
		a.AppWithPoolsBase = app_with_pools.New(buildConfig)
	}
	if a.pubsub == nil {
		a.pubsub = NewPubsub()
	}

	return a
}

func (a *AppWithPubsubBase) InitWithArgs(configFile string, args []string, configType ...string) (op_context.Context, context.Context, error) {

	opCtx, sctx, err := a.AppWithPoolsBase.InitWithArgs(configFile, args, configType...)
	if err != nil {
		return nil, nil, err
	}

	c := opCtx.TraceInMethod("AppWithPubsub.Init")
	defer opCtx.TraceOutMethod()

	err = a.pubsub.Init(a, a.Pools())
	if err != nil {
		msg := "failed to init pubsub"
		c.SetMessage(msg)
		return opCtx, sctx, opCtx.Logger().PushFatalStack(msg, c.SetError(err))
	}

	return opCtx, sctx, nil
}

func (a *AppWithPubsubBase) Init(configFile string, configType ...string) (op_context.Context, context.Context, error) {
	return a.InitWithArgs(configFile, nil, configType...)
}

func (a *AppWithPubsubBase) Close() {
	a.AppWithPoolsBase.Close()
}
