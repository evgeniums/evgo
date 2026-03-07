package pubsub_factory

import (
	"errors"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/message"
	"github.com/evgeniums/evgo/pkg/message/message_json"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pubsub"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_providers/pubsub_inmem"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_providers/pubsub_redis"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_subscriber"
	"github.com/evgeniums/evgo/pkg/utils"
)

const SingletonInmemProvider string = "singleton_inmem"

type PubsubConfigI interface {
	GetPoolService() *pool.PoolServiceBinding
	GetConfigKeyPath() string
}

type PubsubConfig struct {
	PoolService   *pool.PoolServiceBinding
	ConfigKeyPath string
}

func (p *PubsubConfig) GetPoolService() *pool.PoolServiceBinding {
	return p.PoolService
}

func (p *PubsubConfig) GetConfigKeyPath() string {
	return p.ConfigKeyPath
}

type PubsubFactory interface {
	MakePublisher(app app_context.Context, config ...PubsubConfigI) (pubsub.Publisher, error)
	MakeSubscriber(app app_context.Context, config ...PubsubConfigI) (pubsub_subscriber.Subscriber, error)
}

var singletonInmem map[string]*pubsub_inmem.PubsubInmem

type PubsubFactoryBase struct {
	serializer message.Serializer
	inmems     map[string]*pubsub_inmem.PubsubInmem
}

func splitConfig(app app_context.Context, config ...PubsubConfigI) (*pool.PoolServiceBinding, string) {
	configPath := "pubsub"
	var poolService *pool.PoolServiceBinding
	if len(config) != 0 {
		configPath = config[0].GetConfigKeyPath()
		poolService = config[0].GetPoolService()
	}
	return poolService, configPath
}

func provider(app app_context.Context, poolService *pool.PoolServiceBinding, configPath string) string {

	if poolService != nil {
		return poolService.Provider()
	}

	providerKey := object_config.Key(configPath, "provider")
	provider := app.Cfg().GetString(providerKey)
	return provider
}

func (p *PubsubFactoryBase) MakeInmemPubsub(app app_context.Context, poolService *pool.PoolServiceBinding) (*pubsub_inmem.PubsubInmem, error) {

	inmem, found := p.inmems[poolService.DbName()]
	if found {
		return inmem, nil
	}

	inmem = pubsub_inmem.New(app, p.serializer)
	p.inmems[poolService.DbName()] = inmem

	return inmem, nil
}

func ResetSingletonInmemPubsub() {
	singletonInmem = nil
}

func MakeSingletonInmemPubsub(serializer message.Serializer, app app_context.Context, poolService *pool.PoolServiceBinding) (*pubsub_inmem.PubsubInmem, error) {

	if singletonInmem == nil {
		singletonInmem = make(map[string]*pubsub_inmem.PubsubInmem)
	}

	inmem, found := singletonInmem[poolService.DbName()]
	if found {
		return inmem, nil
	}

	inmem = pubsub_inmem.New(app, serializer)
	singletonInmem[poolService.DbName()] = inmem

	return inmem, nil
}

func (p *PubsubFactoryBase) MakePublisher(app app_context.Context, config ...PubsubConfigI) (pubsub.Publisher, error) {

	poolService, configPath := splitConfig(app, config...)
	provider := provider(app, poolService, configPath)
	if provider == pubsub_redis.Provider {
		publisher := pubsub_redis.NewPublisher(p.serializer)
		err := initRedis(app, &publisher.RedisClient, poolService, configPath)
		if err != nil {
			return nil, err
		}
		return publisher, nil
	} else if provider == pubsub_inmem.Provider {
		return p.MakeInmemPubsub(app, poolService)
	} else if provider == SingletonInmemProvider {
		return MakeSingletonInmemPubsub(p.serializer, app, poolService)
	}

	return nil, errors.New("unknown pubsub publisher provider")
}

func initRedis(app app_context.Context, r *pubsub_redis.RedisClient, poolService *pool.PoolServiceBinding, configPath string) error {

	if poolService == nil {
		return r.Init(app.Cfg(), app.Logger(), app.Validator(), configPath)
	}

	cfg := &pubsub_redis.RedisConfig{}
	cfg.Host = poolService.PrivateHost()
	cfg.Port = poolService.PrivatePort()
	cfg.Password = poolService.Secret1()
	db := 0
	if poolService.DbName() != "" {
		dbU, err := utils.StrToUint32(poolService.DbName())
		if err != nil {
			return errors.New("invalid number of redis database")
		}
		db = int(dbU)
	}
	cfg.Db = db
	return r.InitWithConfig(app.Logger(), cfg)
}

func (p *PubsubFactoryBase) MakeSubscriber(app app_context.Context, config ...PubsubConfigI) (pubsub_subscriber.Subscriber, error) {

	poolService, configPath := splitConfig(app, config...)
	provider := provider(app, poolService, configPath)

	if provider == pubsub_redis.Provider {
		subsciber := pubsub_redis.NewSubscriber(app, p.serializer)
		err := initRedis(app, &subsciber.RedisClient, poolService, configPath)
		if err != nil {
			return nil, err
		}
		return subsciber, nil
	} else if provider == pubsub_inmem.Provider {
		return p.MakeInmemPubsub(app, poolService)
	} else if provider == SingletonInmemProvider {
		return MakeSingletonInmemPubsub(p.serializer, app, poolService)
	}

	return nil, errors.New("unknown pubsub subscriber provider")
}

func DefaultPubsubFactory(serializer ...message.Serializer) PubsubFactory {
	f := &PubsubFactoryBase{}
	f.serializer = utils.OptionalArg(message.Serializer(message_json.Serializer), serializer...)
	f.inmems = make(map[string]*pubsub_inmem.PubsubInmem)
	return f
}
