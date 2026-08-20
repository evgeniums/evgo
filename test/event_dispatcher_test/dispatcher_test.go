package event_dispatcher_test

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/event_dispatcher"
	"github.com/evgeniums/evgo/pkg/event_dispatcher/default_event_dispatcher"
	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _, testBasePath, _, _ = runtime.Caller(0)
var testDir = filepath.Dir(testBasePath)

func initApp(t *testing.T, configFile string) app_context.Context {
	t.Helper()
	return test_utils.InitAppContextNoDb(t, testDir, configFile)
}

func newDispatcher(t *testing.T, configFile string, opt ...default_event_dispatcher.DispatcherOptions) (*default_event_dispatcher.DispatcherBase, app_context.Context) {
	t.Helper()
	app := initApp(t, configFile)
	d := default_event_dispatcher.New(opt...)
	require.NoError(t, d.Init(app, ""))
	return d, app
}

// skip is a sentinel for newEvent/newKey: a selector position holding this
// value is left unset (None) rather than set to Some(""). A literal "" would
// be a distinct, set value (see TestEventKeyGetSelectorsReflectsSetState's
// message_queue_test analogue, Some("") != None).
const skip = "\x00skip\x00"

func newEvent(selectors ...string) *event_dispatcher.Event {
	e := &event_dispatcher.Event{}
	for i, s := range selectors {
		if s == skip {
			continue
		}
		e.SetSelector(i, s)
	}
	return e
}

func newKey(selectors ...string) event_dispatcher.EventKey {
	k := event_dispatcher.EventKey{}
	for i, s := range selectors {
		if s == skip {
			continue
		}
		k.SetSelector(i, s)
	}
	return k
}

func TestDispatcherPublishSubscribeDelivers(t *testing.T) {
	d, app := newDispatcher(t, "default.jsonc")
	defer app.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub, err := d.Subscribe(ctx, newKey("chat", "1"))
	require.NoError(t, err)
	defer sub.Unsubscribe(ctx)

	event := newEvent("chat", "1")
	event.MessageType = "created"
	require.NoError(t, d.Publish(ctx, event))

	sub.Next()
	select {
	case v := <-sub.Channel():
		wrapper, ok := v.(event_dispatcher.EventWrapper)
		require.True(t, ok)
		assert.Equal(t, "created", wrapper.MessageType)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for delivery")
	}
}

func TestDispatcherNonMatchingSelectorsDeliverNothing(t *testing.T) {
	d, app := newDispatcher(t, "default.jsonc")
	defer app.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub, err := d.Subscribe(ctx, newKey("chat", "1"))
	require.NoError(t, err)
	defer sub.Unsubscribe(ctx)

	require.NoError(t, d.Publish(ctx, newEvent("chat", "2")))

	select {
	case v := <-sub.Channel():
		t.Fatalf("unexpected delivery for non-matching selectors: %v", v)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestDispatcherUnsubscribeStopsDelivery(t *testing.T) {
	d, app := newDispatcher(t, "default.jsonc")
	defer app.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub, err := d.Subscribe(ctx, newKey("chat", "1"))
	require.NoError(t, err)

	sub.Unsubscribe(ctx)

	require.NoError(t, d.Publish(ctx, newEvent("chat", "1")))

	select {
	case v, ok := <-sub.Channel():
		assert.False(t, ok, "channel must be closed, not deliver a message, after Unsubscribe: got %v", v)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Unsubscribe must close the consumer's feeder channel promptly")
	}
}

// TestDispatcherLevelTrieFindsSubsumedSubscriber is the B8 regression this
// session's message_queue work never had a test for at the event_dispatcher
// layer: with INMEM_LEVEL_TRIE=true in config, DispatcherBase.Init must
// actually build a LevelTrie (the original bug built a SelectorTrie
// regardless of the flag, and separately discarded any custom registry via
// an inverted nil-check in NewInmemMq). A subsuming subscriber (selector 1
// left unset) must still be found despite a competing sibling registered
// under the same prefix with selector 1 set - the exact shape a plain
// SelectorTrie's non-backtracking walk would hide it behind.
func TestDispatcherLevelTrieFindsSubsumedSubscriber(t *testing.T) {
	d, app := newDispatcher(t, "level_trie.jsonc")
	defer app.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	generalSub, err := d.Subscribe(ctx, newKey("chat", skip, "created"))
	require.NoError(t, err)
	defer generalSub.Unsubscribe(ctx)

	siblingSub, err := d.Subscribe(ctx, newKey("chat", "1", "typing"))
	require.NoError(t, err)
	defer siblingSub.Unsubscribe(ctx)

	require.NoError(t, d.Publish(ctx, newEvent("chat", "1", "created")))

	select {
	case v := <-generalSub.Channel():
		_, ok := v.(event_dispatcher.EventWrapper)
		assert.True(t, ok)
	case <-time.After(2 * time.Second):
		t.Fatal("with INMEM_LEVEL_TRIE=true, a subsuming subscriber must still be found despite a competing sibling")
	}

	select {
	case v := <-siblingSub.Channel():
		t.Fatalf("sibling subscribed to a different selector must not receive this event: %v", v)
	case <-time.After(200 * time.Millisecond):
	}
}

// countingRegistry wraps a real AttributeRegistry and counts calls, so tests
// can prove a custom ConsumerRegistryBuilder's returned object is actually
// used for routing - not just invoked once and then silently discarded
// (the original NewInmemMq bug this mirrors at the builder-override level).
type countingRegistry struct {
	inner     message_queue.AttributeRegistry[event_dispatcher.EventConsumer]
	registers int
	finds     int
}

func (r *countingRegistry) Register(item message_queue.Matchable, obj event_dispatcher.EventConsumer) (*message_queue.RegistrySubscription, error) {
	r.registers++
	return r.inner.Register(item, obj)
}

func (r *countingRegistry) Unregister(sub *message_queue.RegistrySubscription) event_dispatcher.EventConsumer {
	return r.inner.Unregister(sub)
}

func (r *countingRegistry) Find(key message_queue.Matchable) []event_dispatcher.EventConsumer {
	r.finds++
	return r.inner.Find(key)
}

func TestDispatcherConsumerRegistryBuilderIsActuallyUsed(t *testing.T) {
	app := initApp(t, "default.jsonc")
	defer app.Close()

	registry := &countingRegistry{inner: message_queue.NewSelectorTrie[event_dispatcher.EventConsumer](event_dispatcher.MaxSelectors)}
	builderCalls := 0
	opt := default_event_dispatcher.DispatcherOptions{
		ConsumerRegistryBuilder: func(maxSelectors int) (event_dispatcher.EventConsumerRegistry, error) {
			builderCalls++
			assert.Equal(t, event_dispatcher.MaxSelectors, maxSelectors)
			return registry, nil
		},
	}
	d := default_event_dispatcher.New(opt)
	require.NoError(t, d.Init(app, ""))
	assert.Equal(t, 1, builderCalls, "ConsumerRegistryBuilder must be invoked exactly once during Init")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub, err := d.Subscribe(ctx, newKey("x"))
	require.NoError(t, err)
	defer sub.Unsubscribe(ctx)
	require.NoError(t, d.Publish(ctx, newEvent("x")))

	select {
	case <-sub.Channel():
	case <-time.After(2 * time.Second):
		t.Fatal("subscription through a custom ConsumerRegistryBuilder must still receive matching publishes")
	}

	assert.Positive(t, registry.registers, "Subscribe must have called Register on the custom registry instance")
	assert.Positive(t, registry.finds, "Publish must have called Find on the custom registry instance")
}

func TestDispatcherSubscriberBuilderErrorPropagates(t *testing.T) {
	app := initApp(t, "default.jsonc")
	defer app.Close()

	wantErr := errors.New("subscriber builder boom")
	opt := default_event_dispatcher.DispatcherOptions{
		SubscriberBuilder: func() (event_dispatcher.EventSubscriber, error) {
			return nil, wantErr
		},
	}
	d := default_event_dispatcher.New(opt)
	require.NoError(t, d.Init(app, ""))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := d.Subscribe(ctx, newKey("x"))
	assert.ErrorIs(t, err, wantErr)
}

func TestDispatcherConsumerFeederBuilderErrorPropagates(t *testing.T) {
	app := initApp(t, "default.jsonc")
	defer app.Close()

	wantErr := errors.New("feeder builder boom")
	opt := default_event_dispatcher.DispatcherOptions{
		ConsumerFeederBuilder: func() (event_dispatcher.EventConsumerFeeder, error) {
			return nil, wantErr
		},
	}
	d := default_event_dispatcher.New(opt)
	require.NoError(t, d.Init(app, ""))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := d.Subscribe(ctx, newKey("x"))
	assert.ErrorIs(t, err, wantErr)
}

func TestDispatcherConsumerQueueBuilderErrorPropagates(t *testing.T) {
	app := initApp(t, "default.jsonc")
	defer app.Close()

	wantErr := errors.New("queue builder boom")
	opt := default_event_dispatcher.DispatcherOptions{
		ConsumerQueueBuilder: func() (event_dispatcher.EventConsumerQueue, error) {
			return nil, wantErr
		},
	}
	d := default_event_dispatcher.New(opt)
	require.NoError(t, d.Init(app, ""))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := d.Subscribe(ctx, newKey("x"))
	assert.ErrorIs(t, err, wantErr)
}
