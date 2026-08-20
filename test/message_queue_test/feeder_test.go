package message_queue_test

import (
	"testing"

	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingProvider records how many times Next() was called.
type countingProvider struct {
	calls int
}

func (p *countingProvider) Next() { p.calls++ }

func TestFeederUntypedPushCapacityAndChannel(t *testing.T) {
	provider := &countingProvider{}
	feeder := message_queue.NewFeeder[string](provider, &message_queue.FeederConfig{
		FEEDER_CHANNEL_DEPTH: 2,
	})

	assert.True(t, feeder.Push("a"))
	assert.True(t, feeder.Push("b"))
	assert.False(t, feeder.Push("c"), "Push must be non-blocking and report false once the channel is full")

	require.NotNil(t, feeder.Channel())
	v := <-feeder.Channel()
	assert.Equal(t, "a", v)

	assert.Nil(t, feeder.TypedChannel(), "typed channel must be nil in untyped mode")
}

func TestFeederTypedChannel(t *testing.T) {
	provider := &countingProvider{}
	feeder := message_queue.NewFeeder[string](provider, &message_queue.FeederConfig{
		FEEDER_TYPED_CHANNEL: true,
		FEEDER_CHANNEL_DEPTH: 1,
	})

	assert.True(t, feeder.Push("a"))
	assert.False(t, feeder.Push("b"), "Push must be non-blocking and report false once the typed channel is full")

	require.NotNil(t, feeder.TypedChannel())
	v := <-feeder.TypedChannel()
	assert.Equal(t, "a", v)

	assert.Nil(t, feeder.Channel(), "untyped channel must be nil in typed mode")
}

func TestFeederDefaultConfig(t *testing.T) {
	provider := &countingProvider{}
	feeder := message_queue.NewFeeder[string](provider)

	// No config passed: falls back to the untyped channel with the package
	// default depth.
	for i := 0; i < message_queue.DEFAULT_FEEDER_CHANNEL_DEPTH; i++ {
		require.True(t, feeder.Push("x"), "push %d must fit within the default depth", i)
	}
	assert.False(t, feeder.Push("overflow"))
}

// TestFeederCloseTypedChannelDoesNotPanic pins the B9 fix: FeederBase.Close
// used to unconditionally close(p.ch), which panics when FEEDER_TYPED_CHANNEL
// is set and p.ch is nil.
func TestFeederCloseTypedChannelDoesNotPanic(t *testing.T) {
	provider := &countingProvider{}
	feeder := message_queue.NewFeeder[string](provider, &message_queue.FeederConfig{
		FEEDER_TYPED_CHANNEL: true,
		FEEDER_CHANNEL_DEPTH: 1,
	})

	assert.NotPanics(t, feeder.Close)

	_, ok := <-feeder.TypedChannel()
	assert.False(t, ok, "typed channel must actually be closed")
}

func TestFeederCloseIsIdempotent(t *testing.T) {
	provider := &countingProvider{}
	feeder := message_queue.NewFeeder[string](provider, &message_queue.FeederConfig{FEEDER_CHANNEL_DEPTH: 1})

	assert.NotPanics(t, feeder.Close)
	assert.NotPanics(t, feeder.Close, "second Close must not panic")
}

func TestFeederCloseUntypedChannelCloses(t *testing.T) {
	provider := &countingProvider{}
	feeder := message_queue.NewFeeder[string](provider, &message_queue.FeederConfig{FEEDER_CHANNEL_DEPTH: 1})

	feeder.Close()
	_, ok := <-feeder.Channel()
	assert.False(t, ok)
}

func TestFeederNextForwardsToProvider(t *testing.T) {
	provider := &countingProvider{}
	feeder := message_queue.NewFeeder[string](provider, &message_queue.FeederConfig{FEEDER_CHANNEL_DEPTH: 1})

	feeder.Next()
	feeder.Next()
	assert.Equal(t, 2, provider.calls)
}
