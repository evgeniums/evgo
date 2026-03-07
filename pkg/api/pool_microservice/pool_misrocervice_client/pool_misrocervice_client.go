package pool_misrocervice_client

import (
	"context"
	"fmt"

	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/http_request"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pool/app_with_pools"
	"github.com/evgeniums/evgo/pkg/utils"
)

type PoolServiceClient interface {
	api_client.Client
	InitForPoolService(httpClient *http_request.HttpClient, service *pool.PoolServiceBinding, clientAgent ...string) error
}

type PoolMicroserviceClientConfig struct {
	POOL_SERVICE_ROLE string `validate:"required" vmessage:"Service role in the pool must be specified"`
}

type PoolMicroserviceClient struct {
	http_request.WithHttpClient
	PoolMicroserviceClientConfig
	PoolServiceClient

	overridePoolName string
}

func NewPoolMicroserviceClient(defaultRole string, client ...PoolServiceClient) *PoolMicroserviceClient {
	p := &PoolMicroserviceClient{}
	if len(client) != 0 {
		p.PoolServiceClient = client[0]
	} else {
		p.WithHttpClient.Construct()
		p.PoolServiceClient = &RestApiPoolServiceClient{}
	}
	p.POOL_SERVICE_ROLE = defaultRole
	return p
}

func (p *PoolMicroserviceClient) Config() interface{} {
	return &p.PoolMicroserviceClientConfig
}

func AppUserAgent(app app_context.Context) string {
	userAgent := fmt.Sprintf("%s/%s/%s", app.Hostname(), app.Application(), app.AppInstance())
	return userAgent
}

func (p *PoolMicroserviceClient) SetOverridePool(poolName string) {
	p.overridePoolName = poolName
}

func (p *PoolMicroserviceClient) Init(app app_with_pools.AppWithPools, configPath ...string) error {

	// load config
	path := utils.OptionalString("microservice_client", configPath...)
	err := object_config.LoadLogValidate(app.Cfg(), app.Logger(), app.Validator(), p, path)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load configuration of microservice api client", err)
	}

	// TODO use generic client, do not bind to HTTP
	// init http client
	err = p.WithHttpClient.Init(app.Cfg(), app.Logger(), app.Validator(), path)
	if err != nil {
		return app.Logger().PushFatalStack("failed to init http client in microservice api client", err)
	}

	// find pool
	var poool pool.Pool
	if p.overridePoolName != "" {
		poool, err = app.Pools().PoolByName(p.overridePoolName)
		if err != nil {
			return app.Logger().PushFatalStack("pool not found for microservice api client", err, logger.Fields{"pool": p.overridePoolName})
		}
	} else {
		poool, err = app.Pools().SelfPool()
		if err != nil {
			return app.Logger().PushFatalStack("self pool must be specified for microservice api client", err)
		}
	}

	// find service for role
	service, err := poool.Service(p.POOL_SERVICE_ROLE)
	if err != nil {
		return app.Logger().PushFatalStack("failed to find service with specified role", err, logger.Fields{"role": p.POOL_SERVICE_ROLE})
	}

	// init client form service data
	err = p.InitForPoolService(p.HttpClient(), service, AppUserAgent(app))
	if err != nil {
		return app.Logger().PushFatalStack("failed to init microservice api client with pool service configuration", err)
	}

	p.SetPropagateAuthUser(true)
	p.SetPropagateContextId(true)

	// done
	return nil
}

func (p *PoolMicroserviceClient) SetService(ctx op_context.Context, service *pool.PoolServiceBinding) error {

	c := ctx.TraceInMethod("PoolMicroserviceClient.SetService")
	defer ctx.TraceOutMethod()

	// init client form service data
	err := p.InitForPoolService(p.HttpClient(), service, AppUserAgent(ctx.App()))
	if err != nil {
		c.SetMessage("failed to init microservice api client with pool service configuration")
		return c.SetError(err)
	}

	// done
	return nil
}

func (p *PoolMicroserviceClient) SetPropagateAuthUser(val bool) {
	p.PoolServiceClient.SetPropagateAuthUser(val)
}

func (p *PoolMicroserviceClient) SetPropagateContextId(val bool) {
	p.PoolServiceClient.SetPropagateContextId(val)
}

func (p *PoolMicroserviceClient) Shutdown(ctx context.Context) error {
	return p.WithHttpClient.Shutdown(ctx)
}
