package message_queue_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/stretchr/testify/require"
)

// testKey is a simple Matchable built from a fixed-length vector of
// Optional[string] selectors, mirroring event_dispatcher.EventKey.
type testKey struct {
	selectors []message_queue.Optional[string]
}

func (k testKey) GetSelectors() []message_queue.Optional[string] {
	return k.selectors
}

// null is used as a readable placeholder for message_queue.None[string]()
// when building a key/selector vector with the key() helper below.
const null = "\x00null\x00"

// key builds a testKey from a list of plain strings, treating the sentinel
// `null` as message_queue.None[string]() and everything else as Some(v).
func key(values ...string) testKey {
	selectors := make([]message_queue.Optional[string], len(values))
	for i, v := range values {
		if v == null {
			selectors[i] = message_queue.None[string]()
		} else {
			selectors[i] = message_queue.Some(v)
		}
	}
	return testKey{selectors: selectors}
}

// testMessage is a plain value-typed message: Key() never panics even on the
// zero value, unlike the production EventWrapper shape (see ptrMessage).
type testMessage struct {
	id      string
	payload string
}

func (m testMessage) Key() string { return m.id }

func msg(id string) testMessage { return testMessage{id: id, payload: id} }

// msgBody/ptrMessage mirror event_dispatcher.EventWrapper's shape: Key() is
// promoted from an embedded pointer, so the zero value's Key() panics with a
// nil pointer dereference. This is what pins down the B4 regression test.
type msgBody struct {
	id string
}

func (b *msgBody) Key() string { return b.id }

type ptrMessage struct {
	*msgBody
}

func ptrMsg(id string) ptrMessage {
	return ptrMessage{msgBody: &msgBody{id: id}}
}

// recordingConsumer is a minimal Consumer[string, testMessage] that just
// records every message it is asked to Consume, without going through
// ConsumerBase's work channel/feeder/backlog machinery. It exists so that
// SelectorTrie/InmemMq routing tests do not depend on ConsumerBase's own
// (separately tested) behaviour.
type recordingConsumer struct {
	mu       sync.Mutex
	received []testMessage
	closed   bool
}

func newRecordingConsumer() *recordingConsumer {
	return &recordingConsumer{}
}

func (c *recordingConsumer) Next() {}

func (c *recordingConsumer) Consume(m testMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.received = append(c.received, m)
}

func (c *recordingConsumer) Feeder() message_queue.Feeder[testMessage] { return nil }

func (c *recordingConsumer) Run(ctx context.Context) {
}

func (c *recordingConsumer) Close(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
}

func (c *recordingConsumer) messages() []testMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]testMessage, len(c.received))
	copy(out, c.received)
	return out
}

func (c *recordingConsumer) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// noopProvider is a MessageProvider whose Next() does nothing; used to build
// standalone Feeder instances in fan-in tests, where nothing above the
// feeder cares about Next() being forwarded anywhere.
type noopProvider struct{}

func (noopProvider) Next() {}

// blockingProvider's Next() blocks until unblocked, letting tests verify
// that a stuck Feeder.Next() call cannot deadlock unrelated Add/Remove calls
// on a FeederFanIn (see the "deadlock guard" tests in fan_in_test.go).
type blockingProvider struct {
	block chan struct{}
}

func (p *blockingProvider) Next() { <-p.block }

// fakeSubscriber is a minimal Subscriber[string, testMessage] for
// SubscriberFanIn tests: it exposes a channel that can be pushed to
// directly, counts Next()/Unsubscribe() calls, and (via blockNext) can
// simulate a subscriber whose Next() blocks.
type fakeSubscriber struct {
	ch chan any

	mu           sync.Mutex
	nextCalls    int
	unsubCalls   int
	unsubscribed bool
	blockNext    chan struct{} // if non-nil, Next() blocks on this until closed
}

func newFakeSubscriber(depth int) *fakeSubscriber {
	return &fakeSubscriber{ch: make(chan any, depth)}
}

func (f *fakeSubscriber) Consumer() message_queue.Consumer[string, testMessage] { return nil }

func (f *fakeSubscriber) Subscribe(ctx context.Context, mq message_queue.MessageQueue[string, testMessage], selectors message_queue.Matchable) error {
	return nil
}

func (f *fakeSubscriber) Channel() <-chan any { return f.ch }

func (f *fakeSubscriber) Next() {
	if f.blockNext != nil {
		<-f.blockNext
		return
	}
	f.mu.Lock()
	f.nextCalls++
	f.mu.Unlock()
}

func (f *fakeSubscriber) Unsubscribe(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.unsubscribed {
		return
	}
	f.unsubscribed = true
	f.unsubCalls++
	close(f.ch)
}

func (f *fakeSubscriber) push(v any) { f.ch <- v }

func (f *fakeSubscriber) wasUnsubscribed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.unsubscribed
}

func (f *fakeSubscriber) unsubscribeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.unsubCalls
}

func (f *fakeSubscriber) nextCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.nextCalls
}

// settleBacklog gives a ConsumerBase's internal process() goroutine time to
// finish handling everything sent to it so far. Consume()/Next() only
// guarantee the message reached the consumer's work channel, not that it has
// already been turned into a feed-vs-backlog(-vs-trim) decision; a test that
// starts draining immediately after a burst of Consume() calls can race
// ahead of that decision (an early receive frees the feeder slot before a
// later Consume() call is processed, changing e.g. which backlog entry ends
// up trimmed or replaced). ConsumerBase exposes no explicit flush/barrier
// hook, so this is a deliberate, generously-bounded sleep rather than a
// spin/poll - the tests that use it assert exact backlog contents, not just
// eventual delivery, and need the writer side to have fully settled first.
func settleBacklog(t *testing.T) {
	t.Helper()
	time.Sleep(100 * time.Millisecond)
}

// drainChannel pulls exactly n values off ch, calling next() before each
// receive to grant credit (mirrors the gRPC stream loop's Next()-then-receive
// pattern), and fails the test if that doesn't happen within timeout.
func drainChannel[V any](t *testing.T, ch <-chan V, next func(), n int, timeout time.Duration) []V {
	t.Helper()
	out := make([]V, 0, n)
	deadline := time.After(timeout)
	for len(out) < n {
		if next != nil {
			next()
		}
		select {
		case v, ok := <-ch:
			require.True(t, ok, "channel closed after receiving only %d/%d values", len(out), n)
			out = append(out, v)
		case <-deadline:
			t.Fatalf("timed out waiting for value %d/%d", len(out)+1, n)
		}
	}
	return out
}
