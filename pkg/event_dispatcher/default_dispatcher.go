package event_dispatcher

import (
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/evgeniums/evgo/pkg/utils"
)

type DispatcherOptions struct {
	SubscriberBuilder       func() (EventSubscriber, error)
	ConsumerBuilder         func() (EventConsumer, error)
	ConsumerFeederBuilder   func() (EventConsumerFeeder, error)
	ConsumerQueueBuilder    func() (EventConsumerQueue, error)
	ConsumerRegistryBuilder func(maxSelectors int) (EventConsumerRegistry, error)
}

type DispatcherBaseConfig struct {
	message_queue.ConsumerConfig
	INMEM_LEVEL_TRIE bool
}

type DispatcherBase struct {
	generic_error.ErrorsExtenderBase
	DispatcherOptions
	DispatcherBaseConfig

	mq EventMq
}

func (d *DispatcherBase) MakeConsumer() (EventConsumer, error) {
	if d.ConsumerBuilder != nil {
		return d.ConsumerBuilder()
	}

	consumer := message_queue.NewConsumer[EventKey, Event](d.ConsumerConfig)

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

func (d *DispatcherBase) MakeSubscriber() (EventSubscriber, error) {
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

func NewDispatcher(opt ...DispatcherOptions) *DispatcherBase {
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

	d.ErrorsExtenderBase.Init(ErrorDescriptions, ErrorHttpCodes)

	path := object_config.Key(parentConfigPath, utils.OptionalString("event_dispatcher", configPath...))
	err := object_config.LoadLogValidateApp(app, d, path)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load configuration of event dispatcher", err)
	}
	var consumers EventConsumerRegistry
	if d.DispatcherOptions.ConsumerRegistryBuilder != nil {
		consumers, err = d.DispatcherOptions.ConsumerRegistryBuilder(MaxSelectors)
		if err != nil {
			return app.Logger().PushFatalStack("failed to create consumers registry", err)
		}
	}

	if consumers == nil && d.INMEM_LEVEL_TRIE {
		consumers = message_queue.NewSelectorTrie[EventConsumer](MaxSelectors)
	}

	if consumers != nil {
		d.mq = message_queue.NewInmemMq[EventKey, Event](MaxSelectors)
	} else {
		d.mq = message_queue.NewInmemMq(MaxSelectors, consumers)
	}

	return nil
}
