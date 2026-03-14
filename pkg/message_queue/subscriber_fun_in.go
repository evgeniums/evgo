package message_queue

import "context"

type SubscriberFanIn[K comparable, M Message[K]] interface {
	MqChannel

	AddSubscriber(subscriber Subscriber[K, M])
	RemoveSubscriber(subscriber Subscriber[K, M])
}

type SubscriberFanInBase[K comparable, M Message[K]] struct {
	FanInBase[any]

	subscribers      map[Subscriber[K, M]]struct{}
	addSubscriber    chan Subscriber[K, M]
	removeSubscriber chan Subscriber[K, M]

	invokeNext  chan struct{}
	unsubscribe chan context.Context
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
	f.unsubscribe = make(chan context.Context)
}

func (f *SubscriberFanInBase[K, M]) Run(ctx context.Context) {

	go f.FanInBase.Run(ctx)

	go func() {

		unsubscribe := func(ctx context.Context) {
			for subscriber := range f.subscribers {
				subscriber.Unsubscribe(ctx)
			}
		}

		for {
			select {

			case <-ctx.Done():
				unsubscribe(ctx)
				return

			case <-f.stopAll:
				unsubscribe(ctx)
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

			case opCtx := <-f.unsubscribe:
				{
					unsubscribe(opCtx)
					f.Close()
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
	f.unsubscribe <- ctx
}
