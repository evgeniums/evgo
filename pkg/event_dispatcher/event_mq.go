package event_dispatcher

import "github.com/evgeniums/evgo/pkg/message_queue"

type EventMq = message_queue.MessageQueue[EventKey, Event]

type EventSubscriber = message_queue.Subscriber[EventKey, Event]
