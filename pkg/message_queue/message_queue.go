package message_queue

import "context"

type MessageQueue[K comparable, M Message[K]] interface {
	Publish(ctx context.Context, consumerSelectors Matchable, message M) error

	Subscribe(ctx context.Context, consumerSelectors Matchable, consumer Consumer[K, M]) (*RegistrySubscription, error)
	Unsubscribe(subscription *RegistrySubscription)
}

type MessageQueueBase[K comparable, M Message[K]] struct {
	consumers AttributeRegistry[Consumer[K, M]]
}

func NewMessageQueue[K comparable, M Message[K]](maxSelectors int, levelTrie bool) *MessageQueueBase[K, M] {
	m := &MessageQueueBase[K, M]{}
	if levelTrie {
		m.consumers = NewLevelTrie[Consumer[K, M]](maxSelectors)
	} else {
		m.consumers = NewSelectorTrie[Consumer[K, M]](maxSelectors)
	}

	return m
}

func (p *MessageQueueBase[K, M]) Publish(ctx context.Context, consumerSelectors Matchable, message M) error {
	consumers := p.consumers.Find(consumerSelectors)
	for _, consumer := range consumers {
		consumer.Consume(message)
	}
	return nil
}

func (p *MessageQueueBase[K, M]) Subscribe(ctx context.Context, consumerSelectors Matchable, consumer Consumer[K, M]) (*RegistrySubscription, error) {
	return p.consumers.Register(consumerSelectors, consumer)
}

func (p *MessageQueueBase[K, M]) Unsubscribe(subscription *RegistrySubscription) {
	p.consumers.Unregister(subscription)
}
