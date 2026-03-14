package message_queue

import (
	"context"
	"sync"
)

type FanIn[T any] interface {
	Run(ctx context.Context)
	AddInput(ch <-chan T)
	RemoveInput(ch <-chan T)
	Close()

	Output() chan T
}

type FanInBase[T any] struct {
	out         chan T
	addInput    chan (<-chan T)
	removeInput chan (<-chan T)

	workers map[<-chan T]context.CancelFunc

	stopAll chan struct{}

	wg sync.WaitGroup
}

func NewFanIn[T any]() *FanInBase[T] {
	f := &FanInBase[T]{}
	f.construct()
	return f
}

func (f *FanInBase[T]) construct() {
	f.out = make(chan T, 1)
	f.addInput = make(chan (<-chan T))
	f.removeInput = make(chan (<-chan T))
	f.stopAll = make(chan struct{}, 1)
	f.workers = make(map[<-chan T]context.CancelFunc)
}

func (f *FanInBase[T]) Run(ctx context.Context) {

	// unified Shutdown Logic
	go func() {
		select {
		case <-ctx.Done(): // External cancellation (timeout/cancel)
		case <-f.stopAll: // Internal manual stop
		}

		// cancel all individual workers via their child contexts
		for _, cancel := range f.workers {
			cancel()
		}

		f.wg.Wait()
		close(f.out)
	}()

	for {
		select {

		case <-ctx.Done():
			return

		case <-f.stopAll:
			return

		case newInput := <-f.addInput:
			{
				// add input channel
				wCtx, wCancel := context.WithCancel(ctx)
				f.workers[newInput] = wCancel

				// run worker for this specific channel
				go func(c <-chan T, workerCtx context.Context) {
					defer f.wg.Done()
					for {
						select {
						case <-workerCtx.Done():
						case v, ok := <-c:
							if !ok {
								return
							}
							f.out <- v
						}
					}
				}(newInput, wCtx)
			}

		case oldCh := <-f.removeInput:
			if cancel, ok := f.workers[oldCh]; ok {
				cancel()
				delete(f.workers, oldCh)
			}
		}
	}
}

func (f *FanInBase[T]) AddInput(ch <-chan T) {
	f.addInput <- ch
}

func (f *FanInBase[T]) RemoveInput(ch <-chan T) {
	f.removeInput <- ch
}

func (f *FanInBase[T]) Close() {
	f.stopAll <- struct{}{}
}
