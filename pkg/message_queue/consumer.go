package message_queue

import (
	"context"
	"sync/atomic"
	"time"
)

const DEFAULT_MAX_QUEUE_DEPTH int = 0
const DEFAULT_WORK_CHANNEL_DEPTH int = 100
const DEFAULT_SHUTDOWN_TIMEOUT int = 1

type Message[K comparable] interface {
	Key() K
}

type Consumer[K comparable, M Message[K]] interface {
	MessageProvider
	Consume(message M)
	Feeder() Feeder[M]

	Run(ctx context.Context)
	Close(ctx context.Context)
}

type messageWrapper[K comparable, M Message[K]] struct {
	message    M
	hasMessage bool
}

type ConsumerConfig struct {
	FeederConfig
	MAX_QUEUE_DEPTH    int
	WORK_CHANNEL_DEPTH int `default:"100"`
	SHUTDOWN_TIMEOUT   int `default:"1"`
}

func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		MAX_QUEUE_DEPTH:    DEFAULT_MAX_QUEUE_DEPTH,
		WORK_CHANNEL_DEPTH: DEFAULT_WORK_CHANNEL_DEPTH,
		SHUTDOWN_TIMEOUT:   DEFAULT_SHUTDOWN_TIMEOUT,
		FeederConfig: FeederConfig{
			FEEDER_CHANNEL_DEPTH: DEFAULT_FEEDER_CHANNEL_DEPTH,
		},
	}
}

type ConsumerBase[K comparable, M Message[K]] struct {
	ConsumerConfig
	queue  RandomAccessQueue[K, M]
	feeder Feeder[M]

	workChannel  chan messageWrapper[K, M]
	closeChannel chan struct{}
	doneChannel  chan struct{}

	closed atomic.Bool
	ctx    context.Context
}

func NewConsumer[K comparable, M Message[K]](config ...ConsumerConfig) *ConsumerBase[K, M] {
	s := &ConsumerBase[K, M]{}
	if len(config) == 0 {
		s.ConsumerConfig = DefaultConsumerConfig()
	} else {
		s.ConsumerConfig = config[0]
	}

	s.workChannel = make(chan messageWrapper[K, M], s.WORK_CHANNEL_DEPTH)
	s.closeChannel = make(chan struct{}, 0)
	s.doneChannel = make(chan struct{}, 0)
	// A live, already-cancelled-free context so that Consume/Next/Close called
	// before Run (or Feeder() called before Run) do not panic on a nil s.ctx;
	// Run replaces it with the real context once the consumer actually starts.
	s.ctx = context.Background()
	return s
}

func (s *ConsumerBase[K, M]) SetFeeder(feeder Feeder[M]) {
	s.feeder = feeder
}

// Feeder returns the consumer's feeder, lazily creating the default one if
// neither SetFeeder nor Run has run yet, so it is always safe to call.
func (s *ConsumerBase[K, M]) Feeder() Feeder[M] {
	if s.feeder == nil {
		s.feeder = NewFeeder[M](s, &s.ConsumerConfig.FeederConfig)
	}
	return s.feeder
}

func (s *ConsumerBase[K, M]) SetQueue(queue RandomAccessQueue[K, M]) {
	s.queue = queue
}

func (s *ConsumerBase[K, M]) Run(ctx context.Context) {

	// Reuses the same lazy-creation path as Feeder() so that a feeder handed
	// out before Run (e.g. via Subscriber.Channel()) is the same one Run uses.
	_ = s.Feeder()

	if s.queue == nil {
		s.queue = NewReplacingQueue[K, M]()
	}

	s.ctx = ctx

	go s.process()
}

func (s *ConsumerBase[K, M]) Consume(message M) {
	s.tryNext(message)
}

func (s *ConsumerBase[K, M]) Next() {
	s.tryNext()
}

func (s *ConsumerBase[K, M]) process() {

	defer func() {
		s.queue.Clear()
		s.feeder.Close()
		close(s.doneChannel)
	}()

	tryNext := func(wrapper messageWrapper[K, M]) {

		// Drain as much of the backlog into the feeder as it will accept.
		// feederReady tracks whether the feeder has room right now: false
		// means the last Push failed (feeder full), in which case the
		// message that failed to push is left at Front() - Front() only
		// peeks, DropFront() removes an item once it is actually pushed, so
		// a backlog message is never skipped or double-consumed.
		feederReady := true
		for !s.closed.Load() {
			message, read := s.queue.Front()
			if !read {
				break
			}
			feederReady = s.feeder.Push(message)
			if !feederReady {
				break
			}
			s.queue.DropFront()
		}

		if !wrapper.hasMessage || s.closed.Load() {
			return
		}

		if feederReady {
			// Backlog is fully drained and the feeder just accepted
			// everything - try the new message directly.
			feederReady = s.feeder.Push(wrapper.message)
		}
		if !feederReady {
			// Feeder had no room (either for a backlog message above, or for
			// the new one just now) - append the new message to the backlog
			// under its own key instead of dropping it. Using the stale
			// drain-loop "message" var here (rather than wrapper.message)
			// was a bug: it is the zero value whenever the backlog was
			// empty, filing the entry under the wrong key at best and
			// panicking on Key() for pointer-backed message types at worst.
			if s.MAX_QUEUE_DEPTH != 0 && s.queue.Depth() >= s.MAX_QUEUE_DEPTH {
				s.queue.DropFront()
			}
			s.queue.Enqueue(wrapper.message.Key(), wrapper.message)
		}
	}

	for {
		select {

		// SIGNAL 1: context done
		case <-s.ctx.Done():
			return

		// SIGNAL 2: consumer closed
		case <-s.closeChannel:
			return

		// SIGNAL 4: try next
		case wrapper, ok := <-s.workChannel:
			if !ok {
				return
			}
			tryNext(wrapper)

		}
	}
}

func (s *ConsumerBase[K, M]) tryNext(msg ...M) {
	if len(msg) > 0 {
		select {
		case s.workChannel <- messageWrapper[K, M]{msg[0], true}:
			return
		case <-s.ctx.Done():
			return
		case <-s.closeChannel:
			return
		}
	} else {
		select {
		case s.workChannel <- messageWrapper[K, M]{}:
			return
		case <-s.ctx.Done():
			return
		case <-s.closeChannel:
			return
		}
	}
}

// Handler for interface of pubsub subscriber client
func (s *ConsumerBase[K, M]) Handle(ctx context.Context, message M) error {
	s.Consume(message)
	return nil
}

func (s *ConsumerBase[K, M]) Close(ctx context.Context) {

	// check if called once, if it was already true, the second caller skips this and exits immediately
	if !s.closed.CompareAndSwap(false, true) {
		return
	}

	// this unblocks ANY select statement listening to <-s.closeChannel instantly
	close(s.closeChannel)

	// wait for the consumer goroutine to finish processing
	deadlineCtx, cancel := context.WithTimeout(ctx, time.Duration(s.SHUTDOWN_TIMEOUT)*time.Second)
	defer cancel()

	select {
	case <-s.doneChannel:
		// consumer exited cleanly within the timeout window
	case <-deadlineCtx.Done():
		// timeout reached before consumer could finish draining its queue
	}
}
