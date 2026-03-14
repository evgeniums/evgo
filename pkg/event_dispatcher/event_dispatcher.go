package event_dispatcher

import (
	"context"

	"github.com/evgeniums/evgo/pkg/generic_error"
)

type Dispatcher interface {
	generic_error.ErrorsExtender

	MakeSubscriber() (EventSubscriber, error)

	Subscribe(ctx context.Context, key EventKey) (EventSubscriber, error)
	Publish(ctx context.Context, event Event) error
}

type SubscriberKey struct{}

func WrapSubscriberContext(ctx context.Context, e EventSubscriber) context.Context {
	newCtx := context.WithValue(ctx, SubscriberKey{}, e)
	return newCtx
}

func MakeSubscriberContext(e EventSubscriber) context.Context {
	ctx := context.WithValue(context.Background(), SubscriberKey{}, e)
	return ctx
}

func SubscriberContext(ctx context.Context) EventSubscriber {
	v, _ := ctx.Value(SubscriberKey{}).(EventSubscriber)
	return v
}
