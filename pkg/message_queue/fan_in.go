package message_queue

import (
	"context"
	"sync"
)

// FanIn merges an arbitrary, dynamically changing set of input channels into
// a single output channel.
type FanIn[T any] interface {
	Run(ctx context.Context)
	AddInput(ch <-chan T)
	RemoveInput(ch <-chan T)
	Close()

	Output() chan T
	Done() <-chan struct{}
}

// FanInBase implements FanIn. Its input set (workers) is small and mutated
// rarely relative to how often values flow through it, so it is protected by
// a plain mutex rather than owned by a command-channel run loop: that removes
// an entire class of shutdown-ordering bugs (see message_queue's test suite
// for the regressions this replaced) at the cost of nothing a mutex over a
// small map can't do essentially for free.
type FanInBase[T any] struct {
	out  chan T
	done chan struct{} // closed once out has been closed

	mu      sync.Mutex
	workers map[<-chan T]workerHandle
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
	closed  bool

	wg        sync.WaitGroup
	closeOnce sync.Once
}

// workerHandle lets RemoveInput wait for a specific worker to actually exit
// (not just be told to), so that once RemoveInput returns, nothing more from
// that input can reach Output() - see RemoveInput's doc comment.
type workerHandle struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func NewFanIn[T any]() *FanInBase[T] {
	f := &FanInBase[T]{}
	f.construct()
	return f
}

func (f *FanInBase[T]) construct() {
	f.out = make(chan T, 1)
	f.done = make(chan struct{})
	f.workers = make(map[<-chan T]workerHandle)
}

// Run starts the fan-in. Unlike a command-channel run loop, this does not
// block: it just arms the base with a context and returns. External
// cancellation of ctx triggers the same shutdown as an explicit Close().
func (f *FanInBase[T]) Run(ctx context.Context) {
	f.mu.Lock()
	if f.started || f.closed {
		f.mu.Unlock()
		return
	}
	f.started = true
	f.ctx, f.cancel = context.WithCancel(ctx)
	f.mu.Unlock()

	go func() {
		<-f.ctx.Done()
		f.Close()
	}()
}

// AddInput starts merging ch into Output(). It is a no-op if the fan-in has
// not been started yet or has already been closed - it never blocks.
func (f *FanInBase[T]) AddInput(ch <-chan T) {
	f.mu.Lock()
	if !f.started || f.closed {
		f.mu.Unlock()
		return
	}
	if _, exists := f.workers[ch]; exists {
		f.mu.Unlock()
		return
	}

	wCtx, wCancel := context.WithCancel(f.ctx)
	workerDone := make(chan struct{})
	f.workers[ch] = workerHandle{cancel: wCancel, done: workerDone}
	f.wg.Add(1)
	f.mu.Unlock()

	go f.worker(ch, wCtx, workerDone)
}

func (f *FanInBase[T]) worker(c <-chan T, workerCtx context.Context, done chan struct{}) {
	defer f.wg.Done()
	defer close(done)
	for {
		select {
		case <-workerCtx.Done():
			return
		case v, ok := <-c:
			if !ok {
				return
			}
			select {
			case f.out <- v:
			case <-workerCtx.Done():
				return
			}
		}
	}
}

// RemoveInput stops merging ch and waits for that specific worker to
// actually exit before returning, so that once RemoveInput returns, no value
// read from ch (even one already sitting in the worker's in-flight select)
// can reach Output(). It never blocks indefinitely: a worker only ever waits
// on its own cancellation or a (non-blocking-forever, ctx-guarded) send, so
// this wait is bounded by how fast the runtime schedules that goroutine.
// It is a no-op, returning immediately, if ch was never added (or was
// already removed, or the fan-in has been closed - Close clears the worker
// set under the same lock this looks up).
func (f *FanInBase[T]) RemoveInput(ch <-chan T) {
	f.mu.Lock()
	handle, ok := f.workers[ch]
	if ok {
		delete(f.workers, ch)
	}
	f.mu.Unlock()

	if !ok {
		return
	}
	handle.cancel()
	<-handle.done
}

// Close stops every worker and closes Output(). It is idempotent and
// synchronous: once it returns, Output() is closed and Done() is closed.
func (f *FanInBase[T]) Close() {
	f.closeOnce.Do(func() {
		f.mu.Lock()
		f.closed = true
		cancel := f.cancel
		f.workers = make(map[<-chan T]workerHandle)
		f.mu.Unlock()

		if cancel != nil {
			cancel()
		}
		f.wg.Wait()
		close(f.out)
		close(f.done)
	})
}

func (f *FanInBase[T]) Output() chan T {
	return f.out
}

func (f *FanInBase[T]) Done() <-chan struct{} {
	return f.done
}
