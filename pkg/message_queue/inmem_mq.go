package message_queue

import "context"

type InmemMq[K comparable, M Message[K]] struct {
	consumers AttributeRegistry[Consumer[K, M]]
}

func NewInmemMq[K comparable, M Message[K]](maxSelectors int, consumers ...AttributeRegistry[Consumer[K, M]]) *InmemMq[K, M] {
	m := &InmemMq[K, M]{}
	if len(consumers) == 0 || consumers[0] == nil {
		m.consumers = NewSelectorTrie[Consumer[K, M]](maxSelectors)
	}
	return m
}

func (p *InmemMq[K, M]) Publish(ctx context.Context, consumerSelectors Matchable, message M) error {
	consumers := p.consumers.Find(consumerSelectors)
	for _, consumer := range consumers {
		consumer.Consume(message)
	}
	return nil
}

func (p *InmemMq[K, M]) Subscribe(ctx context.Context, consumerSelectors Matchable, consumer Consumer[K, M]) (*RegistrySubscription, error) {
	consumer.Run(ctx)
	return p.consumers.Register(consumerSelectors, consumer)
}

func (p *InmemMq[K, M]) Unsubscribe(ctx context.Context, subscription *RegistrySubscription) {
	existing := p.consumers.Unregister(subscription)
	if existing != nil {
		existing.Close(ctx)
	}
}
