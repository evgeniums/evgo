package message_queue

import (
	"context"
	"sync/atomic"
	"time"
)

const DEFAULT_MAX_QUEUE_DEPTH int = 0
const DEFAULT_WORK_CHANNEL_DEPTH int = 100

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
	s.closeChannel = make(chan struct{}, 1)
	return s
}

func (s *ConsumerBase[K, M]) SetFeeder(feeder Feeder[M]) {
	s.feeder = feeder
}

func (s *ConsumerBase[K, M]) Feeder() Feeder[M] {
	return s.feeder
}

func (s *ConsumerBase[K, M]) SetQueue(queue RandomAccessQueue[K, M]) {
	s.queue = queue
}

func (s *ConsumerBase[K, M]) Run(ctx context.Context) {

	if s.feeder == nil {
		s.feeder = NewFeeder[M](s, &s.ConsumerConfig.FeederConfig)
	}

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
	}()

	tryNext := func(wrapper messageWrapper[K, M]) {

		// try to push messages from queue
		readyToPush := true

		// read queue
		message, read := s.queue.Front()
		for read && readyToPush && !s.closed.Load() {
			// push dequeued message
			readyToPush = s.feeder.Push(message)
			if readyToPush {
				s.queue.DropFront()
				// read queue again
				message, read = s.queue.Dequeue()
			}
		}

		if readyToPush && wrapper.hasMessage && !s.closed.Load() {

			// try to push new message
			readyToPush = s.feeder.Push(wrapper.message)
			if !readyToPush {
				// cannot push, then enqueue

				// drop oldest message if queue is full
				depth := s.queue.Depth()
				if s.MAX_QUEUE_DEPTH != 0 && (depth+1) > s.MAX_QUEUE_DEPTH {
					s.queue.DropFront()
				}

				// enqueue message
				s.queue.Enqueue(message.Key(), wrapper.message)
			}
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
	if !s.closed.CompareAndSwap(false, true) {
		s.closed.Store(true)

		deadlineCtx, cancel := context.WithTimeout(ctx, time.Duration(s.SHUTDOWN_TIMEOUT)*time.Second)
		defer cancel()

		go func() {
			s.closeChannel <- struct{}{}
			cancel()
		}()

		<-deadlineCtx.Done()
	}
}
