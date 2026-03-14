package message_queue

import "context"

type FeederFanIn[T any] interface {
	FanIn[T]
	AddFeeder(feeder Feeder[T])
	RemoveFeeder(feeder Feeder[T])

	Channel() chan any
	Next()
}

type FeederFanInBase[T any] struct {
	FanInBase[any]

	feeders      map[Feeder[T]]struct{}
	addFeeder    chan Feeder[T]
	removeFeeder chan Feeder[T]

	invokeNext chan struct{}
}

func NewFeederFanIn[T any]() *FeederFanInBase[T] {
	f := &FeederFanInBase[T]{}
	f.construct()
	return f
}

func (f *FeederFanInBase[T]) construct() {
	f.FanInBase.construct()

	f.feeders = make(map[Feeder[T]]struct{})
	f.addFeeder = make(chan Feeder[T])
	f.removeFeeder = make(chan Feeder[T])
	f.invokeNext = make(chan struct{}, 1)
}

func (f *FeederFanInBase[T]) Run(ctx context.Context) {

	go f.FanInBase.Run(ctx)

	go func() {
		for {
			select {

			case <-ctx.Done():
				return

			case <-f.stopAll:
				return

			case feeder := <-f.addFeeder:
				{
					f.feeders[feeder] = struct{}{}
					f.FanInBase.AddInput(feeder.Channel())
				}

			case feeder := <-f.removeFeeder:
				{
					delete(f.feeders, feeder)
					f.FanInBase.RemoveInput(feeder.Channel())
				}

			case <-f.invokeNext:
				{
					for feeder := range f.feeders {
						feeder.Next()
					}
				}
			}
		}
	}()
}

func (f *FeederFanInBase[T]) Channel() chan any {
	return f.out
}

func (f *FeederFanInBase[T]) AddFeeder(feeder Feeder[T]) {
	f.addFeeder <- feeder
}

func (f *FeederFanInBase[T]) RemoveFeeder(feeder Feeder[T]) {
	f.removeFeeder <- feeder
}

func (f *FeederFanInBase[T]) Next() {
	f.invokeNext <- struct{}{}
}
