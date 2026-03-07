package pool_pubsub

import (
	"context"
	"errors"
	"fmt"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/pubsub"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_providers/pubsub_factory"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_subscriber"
)

type PoolPubsub interface {
	Shutdown(ctx context.Context) error

	PublishSelfPool(topicName string, msg interface{}) error
	PublishPools(topicName string, msg interface{}, poolIds ...string) error

	SubscribeSelfPool(ctx op_context.Context, topic pubsub_subscriber.Topic) (string, error)
	UnsubscribeSelfPool(topicName string)

	SubscribePools(ctx op_context.Context, topic pubsub_subscriber.Topic, poolIds ...string) (map[string]string, error)
	UnsubscribePools(topicName string, poolIds ...string)
}

type PoolPubsubBase struct {
	factory            pubsub_factory.PubsubFactory
	selfPoolSubscriber pubsub_subscriber.Subscriber
	publishers         map[string]pubsub.Publisher
	selfPoolPublisher  pubsub.Publisher
	subscribers        map[string]pubsub_subscriber.Subscriber
}

func NewPubsub(factory ...pubsub_factory.PubsubFactory) *PoolPubsubBase {
	p := &PoolPubsubBase{}
	if len(factory) != 0 {
		p.factory = factory[0]
	}
	if p.factory == nil {
		p.factory = pubsub_factory.DefaultPubsubFactory()
	}
	p.subscribers = make(map[string]pubsub_subscriber.Subscriber)
	p.publishers = make(map[string]pubsub.Publisher)
	return p
}

func (p *PoolPubsubBase) Init(app app_context.Context, pools pool.PoolStore) error {

	makePublisher := func(poo pool.Pool) (pubsub.Publisher, *pubsub_factory.PubsubConfig, error) {

		fields := db.Fields{"pool_id": poo.GetID(), "pool_name": poo.Name()}

		service, err := poo.Service(pool.TypePubsub)
		if err != nil {
			return nil, nil, app.Logger().PushFatalStack("failed to find pubsub service in pool", err, fields)
		}
		if !service.IsActive() {
			fields["service_name"] = service.ServiceName
			app.Logger().Warn("Pubsub skipped for inactive service", fields)
			return nil, nil, nil
		}
		cfg := &pubsub_factory.PubsubConfig{PoolService: service}
		publisher, err := p.factory.MakePublisher(app, cfg)
		if err != nil {
			return nil, nil, app.Logger().PushFatalStack("failed to make pubsub publisher for pool", err, fields)
		}
		p.publishers[poo.GetID()] = publisher
		app.Logger().Info("Pubsub publisher connected", fields)
		return publisher, cfg, nil
	}

	makeSubscriber := func(poo pool.Pool, cfg *pubsub_factory.PubsubConfig) (pubsub_subscriber.Subscriber, error) {

		fields := db.Fields{"pool_id": poo.GetID(), "pool_name": poo.Name()}

		subscriber, err := p.factory.MakeSubscriber(app, cfg)
		if err != nil {
			return nil, app.Logger().PushFatalStack("failed to make pubsub subscriber for pool", err, fields)
		}
		p.subscribers[poo.GetID()] = subscriber
		app.Logger().Info("Pubsub subscriber connected", fields)
		return subscriber, nil
	}

	selfPool, err := pools.SelfPool()
	if err == nil {

		if !selfPool.IsActive() {
			fields := db.Fields{"pool_id": selfPool.GetID(), "pool_name": selfPool.Name()}
			app.Logger().Warn("Pubsub skipped for inactive pool", fields)
			return nil
		}

		var cfg *pubsub_factory.PubsubConfig
		p.selfPoolPublisher, cfg, err = makePublisher(selfPool)
		if err != nil {
			return app.Logger().PushFatalStack("failed to make pubsub publisher in self pool", err)
		}
		if cfg != nil {
			p.selfPoolSubscriber, err = makeSubscriber(selfPool, cfg)
			if err != nil {
				return app.Logger().PushFatalStack("failed to make pubsub subscriber in self pool", err)
			}
		}
		return nil
	}

	for _, pool := range pools.Pools() {
		if pool.IsActive() {
			_, cfg, err := makePublisher(pool)
			if err != nil {
				return err
			}
			if cfg != nil {
				_, err = makeSubscriber(pool, cfg)
				if err != nil {
					return err
				}
			}
		} else {
			fields := db.Fields{"pool_id": pool.GetID(), "pool_name": pool.Name()}
			app.Logger().Warn("Pubsub skipped for inactive pool", fields)
		}
	}

	return nil
}

func (p *PoolPubsubBase) Shutdown(ctx context.Context) error {
	var err error
	if p.selfPoolSubscriber != nil {
		err1 := p.selfPoolSubscriber.Shutdown(ctx)
		if err1 != nil {
			err = err1
		}
	}
	for _, publisher := range p.publishers {
		err1 := publisher.Shutdown(ctx)
		if err1 != nil {
			err = err1
		}
	}
	return err
}

func (p *PoolPubsubBase) PublishSelfPool(topicName string, msg interface{}) error {
	if p.selfPoolPublisher == nil {
		return errors.New("self publisher not set")
	}
	return p.selfPoolPublisher.Publish(topicName, msg)
}

func (p *PoolPubsubBase) PublishPools(topicName string, msg interface{}, poolIds ...string) error {
	if len(poolIds) == 0 {
		// publish to all pools
		for poolId, publisher := range p.publishers {
			err := publisher.Publish(topicName, msg)
			if err != nil {
				return fmt.Errorf("failed to publish to %s pool", poolId)
			}
		}
	} else {
		// publish to specific pools
		for _, poolId := range poolIds {
			publisher, ok := p.publishers[poolId]
			if ok {
				err := publisher.Publish(topicName, msg)
				if err != nil {
					return fmt.Errorf("failed to publish to %s pool", poolId)
				}
			}
		}
	}
	return nil
}

func (p *PoolPubsubBase) SubscribeSelfPool(ctx op_context.Context, topic pubsub_subscriber.Topic) (string, error) {

	c := ctx.TraceInMethod("PoolPubsub.SubscribeSelfPool", logger.Fields{"topic": topic.Name(), "app": ctx.App().Application(), "app_instance": ctx.App().AppInstance()})
	defer ctx.TraceOutMethod()

	if p.selfPoolSubscriber == nil {
		return "", c.SetErrorStr("self pool subscriber not set")
	}
	subscriptionId, err := p.selfPoolSubscriber.Subscribe(topic)
	if err != nil {
		return "", c.SetError(err)
	}
	c.SetLoggerField("subscription_id", subscriptionId)
	c.Logger().Info("topic was subscribed to self pool")
	return subscriptionId, nil
}

func (p *PoolPubsubBase) UnsubscribeSelfPool(topicName string) {
	if p.selfPoolSubscriber != nil {
		p.selfPoolSubscriber.Unsubscribe(topicName)
	}
}

func (p *PoolPubsubBase) SubscribePools(ctx op_context.Context, topic pubsub_subscriber.Topic, poolIds ...string) (map[string]string, error) {

	c := ctx.TraceInMethod("PoolPubsub.SubscribePools", logger.Fields{"topic": topic.Name(), "app": ctx.App().Application(), "app_instance": ctx.App().AppInstance()})
	defer ctx.TraceOutMethod()

	poolSubscriptions := make(map[string]string)

	if len(poolIds) == 0 {
		// subscribe to all pools
		for poolId, subscriber := range p.subscribers {
			c.SetLoggerField("pool_id", poolId)
			subscriptionId, err := subscriber.Subscribe(topic)
			if err != nil {
				c.SetMessage("failed to subscribe topic to pool")
				return nil, c.SetError(err)
			}
			c.SetLoggerField("subscription_id", subscriptionId)
			poolSubscriptions[poolId] = subscriptionId
			c.Logger().Info("topic was subscribed to pool")
		}
	} else {
		// subscribe to specific pools
		for _, poolId := range poolIds {
			c.SetLoggerField("pool_id", poolId)
			subscriber, ok := p.subscribers[poolId]
			if ok {
				subscriptionId, err := subscriber.Subscribe(topic)
				if err != nil {
					c.SetMessage("failed to subscribe topic to pool")
					return nil, c.SetError(err)
				}
				c.SetLoggerField("subscription_id", subscriptionId)
				poolSubscriptions[poolId] = subscriptionId
				c.Logger().Info("topic was subscribed to pool")
			} else {
				c.Logger().Warn("pubsub subscriber not found in pool")
			}
		}
	}
	return poolSubscriptions, nil
}

func (p *PoolPubsubBase) UnsubscribePools(topicName string, poolIds ...string) {
	if len(poolIds) == 0 {
		// unsubscribe from all pools
		for _, subscriber := range p.subscribers {
			subscriber.Unsubscribe(topicName)
		}
	} else {
		// unsubscribe from specific pools
		for _, poolId := range poolIds {
			subscriber, ok := p.subscribers[poolId]
			if ok {
				subscriber.Unsubscribe(topicName)
			}
		}
	}
}
