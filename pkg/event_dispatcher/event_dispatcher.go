package event_dispatcher

import (
	"context"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/evgeniums/evgo/pkg/utils"
)

type Dispatcher interface {
	generic_error.ErrorsExtender

	SubscriberConfig() message_queue.ConsumerConfig

	Subscribe(sctx context.Context, key EventKey, subscriber EventSubscriber) error
	Publish(sctx context.Context, event EventKey) error
}

type DispatcherConfig struct {
	LEVEL_TRIE bool
}

type DispatcherBase struct {
	DispatcherConfig
	generic_error.ErrorsExtenderBase
	message_queue.ConsumerConfig

	mq EventMq
}

func New() *DispatcherBase {
	m := &DispatcherBase{}
	return m
}

func (d *DispatcherBase) Config() interface{} {
	return &d.DispatcherConfig
}

func (d *DispatcherBase) SubscriberConfig() message_queue.ConsumerConfig {
	return d.ConsumerConfig
}

func (d *DispatcherBase) Init(app app_context.Context, parentConfigPath string, configPath ...string) error {

	d.ErrorsExtenderBase.Init(ErrorDescriptions, ErrorHttpCodes)

	path := object_config.Key(parentConfigPath, utils.OptionalString("event_dispatcher", configPath...))
	err := object_config.LoadLogValidateApp(app, d, path)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load configuration of event dispatcher", err)
	}

	d.mq = message_queue.NewMessageQueue[EventKey, Event](MaxSelectors, d.LEVEL_TRIE)

	return nil
}
