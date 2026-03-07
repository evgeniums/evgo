package pubsub_subscriber

import (
	"sync"

	"github.com/evgeniums/evgo/pkg/message"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type SubscriberClient[T any] interface {
	Name() string
	Handle(ctx op_context.Context, msg T) error
}

type SubscriberClientBase struct {
	name string
}

func (s *SubscriberClientBase) Init(name string) {
	s.name = name
}

func (s *SubscriberClientBase) Name() string {
	return s.name
}

type Topic interface {
	Name() string
	Handle(ctx op_context.Context, msg []byte, serializer message.Serializer) error
	Unsubscribe(id string)
}

type TopicT[T any] interface {
	Topic
	Subscribe(subscriber SubscriberClient[T])
}

type TopicBase[T any] struct {
	mutex       sync.RWMutex
	name        string
	subscribers map[string]SubscriberClient[T]
	builder     func() T
}

func New[T any](name string, builder func() T) *TopicBase[T] {
	t := &TopicBase[T]{}
	t.name = name
	t.builder = builder
	t.subscribers = make(map[string]SubscriberClient[T])
	return t
}

func (t *TopicBase[T]) Name() string {
	return t.name
}

func (t *TopicBase[T]) Unsubscribe(id string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	delete(t.subscribers, id)
}

func (t *TopicBase[T]) Subscribe(subscriber SubscriberClient[T]) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.subscribers[subscriber.Name()] = subscriber
}

func (t *TopicBase[T]) Handle(ctx op_context.Context, msg []byte, serializer message.Serializer) error {

	c := ctx.TraceInMethod("pubsub.Topic.Handle")
	defer ctx.TraceOutMethod()

	obj := t.builder()
	err := serializer.ParseMessage(msg, obj)
	if err != nil {
		c.SetMessage("failed to unmarshal message")
		return c.SetError(err)
	}

	t.mutex.RLock()
	subscribers := utils.AllMapValues(t.subscribers)
	t.mutex.RUnlock()

	for _, subscriber := range subscribers {
		err = subscriber.Handle(ctx, obj)
		if err != nil {
			c.SetLoggerField("subscriber", subscriber.Name())
			c.SetMessage("failed to handle message")
			ctx.DumpLog()
		}
	}

	return nil
}
