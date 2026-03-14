package message_queue

import "context"

type SubscriberExt[K comparable, M Message[K]] interface {
	Consumer() Consumer[K, M]
	Subscribe(producer Producer[K, M], selectors Matchable)
	Unsubscribe(producer Producer[K, M])
}

type Subscriber[K comparable, M Message[K]] interface {
	Consumer() Consumer[K, M]
	Subscribe(producer Producer[K, M], selectors Matchable)
	Unsubscribe()
}

type SubscriberExtBase[K comparable, M Message[K]] struct {
	consumer     Consumer[K, M]
	subscription *RegistrySubscription
}

func NewSubscriberExt[K comparable, M Message[K]](consumer ...Consumer[K, M]) *SubscriberExtBase[K, M] {
	s := &SubscriberExtBase[K, M]{}
	if len(consumer) == 0 {
		s.consumer = NewConsumer[K, M]()
	} else {
		s.consumer = consumer[0]
	}
	return s
}

func (s *SubscriberExtBase[K, M]) Subscribe(producer Producer[K, M], selectors Matchable) {
	s.subscription = producer.Subscribe(selectors, s.consumer)
}

func (s *SubscriberExtBase[K, M]) Unsubscribe(producer Producer[K, M]) {
	producer.Unsubscribe(s.subscription)
}

func (s *SubscriberExtBase[K, M]) Consumer() Consumer[K, M] {
	return s.consumer
}

func (s *SubscriberExtBase[K, M]) Channel() <-chan any {
	return s.consumer.Feeder().Channel()
}

func (s *SubscriberExtBase[K, M]) Next() {
	s.consumer.Feeder().Next()
}

type SubscriberBase[K comparable, M Message[K]] struct {
	SubscriberExtBase[K, M]
	producer Producer[K, M]
}

func NewSubscriber[K comparable, M Message[K]](consumer ...Consumer[K, M]) *SubscriberBase[K, M] {
	s := &SubscriberBase[K, M]{}
	if len(consumer) == 0 {
		s.consumer = NewConsumer[K, M]()
	} else {
		s.consumer = consumer[0]
	}
	return s
}

func (s *SubscriberBase[K, M]) Subscribe(producer Producer[K, M], selectors Matchable) {
	s.SubscriberExtBase.Subscribe(producer, selectors)
	s.producer = producer
}

func (s *SubscriberBase[K, M]) Unsubscribe() {
	if s.producer != nil {
		s.SubscriberExtBase.Unsubscribe(s.producer)
	}
}

type MqChannel interface {
	Channel() <-chan any
	Unsubscribe()
	Next()
}

type MqKey struct{}

func WrapMqContext(ctx context.Context, c MqChannel) context.Context {
	newCtx := context.WithValue(ctx, MqKey{}, c)
	return newCtx
}

func MakeMqContext(c MqChannel) context.Context {
	ctx := context.WithValue(context.Background(), MqKey{}, c)
	return ctx
}

func MqContext(ctx context.Context) MqChannel {
	v, _ := ctx.Value(MqKey{}).(MqChannel)
	return v
}
