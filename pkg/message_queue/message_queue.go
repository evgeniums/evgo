package message_queue

import "context"

type MessageQueue[K comparable, M Message[K]] interface {
	Publish(ctx context.Context, consumerSelectors Matchable, message M) error

	Subscribe(ctx context.Context, consumerSelectors Matchable, consumer Consumer[K, M]) (*RegistrySubscription, error)
	Unsubscribe(ctx context.Context, subscription *RegistrySubscription)
}
