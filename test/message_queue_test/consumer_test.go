package message_queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConsumerConfig(feederDepth, maxQueueDepth int) message_queue.ConsumerConfig {
	cfg := message_queue.DefaultConsumerConfig()
	cfg.FeederConfig.FEEDER_CHANNEL_DEPTH = feederDepth
	cfg.MAX_QUEUE_DEPTH = maxQueueDepth
	cfg.WORK_CHANNEL_DEPTH = 64
	return cfg
}

func runConsumer[K comparable, M message_queue.Message[K]](t *testing.T, cfg message_queue.ConsumerConfig) (*message_queue.ConsumerBase[K, M], context.CancelFunc) {
	t.Helper()
	consumer := message_queue.NewConsumer[K, M](cfg)
	ctx, cancel := context.WithCancel(context.Background())
	consumer.Run(ctx)
	t.Cleanup(cancel)
	return consumer, cancel
}

// TestConsumerBacklogDeliversEverythingInOrder is the B3 regression: the
// original drain loop used Dequeue() (remove-and-return) after DropFront()
// (remove) on the same iteration, silently discarding every other
// backlogged message once the feeder filled up.
func TestConsumerBacklogDeliversEverythingInOrder(t *testing.T) {
	const feederDepth = 2
	const n = 6
	cfg := newTestConsumerConfig(feederDepth, 0)
	consumer, _ := runConsumer[string, testMessage](t, cfg)

	for i := 0; i < n; i++ {
		consumer.Consume(msg(string(rune('0' + i))))
	}

	feeder := consumer.Feeder()
	got := drainChannel(t, feeder.Channel(), consumer.Next, n, 3*time.Second)

	ids := make([]string, len(got))
	for i, v := range got {
		ids[i] = v.(testMessage).id
	}
	assert.Equal(t, []string{"0", "1", "2", "3", "4", "5"}, ids,
		"all backlogged messages must arrive, in FIFO order - none silently dropped")
}

// TestConsumerBacklogEmptyPointerKeyDoesNotPanic is the B4 regression: the
// enqueue path used the stale drain-loop "message" variable instead of
// wrapper.message to compute the backlog key. When the backlog is empty that
// stale variable is the zero value of M; for a pointer-embedding message
// type (mirroring the production EventWrapper shape) that zero value panics
// on Key(). Two messages with a feeder depth of 1 hits exactly the
// empty-backlog first-enqueue path.
func TestConsumerBacklogEmptyPointerKeyDoesNotPanic(t *testing.T) {
	cfg := newTestConsumerConfig(1, 0)
	consumer, _ := runConsumer[string, ptrMessage](t, cfg)

	require.NotPanics(t, func() {
		consumer.Consume(ptrMsg("0"))
		consumer.Consume(ptrMsg("1")) // feeder is full here: backlog was empty until this call
	})

	feeder := consumer.Feeder()
	got := drainChannel(t, feeder.Channel(), consumer.Next, 2, 3*time.Second)
	ids := []string{got[0].(ptrMessage).id, got[1].(ptrMessage).id}
	assert.ElementsMatch(t, []string{"0", "1"}, ids)
}

// TestConsumerBacklogDistinctKeysAllSurvive checks that backlogged messages
// enqueued while the backlog was already non-empty are filed under their own
// key (not the previous item's), so distinct messages don't collide/replace
// each other in the ReplacingQueue.
func TestConsumerBacklogDistinctKeysAllSurvive(t *testing.T) {
	cfg := newTestConsumerConfig(1, 0)
	consumer, _ := runConsumer[string, testMessage](t, cfg)

	consumer.Consume(msg("a")) // fills the feeder (depth 1)
	consumer.Consume(msg("b")) // backlog: [b]
	consumer.Consume(msg("c")) // backlog: [b, c] - must not collide with "b"

	feeder := consumer.Feeder()
	got := drainChannel(t, feeder.Channel(), consumer.Next, 3, 3*time.Second)
	ids := make([]string, len(got))
	for i, v := range got {
		ids[i] = v.(testMessage).id
	}
	assert.Equal(t, []string{"a", "b", "c"}, ids)
}

func TestConsumerMaxQueueDepthTrimsOldest(t *testing.T) {
	cfg := newTestConsumerConfig(1, 2) // feeder holds 1, backlog capped at 2
	consumer, _ := runConsumer[string, testMessage](t, cfg)

	consumer.Consume(msg("a")) // into feeder
	consumer.Consume(msg("b")) // backlog: [b]
	consumer.Consume(msg("c")) // backlog: [b, c]
	consumer.Consume(msg("d")) // backlog full at 2 -> drop "b", backlog: [c, d]

	// Consume() only guarantees the message reaches the consumer's internal
	// work channel, not that process() has already turned it into a
	// feed-vs-backlog decision. Without this settle point, draining below
	// can race ahead of process() and free the feeder slot before "b"/"c"/"d"
	// are processed - e.g. draining "a" before "b" is processed lets "b" go
	// straight to the feeder instead of the backlog, changing which message
	// ends up trimmed. See settleBacklog's doc comment.
	settleBacklog(t)

	feeder := consumer.Feeder()
	got := drainChannel(t, feeder.Channel(), consumer.Next, 3, 3*time.Second)
	ids := make([]string, len(got))
	for i, v := range got {
		ids[i] = v.(testMessage).id
	}
	assert.Equal(t, []string{"a", "c", "d"}, ids, "\"b\" must have been dropped as the oldest backlog entry")
}

func TestConsumerSameKeyReplacesInBacklog(t *testing.T) {
	cfg := newTestConsumerConfig(1, 0)
	consumer, _ := runConsumer[string, testMessage](t, cfg)

	consumer.Consume(msg("a"))                                // into feeder
	consumer.Consume(testMessage{id: "k", payload: "first"})  // backlog: [k=first]
	consumer.Consume(testMessage{id: "k", payload: "second"}) // replaces in place, still 1 entry
	settleBacklog(t)                                          // see TestConsumerMaxQueueDepthTrimsOldest

	feeder := consumer.Feeder()
	got := drainChannel(t, feeder.Channel(), consumer.Next, 2, 3*time.Second)
	require.Len(t, got, 2)
	assert.Equal(t, "a", got[0].(testMessage).id)
	assert.Equal(t, "second", got[1].(testMessage).payload, "later Consume for the same key must replace, not queue twice")
}

func TestConsumerCloseIsIdempotentAndWaits(t *testing.T) {
	cfg := newTestConsumerConfig(4, 0)
	consumer := message_queue.NewConsumer[string, testMessage](cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	consumer.Run(ctx)

	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer closeCancel()

	assert.NotPanics(t, func() { consumer.Close(closeCtx) })
	assert.NotPanics(t, func() { consumer.Close(closeCtx) }, "second Close must be a no-op, not panic or hang")

	_, ok := <-consumer.Feeder().Channel()
	assert.False(t, ok, "Close must close the feeder channel")
}

// TestConsumerConsumeBeforeRunDoesNotPanic is the B10 regression: tryNext
// selects on s.ctx.Done(), and s.ctx used to be assigned only in Run, so
// calling Consume/Next before Run panicked on a nil context.
func TestConsumerConsumeBeforeRunDoesNotPanic(t *testing.T) {
	cfg := newTestConsumerConfig(4, 0)
	consumer := message_queue.NewConsumer[string, testMessage](cfg)

	assert.NotPanics(t, func() {
		consumer.Consume(msg("a"))
	})
}

// TestConsumerFeederAvailableBeforeRun is the B11 regression:
// SubscriberExtBase.Channel() calls consumer.Feeder().Channel() and used to
// panic on a nil Feeder() when called before Run/Subscribe.
func TestConsumerFeederAvailableBeforeRun(t *testing.T) {
	cfg := newTestConsumerConfig(4, 0)
	consumer := message_queue.NewConsumer[string, testMessage](cfg)

	feeder := consumer.Feeder()
	require.NotNil(t, feeder)
	assert.NotNil(t, feeder.Channel())
}
