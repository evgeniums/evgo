package default_event_dispatcher

import (
	"context"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/event_dispatcher"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type DispatcherOptions struct {
	SubscriberBuilder       func() (event_dispatcher.EventSubscriber, error)
	ConsumerBuilder         func() (event_dispatcher.EventConsumer, error)
	ConsumerFeederBuilder   func() (event_dispatcher.EventConsumerFeeder, error)
	ConsumerQueueBuilder    func() (event_dispatcher.EventConsumerQueue, error)
	ConsumerRegistryBuilder func(maxSelectors int) (event_dispatcher.EventConsumerRegistry, error)
}

type DispatcherBaseConfig struct {
	message_queue.ConsumerConfig
	INMEM_LEVEL_TRIE bool
}

type DispatcherBase struct {
	generic_error.ErrorsExtenderBase
	DispatcherOptions
	DispatcherBaseConfig

	mq event_dispatcher.EventMq
}

func (d *DispatcherBase) MakeConsumer() (event_dispatcher.EventConsumer, error) {
	if d.ConsumerBuilder != nil {
		return d.ConsumerBuilder()
	}

	consumer := message_queue.NewConsumer[event_dispatcher.EventKey, event_dispatcher.Event](d.ConsumerConfig)

	if d.ConsumerFeederBuilder != nil {
		feeder, err := d.ConsumerFeederBuilder()
		if err != nil {
			return nil, err
		}
		consumer.SetFeeder(feeder)
	}

	if d.ConsumerQueueBuilder != nil {
		queue, err := d.ConsumerQueueBuilder()
		if err != nil {
			return nil, err
		}
		consumer.SetQueue(queue)
	}

	return consumer, nil
}

func (d *DispatcherBase) MakeSubscriber() (event_dispatcher.EventSubscriber, error) {
	if d.SubscriberBuilder != nil {
		return d.SubscriberBuilder()
	}

	consumer, err := d.MakeConsumer()
	if err != nil {
		return nil, err
	}

	return message_queue.NewSubscriber(consumer), nil
}

func (d *DispatcherBase) SubscriberConfig() message_queue.ConsumerConfig {
	return d.ConsumerConfig
}

func New(opt ...DispatcherOptions) *DispatcherBase {
	m := &DispatcherBase{}
	if len(opt) != 0 {
		m.DispatcherOptions = opt[0]
	}
	return m
}

func (d *DispatcherBase) Config() interface{} {
	return &d.DispatcherBaseConfig
}

func (d *DispatcherBase) Init(app app_context.Context, parentConfigPath string, configPath ...string) error {

	d.ErrorsExtenderBase.Init(event_dispatcher.ErrorDescriptions, event_dispatcher.ErrorHttpCodes)

	path := object_config.Key(parentConfigPath, utils.OptionalString("event_dispatcher", configPath...))
	err := object_config.LoadLogValidateApp(app, d, path)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load configuration of event dispatcher", err)
	}
	var consumers event_dispatcher.EventConsumerRegistry
	if d.DispatcherOptions.ConsumerRegistryBuilder != nil {
		consumers, err = d.DispatcherOptions.ConsumerRegistryBuilder(event_dispatcher.MaxSelectors)
		if err != nil {
			return app.Logger().PushFatalStack("failed to create consumers registry", err)
		}
	}

	if consumers == nil && d.INMEM_LEVEL_TRIE {
		consumers = message_queue.NewSelectorTrie[event_dispatcher.EventConsumer](event_dispatcher.MaxSelectors)
	}

	if consumers != nil {
		d.mq = message_queue.NewInmemMq[event_dispatcher.EventKey, event_dispatcher.Event](event_dispatcher.MaxSelectors)
	} else {
		d.mq = message_queue.NewInmemMq(event_dispatcher.MaxSelectors, consumers)
	}

	return nil
}

func (d *DispatcherBase) InitWithCtx(sctx context.Context, cfg DispatcherBaseConfig, opt DispatcherOptions) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("EventDispatcher.Init")
	defer ctx.TraceOutMethod()
	var err error

	d.ErrorsExtenderBase.Init(event_dispatcher.ErrorDescriptions, event_dispatcher.ErrorHttpCodes)
	d.DispatcherBaseConfig = cfg
	d.DispatcherOptions = opt

	var consumers event_dispatcher.EventConsumerRegistry
	if d.DispatcherOptions.ConsumerRegistryBuilder != nil {
		consumers, err = d.DispatcherOptions.ConsumerRegistryBuilder(event_dispatcher.MaxSelectors)
		if err != nil {
			c.SetMessage("failed to create consumers registry")
			return c.SetError(err)
		}
	}

	if consumers == nil && d.INMEM_LEVEL_TRIE {
		consumers = message_queue.NewSelectorTrie[event_dispatcher.EventConsumer](event_dispatcher.MaxSelectors)
	}

	if consumers != nil {
		d.mq = message_queue.NewInmemMq[event_dispatcher.EventKey, event_dispatcher.Event](event_dispatcher.MaxSelectors)
	} else {
		d.mq = message_queue.NewInmemMq(event_dispatcher.MaxSelectors, consumers)
	}

	return nil
}
