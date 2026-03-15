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

	subscribers      map[Subscriber[K, M]]struct{}
	addSubscriber    chan Subscriber[K, M]
	removeSubscriber chan Subscriber[K, M]

	invokeNext  chan struct{}
	unsubscribe chan struct{}

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
	f.addSubscriber = make(chan Subscriber[K, M])
	f.removeSubscriber = make(chan Subscriber[K, M])
	f.invokeNext = make(chan struct{}, 1)
	f.unsubscribe = make(chan struct{})
}

func (f *SubscriberFanInBase[K, M]) Run(ctx context.Context) {

	go f.FanInBase.Run(ctx)

	go func() {

		unsubscribe := func() {
			unsubscribeCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			for subscriber := range f.subscribers {
				subscriber.Unsubscribe(unsubscribeCtx)
			}
			clear(f.subscribers)
		}

		for {
			select {

			case <-ctx.Done():
				unsubscribe()
				return

			case <-f.stopAll:
				unsubscribe()
				return

			case subscriber := <-f.addSubscriber:
				{
					f.subscribers[subscriber] = struct{}{}
					f.FanInBase.AddInput(subscriber.Channel())
				}

			case subscriber := <-f.removeSubscriber:
				{
					delete(f.subscribers, subscriber)
					f.FanInBase.RemoveInput(subscriber.Channel())
					subscriber.Unsubscribe(ctx)
				}

			case <-f.invokeNext:
				{
					for subscriber := range f.subscribers {
						subscriber.Next()
					}
				}

			case <-f.unsubscribe:
				{
					unsubscribe()
					f.Close()
					return
				}
			}
		}
	}()
}

func (f *SubscriberFanInBase[K, M]) Channel() <-chan any {
	return f.out
}

func (f *SubscriberFanInBase[K, M]) AddSubscriber(subscriber Subscriber[K, M]) {
	f.addSubscriber <- subscriber
}

func (f *SubscriberFanInBase[K, M]) RemoveSubscriber(subscriber Subscriber[K, M]) {
	f.removeSubscriber <- subscriber
}

func (f *SubscriberFanInBase[K, M]) Next() {
	f.invokeNext <- struct{}{}
}

func (f *SubscriberFanInBase[K, M]) Unsubscribe(ctx context.Context) {
	f.unsubscribeOnce.Do(func() {
		close(f.unsubscribe)
	})
}
