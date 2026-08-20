package message_queue

import (
	"context"
	"sync"
)

type FeederFanIn[T any] interface {
	FanIn[any]
	AddFeeder(feeder Feeder[T])
	RemoveFeeder(feeder Feeder[T])

	Channel() <-chan any
	Next()
}

type FeederFanInBase[T any] struct {
	FanInBase[any]

	feedersMu sync.Mutex
	feeders   map[Feeder[T]]struct{}
}

func NewFeederFanIn[T any]() *FeederFanInBase[T] {
	f := &FeederFanInBase[T]{}
	f.construct()
	return f
}

func (f *FeederFanInBase[T]) construct() {
	f.FanInBase.construct()
	f.feeders = make(map[Feeder[T]]struct{})
}

func (f *FeederFanInBase[T]) Run(ctx context.Context) {
	f.FanInBase.Run(ctx)
}

func (f *FeederFanInBase[T]) Channel() <-chan any {
	return f.out
}

// AddFeeder starts merging feeder's channel into Channel(). Never blocks.
func (f *FeederFanInBase[T]) AddFeeder(feeder Feeder[T]) {
	f.feedersMu.Lock()
	if _, exists := f.feeders[feeder]; exists {
		f.feedersMu.Unlock()
		return
	}
	f.feeders[feeder] = struct{}{}
	f.feedersMu.Unlock()

	f.FanInBase.AddInput(feeder.Channel())
}

// RemoveFeeder stops merging feeder's channel. Never blocks.
func (f *FeederFanInBase[T]) RemoveFeeder(feeder Feeder[T]) {
	f.feedersMu.Lock()
	delete(f.feeders, feeder)
	f.feedersMu.Unlock()

	f.FanInBase.RemoveInput(feeder.Channel())
}

// Next asks every current feeder for its next message. The feeder set is
// snapshotted under feedersMu and released before calling out - Feeder.Next()
// can block (it forwards to ConsumerBase.tryNext, which may block on a full
// work channel), so it must never run while feedersMu is held or one slow
// feeder would stall every other Add/Remove/Next call.
func (f *FeederFanInBase[T]) Next() {
	f.feedersMu.Lock()
	snapshot := make([]Feeder[T], 0, len(f.feeders))
	for feeder := range f.feeders {
		snapshot = append(snapshot, feeder)
	}
	f.feedersMu.Unlock()

	for _, feeder := range snapshot {
		feeder.Next()
	}
}
