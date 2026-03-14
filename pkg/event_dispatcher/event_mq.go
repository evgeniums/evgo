package event_dispatcher

import "github.com/evgeniums/evgo/pkg/message_queue"

type EventMq = message_queue.MessageQueue[EventKey, Event]

type EventSubscriber = message_queue.Subscriber[EventKey, Event]

type EventConsumer = message_queue.Consumer[EventKey, Event]

type EventConsumerRegistry = message_queue.AttributeRegistry[EventConsumer]

type EventConsumerFeeder = message_queue.Feeder[Event]

type EventConsumerQueue = message_queue.RandomAccessQueue[EventKey, Event]
