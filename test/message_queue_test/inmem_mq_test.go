package message_queue_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMq(maxSelectors int) *message_queue.InmemMq[string, testMessage] {
	return message_queue.NewInmemMq[string, testMessage](maxSelectors)
}

// subscribe wires up a real ConsumerBase (via NewSubscriber) against mq, so
// these tests exercise the full Publish -> SelectorTrie -> workChannel ->
// feeder pipeline, not just the registry in isolation.
func subscribe(t *testing.T, ctx context.Context, mq *message_queue.InmemMq[string, testMessage], selectors message_queue.Matchable) message_queue.Subscriber[string, testMessage] {
	t.Helper()
	cfg := message_queue.DefaultConsumerConfig()
	cfg.FeederConfig.FEEDER_CHANNEL_DEPTH = 8
	consumer := message_queue.NewConsumer[string, testMessage](cfg)
	sub := message_queue.NewSubscriber(consumer)
	require.NoError(t, sub.Subscribe(ctx, mq, selectors))
	return sub
}

func TestInmemMqPublishSubscribeDelivers(t *testing.T) {
	mq := newMq(3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := subscribe(t, ctx, mq, key("chat", "1", null))
	defer sub.Unsubscribe(ctx)

	require.NoError(t, mq.Publish(ctx, key("chat", "1", "created"), msg("hello")))

	got := drainChannel(t, sub.Channel(), sub.Next, 1, 2*time.Second)
	assert.Equal(t, "hello", got[0].(testMessage).id)
}

func TestInmemMqNonMatchingSelectorsDeliverNothing(t *testing.T) {
	mq := newMq(3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := subscribe(t, ctx, mq, key("chat", "1", null))
	defer sub.Unsubscribe(ctx)

	require.NoError(t, mq.Publish(ctx, key("chat", "2", "created"), msg("other-chat")))

	select {
	case v := <-sub.Channel():
		t.Fatalf("unexpected delivery for non-matching selectors: %v", v)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestInmemMqFanOutToMultipleSubscribers(t *testing.T) {
	mq := newMq(3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subA := subscribe(t, ctx, mq, key("chat", "1", null))
	defer subA.Unsubscribe(ctx)
	subB := subscribe(t, ctx, mq, key("chat", null, null))
	defer subB.Unsubscribe(ctx)

	require.NoError(t, mq.Publish(ctx, key("chat", "1", "created"), msg("hello")))

	gotA := drainChannel(t, subA.Channel(), subA.Next, 1, 2*time.Second)
	gotB := drainChannel(t, subB.Channel(), subB.Next, 1, 2*time.Second)
	assert.Equal(t, "hello", gotA[0].(testMessage).id)
	assert.Equal(t, "hello", gotB[0].(testMessage).id)
}

func TestInmemMqUnsubscribeStopsDelivery(t *testing.T) {
	mq := newMq(3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := subscribe(t, ctx, mq, key("chat", "1", null))

	sub.Unsubscribe(ctx)

	require.NoError(t, mq.Publish(ctx, key("chat", "1", "created"), msg("hello")))

	select {
	case v, ok := <-sub.Channel():
		assert.False(t, ok, "channel must be closed, not deliver a message, after Unsubscribe: got %v", v)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Unsubscribe must close the consumer's feeder channel promptly")
	}
}

func TestInmemMqUnsubscribeIsSafeToCallTwiceAndWithNil(t *testing.T) {
	mq := newMq(3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := subscribe(t, ctx, mq, key("chat", "1", null))
	assert.NotPanics(t, func() { sub.Unsubscribe(ctx) })
	assert.NotPanics(t, func() { sub.Unsubscribe(ctx) }, "second Unsubscribe must be a no-op, not panic")

	assert.NotPanics(t, func() { mq.Unsubscribe(ctx, nil) })
}

// TestInmemMqUnsubscribeDoesNotAffectSiblingSubscriber is the B1 end-to-end
// regression, the highest-value test in this suite: unsubscribing one
// consumer must not silently detach an unrelated, still-live sibling that
// happens to share a selector prefix. Before the SelectorTrie.Unregister fix,
// this scenario left subB permanently unable to receive further messages.
func TestInmemMqUnsubscribeDoesNotAffectSiblingSubscriber(t *testing.T) {
	mq := newMq(3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subA := subscribe(t, ctx, mq, key("chat", "1", "typing"))
	subB := subscribe(t, ctx, mq, key("chat", "1", "created"))
	defer subB.Unsubscribe(ctx)

	subA.Unsubscribe(ctx)

	require.NoError(t, mq.Publish(ctx, key("chat", "1", "created"), msg("still-alive")))

	got := drainChannel(t, subB.Channel(), subB.Next, 1, 2*time.Second)
	assert.Equal(t, "still-alive", got[0].(testMessage).id,
		"unsubscribing a sibling must not detach this still-live subscription")
}

// TestInmemMqNewInmemMqAcceptsExplicitRegistry is the B5 regression:
// NewInmemMq used to silently leave m.consumers nil whenever a non-nil
// registry was passed explicitly, panicking on first use.
func TestInmemMqNewInmemMqAcceptsExplicitRegistry(t *testing.T) {
	registry := message_queue.NewLevelTrie[message_queue.Consumer[string, testMessage]](3)
	mq := message_queue.NewInmemMq[string, testMessage](3, registry)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Registered under a sibling-shadowed prefix (see
	// TestLevelTrieFindsWhatSelectorTrieMisses): only a LevelTrie-backed mq
	// finds this via its branching None search.
	sub := subscribe(t, ctx, mq, key("chat", null, "created"))
	defer sub.Unsubscribe(ctx)
	sibling := subscribe(t, ctx, mq, key("chat", "1", "typing")) // unrelated, under the same prefix
	defer sibling.Unsubscribe(ctx)

	require.NoError(t, mq.Publish(ctx, key("chat", "1", "created"), msg("hello")))

	got := drainChannel(t, sub.Channel(), sub.Next, 1, 2*time.Second)
	assert.Equal(t, "hello", got[0].(testMessage).id)
}

func TestInmemMqNewInmemMqWithNilRegistryVariadicFallsBackToDefault(t *testing.T) {
	var nilRegistry message_queue.AttributeRegistry[message_queue.Consumer[string, testMessage]]
	mq := message_queue.NewInmemMq[string, testMessage](3, nilRegistry)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub := subscribe(t, ctx, mq, key("chat", "1", null))
	defer sub.Unsubscribe(ctx)

	require.NoError(t, mq.Publish(ctx, key("chat", "1", "created"), msg("hello")))
	got := drainChannel(t, sub.Channel(), sub.Next, 1, 2*time.Second)
	assert.Equal(t, "hello", got[0].(testMessage).id)
}

// TestInmemMqConcurrentPublishAndSubscribeNoLoss drives several publisher
// goroutines against several subscribers concurrently and checks that every
// expected message is received exactly once per matching subscriber, and
// that everything shuts down cleanly. Intended to run under -race.
func TestInmemMqConcurrentPublishAndSubscribeNoLoss(t *testing.T) {
	const chats = 4
	const messagesPerChat = 25

	mq := newMq(3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subs := make([]message_queue.Subscriber[string, testMessage], chats)
	received := make([][]string, chats)
	var mus [chats]sync.Mutex
	var wgReceivers sync.WaitGroup

	for i := 0; i < chats; i++ {
		chatID := fmt.Sprintf("%d", i)
		sub := subscribe(t, ctx, mq, key("chat", chatID, null))
		subs[i] = sub

		wgReceivers.Add(1)
		go func(i int, sub message_queue.Subscriber[string, testMessage]) {
			defer wgReceivers.Done()
			for {
				sub.Next()
				select {
				case v, ok := <-sub.Channel():
					if !ok {
						return
					}
					mus[i].Lock()
					received[i] = append(received[i], v.(testMessage).id)
					mus[i].Unlock()
				case <-time.After(3 * time.Second):
					return
				}
			}
		}(i, sub)
	}

	var wgPublishers sync.WaitGroup
	for i := 0; i < chats; i++ {
		chatID := fmt.Sprintf("%d", i)
		wgPublishers.Add(1)
		go func(chatID string) {
			defer wgPublishers.Done()
			for j := 0; j < messagesPerChat; j++ {
				id := fmt.Sprintf("%s-%d", chatID, j)
				require.NoError(t, mq.Publish(ctx, key("chat", chatID, "created"), msg(id)))
			}
		}(chatID)
	}
	wgPublishers.Wait()

	// Give the receiver goroutines a moment to drain everything, then close
	// each subscription so the receiver loops observe the closed channel and
	// exit instead of idling out on their own 3s timeout.
	require.Eventually(t, func() bool {
		for i := 0; i < chats; i++ {
			mus[i].Lock()
			n := len(received[i])
			mus[i].Unlock()
			if n < messagesPerChat {
				return false
			}
		}
		return true
	}, 5*time.Second, 20*time.Millisecond, "every published message must eventually be delivered")

	for i := 0; i < chats; i++ {
		subs[i].Unsubscribe(ctx)
	}
	wgReceivers.Wait()

	for i := 0; i < chats; i++ {
		mus[i].Lock()
		got := append([]string(nil), received[i]...)
		mus[i].Unlock()

		require.Len(t, got, messagesPerChat, "chat %d must receive exactly its own messages, no more, no less", i)
		seen := map[string]bool{}
		for _, id := range got {
			assert.False(t, seen[id], "message %s delivered more than once", id)
			seen[id] = true
			var gotChat int
			var gotSeq int
			_, err := fmt.Sscanf(id, "%d-%d", &gotChat, &gotSeq)
			require.NoError(t, err)
			assert.Equal(t, i, gotChat, "chat %d received a message meant for a different chat: %s", i, id)
		}
	}
}
