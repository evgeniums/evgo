package event_dispatcher

import (
	"context"

	"github.com/evgeniums/evgo/pkg/message_queue"
)

type EventWrapper struct {
	*Event
	Context context.Context
}

type EventMq = message_queue.MessageQueue[EventKey, EventWrapper]

type EventSubscriber = message_queue.Subscriber[EventKey, EventWrapper]

type EventConsumer = message_queue.Consumer[EventKey, EventWrapper]

type EventConsumerRegistry = message_queue.AttributeRegistry[EventConsumer]

type EventConsumerFeeder = message_queue.Feeder[EventWrapper]

type EventConsumerQueue = message_queue.RandomAccessQueue[EventKey, EventWrapper]

type EventSubscriberAgregation = message_queue.SubscriberFanIn[EventKey, EventWrapper]

func NewEventSubscriberAggregation() EventSubscriberAgregation {
	return message_queue.NewSubscriberFanIn[EventKey, EventWrapper]()
}
