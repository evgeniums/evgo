package message_queue

import "context"

type Producer[K comparable, M Message[K]] interface {
	Publish(ctx context.Context, consumerSelectors Matchable, message M)

	Subscribe(consumerSelectors Matchable, consumer Consumer[K, M]) *RegistrySubscription
	Unsubscribe(subscription *RegistrySubscription)
}

type ProducerBase[K comparable, M Message[K]] struct {
	consumers AttributeRegistry[Consumer[K, M]]
}

func (p *ProducerBase[K, M]) Publish(ctx context.Context, consumerSelectors Matchable, message M) {
	consumers := p.consumers.Find(consumerSelectors)
	for _, consumer := range consumers {
		consumer.Consume(message)
	}
}

func (p *ProducerBase[K, M]) Subscribe(consumerSelectors Matchable, consumer Consumer[K, M]) *RegistrySubscription {
	return p.consumers.Register(consumerSelectors, consumer)
}

func (p *ProducerBase[K, M]) Unsubscribe(subscription *RegistrySubscription) {
	p.consumers.Unregister(subscription)
}
