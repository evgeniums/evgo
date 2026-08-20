package message_queue

import (
	"context"
	"sync"
	"time"
)

type SubscriberFanIn[K comparable, M Message[K]] interface {
	MqChannel

	AddSubscriber(subscriber Subscriber[K, M])
	RemoveSubscriber(subscriber Subscriber[K, M])

	Run(ctx context.Context)
}

type SubscriberFanInBase[K comparable, M Message[K]] struct {
	FanInBase[any]

	subscribersMu sync.Mutex
	subscribers   map[Subscriber[K, M]]struct{}
	closed        bool

	unsubscribeOnce sync.Once
}

func NewSubscriberFanIn[K comparable, M Message[K]]() *SubscriberFanInBase[K, M] {
	f := &SubscriberFanInBase[K, M]{}
	f.construct()
	return f
}

func (f *SubscriberFanInBase[K, M]) construct() {
	f.FanInBase.construct()
	f.subscribers = make(map[Subscriber[K, M]]struct{})
}

func (f *SubscriberFanInBase[K, M]) Run(ctx context.Context) {
	f.FanInBase.Run(ctx)

	// Mirrors the pre-refactor behaviour: external cancellation of ctx (not
	// just an explicit Unsubscribe) still releases every subscription.
	go func() {
		select {
		case <-ctx.Done():
		case <-f.FanInBase.Done():
		}
		f.Unsubscribe(ctx)
	}()
}

func (f *SubscriberFanInBase[K, M]) Channel() <-chan any {
	return f.out
}

// AddSubscriber starts merging subscriber's channel into Channel(). If the
// fan-in has already been unsubscribed/closed, subscriber is immediately
// unsubscribed instead of being silently dropped or blocking the caller.
func (f *SubscriberFanInBase[K, M]) AddSubscriber(subscriber Subscriber[K, M]) {
	f.subscribersMu.Lock()
	if f.closed {
		f.subscribersMu.Unlock()
		unsubscribeCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		subscriber.Unsubscribe(unsubscribeCtx)
		return
	}
	f.subscribers[subscriber] = struct{}{}
	f.subscribersMu.Unlock()

	f.FanInBase.AddInput(subscriber.Channel())
}

// RemoveSubscriber stops merging subscriber's channel and unsubscribes it.
func (f *SubscriberFanInBase[K, M]) RemoveSubscriber(subscriber Subscriber[K, M]) {
	f.subscribersMu.Lock()
	_, existed := f.subscribers[subscriber]
	delete(f.subscribers, subscriber)
	f.subscribersMu.Unlock()

	if !existed {
		return
	}
	f.FanInBase.RemoveInput(subscriber.Channel())
	subscriber.Unsubscribe(context.Background())
}

// Next asks every current subscriber's consumer for its next message. As in
// FeederFanInBase.Next, the subscriber set is snapshotted and released before
// calling out, since Next() can block.
func (f *SubscriberFanInBase[K, M]) Next() {
	f.subscribersMu.Lock()
	snapshot := make([]Subscriber[K, M], 0, len(f.subscribers))
	for subscriber := range f.subscribers {
		snapshot = append(snapshot, subscriber)
	}
	f.subscribersMu.Unlock()

	for _, subscriber := range snapshot {
		subscriber.Next()
	}
}

// Unsubscribe unsubscribes every current subscriber and closes the fan-in.
// Idempotent and safe to call concurrently.
func (f *SubscriberFanInBase[K, M]) Unsubscribe(ctx context.Context) {
	f.unsubscribeOnce.Do(func() {
		f.subscribersMu.Lock()
		f.closed = true
		snapshot := make([]Subscriber[K, M], 0, len(f.subscribers))
		for subscriber := range f.subscribers {
			snapshot = append(snapshot, subscriber)
		}
		clear(f.subscribers)
		f.subscribersMu.Unlock()

		unsubscribeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 1*time.Second)
		defer cancel()
		for _, subscriber := range snapshot {
			subscriber.Unsubscribe(unsubscribeCtx)
		}

		f.FanInBase.Close()
	})
}
