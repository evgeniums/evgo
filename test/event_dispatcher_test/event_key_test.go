package event_dispatcher_test

import (
	"testing"

	"github.com/evgeniums/evgo/pkg/event_dispatcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventKeyLength(t *testing.T) {
	var k event_dispatcher.EventKey
	assert.Equal(t, event_dispatcher.MaxSelectors, k.Length())
}

func TestEventKeySetGetSelector(t *testing.T) {
	var k event_dispatcher.EventKey
	k.SetSelector(0, "account-1")
	k.SetSelector(2, "topic-1")

	v, ok := k.GetSelector(0)
	assert.True(t, ok)
	assert.Equal(t, "account-1", v)

	v, ok = k.GetSelector(1)
	assert.False(t, ok, "an unset selector must report ok=false")
	assert.Equal(t, "", v)

	v, ok = k.GetSelector(2)
	assert.True(t, ok)
	assert.Equal(t, "topic-1", v)
}

func TestEventKeyUnsetSelector(t *testing.T) {
	var k event_dispatcher.EventKey
	k.SetSelector(0, "account-1")

	v, ok := k.GetSelector(0)
	require.True(t, ok)
	require.Equal(t, "account-1", v)

	k.UnsetSelector(0)

	_, ok = k.GetSelector(0)
	assert.False(t, ok)
}

// TestEventKeyGetSelectorOutOfRange pins the added bounds guard: GetSelector
// used to check only the upper bound (i >= Length()), so a negative index
// panicked with a slice-index-out-of-range instead of reporting "not set".
func TestEventKeyGetSelectorOutOfRange(t *testing.T) {
	var k event_dispatcher.EventKey

	assert.NotPanics(t, func() {
		_, ok := k.GetSelector(event_dispatcher.MaxSelectors)
		assert.False(t, ok)
	})
	assert.NotPanics(t, func() {
		_, ok := k.GetSelector(-1)
		assert.False(t, ok)
	})
}

func TestEventKeyGetSelectorsReflectsSetState(t *testing.T) {
	var k event_dispatcher.EventKey
	k.SetSelector(0, "account-1")

	selectors := k.GetSelectors()
	require.Len(t, selectors, event_dispatcher.MaxSelectors)

	assert.True(t, selectors[0].IsSet)
	assert.Equal(t, "account-1", selectors[0].Value)

	for i := 1; i < len(selectors); i++ {
		assert.Falsef(t, selectors[i].IsSet, "selector %d must be unset", i)
	}
}

func TestEventKeyKeyReturnsItself(t *testing.T) {
	var k event_dispatcher.EventKey
	k.SetSelector(0, "account-1")
	k.SetSelector(3, "type-1")

	assert.Equal(t, k, k.Key())
}
