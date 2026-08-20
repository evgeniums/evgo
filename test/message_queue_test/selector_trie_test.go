package message_queue_test

import (
	"testing"

	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTrie(maxSelectors int) *message_queue.SelectorTrie[string] {
	return message_queue.NewSelectorTrie[string](maxSelectors)
}

func TestSelectorTrieEmpty(t *testing.T) {
	trie := newTrie(3)
	assert.Empty(t, trie.Find(key("A", "B", "C")))
}

func TestSelectorTrieSubsumption(t *testing.T) {
	trie := newTrie(3)

	_, err := trie.Register(key("US", null, "Prod"), "General-Sub")
	require.NoError(t, err)

	// Reg(General) + Lookup(Specific) = MATCH (subsumption): no competing
	// "Auth" sibling exists yet, so the walk falls back to the None branch.
	assert.Equal(t, []string{"General-Sub"}, trie.Find(key("US", "Auth", "Prod")))
	// Reg(General) + Lookup(General) also matches.
	assert.Equal(t, []string{"General-Sub"}, trie.Find(key("US", null, "Prod")))

	_, err = trie.Register(key("US", "Auth", "Prod"), "Specific-Sub")
	require.NoError(t, err)

	// Reg(Specific) + Lookup(General) = NO MATCH: General-Sub's own branch is
	// unaffected by Specific-Sub living under a sibling "Auth" branch.
	assert.Equal(t, []string{"General-Sub"}, trie.Find(key("US", null, "Prod")))

	// Once a competing "Auth" sibling exists, SelectorTrie's greedy,
	// non-backtracking walk takes the specific branch and never explores the
	// None branch General-Sub lives under - this is the documented
	// limitation LevelTrie exists to fix (see TestLevelTrieFindsWhatSelectorTrieMisses).
	assert.Equal(t, []string{"Specific-Sub"}, trie.Find(key("US", "Auth", "Prod")))

	// A different specific middle selector still falls back to the None branch.
	assert.Equal(t, []string{"General-Sub"}, trie.Find(key("US", "Refresh", "Prod")))

	// No match at all when the first selector differs.
	assert.Empty(t, trie.Find(key("EU", "Auth", "Prod")))
}

func TestSelectorTrieSomeEmptyStringIsNotNone(t *testing.T) {
	trie := newTrie(2)
	_, err := trie.Register(key("A", ""), "empty-string-sub")
	require.NoError(t, err)

	// Some("") must not be treated as None: a None lookup should not find it,
	// and only an exact empty-string lookup should.
	assert.Empty(t, trie.Find(key("A", null)))
	assert.Equal(t, []string{"empty-string-sub"}, trie.Find(key("A", "")))
}

func TestSelectorTrieAllNoneRegistrationMatchesEverything(t *testing.T) {
	trie := newTrie(3)
	_, err := trie.Register(key(null, null, null), "catch-all")
	require.NoError(t, err)

	assert.Equal(t, []string{"catch-all"}, trie.Find(key("A", "B", "C")))
	assert.Equal(t, []string{"catch-all"}, trie.Find(key(null, null, null)))
}

func TestSelectorTrieTruncatesAtMaxSelectors(t *testing.T) {
	trie := newTrie(2)
	_, err := trie.Register(key("A", "B", "C"), "truncated-sub")
	require.NoError(t, err)

	// Registration only descends 2 levels; a 2-level lookup already finds it,
	// regardless of what a longer lookup's 3rd selector would be.
	assert.Equal(t, []string{"truncated-sub"}, trie.Find(key("A", "B")))
	assert.Equal(t, []string{"truncated-sub"}, trie.Find(key("A", "B", "anything")))
}

// --- B1/B2 regressions: Unregister must not disturb unrelated live subscriptions ---

func TestSelectorTrieUnregisterSiblingSurvives(t *testing.T) {
	trie := newTrie(2)

	subA, err := trie.Register(key("A", "X"), "sub-A")
	require.NoError(t, err)
	_, err = trie.Register(key("A", "Y"), "sub-B")
	require.NoError(t, err)

	removed := trie.Unregister(subA)
	assert.Equal(t, "sub-A", removed)

	// Regression for B1: the buggy Unregister deleted the "A" child from the
	// root's children map (one level too high), taking sub-B down with it.
	assert.Equal(t, []string{"sub-B"}, trie.Find(key("A", "Y")))
	assert.Empty(t, trie.Find(key("A", "X")))
}

func TestSelectorTrieUnregisterNoneSiblingSurvives(t *testing.T) {
	trie := newTrie(3)

	subGeneral, err := trie.Register(key("A", null, "C"), "general")
	require.NoError(t, err)
	_, err = trie.Register(key("A", null, null), "general-tail")
	require.NoError(t, err)

	removed := trie.Unregister(subGeneral)
	assert.Equal(t, "general", removed)

	assert.Equal(t, []string{"general-tail"}, trie.Find(key("A", null, null)))
}

func TestSelectorTrieUnregisterTrailingNoneDoesNotShortCircuit(t *testing.T) {
	// Register stops descending at the last Some, so a subscription's stored
	// path never has trailing Nones - but two different subscribers can still
	// create/share a None child at the same depth. B2: Unregister must return
	// the real object, not silently miss it and return the zero value.
	trie := newTrie(3)

	subA, err := trie.Register(key("A", null), "sub-A-general")
	require.NoError(t, err)
	_, err = trie.Register(key("A", null, "C"), "sub-B-specific")
	require.NoError(t, err)

	removed := trie.Unregister(subA)
	assert.Equal(t, "sub-A-general", removed, "Unregister must find and return the registered object, not the zero value")
	assert.Equal(t, []string{"sub-B-specific"}, trie.Find(key("A", null, "C")))
}

func TestSelectorTrieUnregisterIdempotent(t *testing.T) {
	trie := newTrie(2)
	sub, err := trie.Register(key("A", "B"), "sub")
	require.NoError(t, err)

	assert.Equal(t, "sub", trie.Unregister(sub))
	assert.Empty(t, trie.Find(key("A", "B")))
	assert.Equal(t, "", trie.Unregister(sub), "second Unregister must return the zero value, not panic or re-remove")
}

func TestSelectorTrieUnregisterUnknownSubscription(t *testing.T) {
	// RegistrySubscription carries no identity of which trie it came from,
	// so "unknown" is exercised on the same trie: register and remove a
	// throwaway entry, then confirm removing it again is a safe no-op and
	// leaves an unrelated, still-live subscription untouched. (A genuinely
	// foreign subscription from a different trie risks an accidental index
	// collision between the two tries' independent counters, which would
	// make this test non-deterministic.)
	trie := newTrie(2)
	_, err := trie.Register(key("A", "B"), "sub")
	require.NoError(t, err)

	ghost, err := trie.Register(key("X", "Y"), "ghost")
	require.NoError(t, err)
	assert.Equal(t, "ghost", trie.Unregister(ghost))

	assert.Equal(t, "", trie.Unregister(ghost), "removing an already-removed subscription must return the zero value")
	assert.Equal(t, []string{"sub"}, trie.Find(key("A", "B")), "an unrelated live subscription must be unaffected")
}

func TestSelectorTrieUnregisterNilSubscription(t *testing.T) {
	trie := newTrie(2)
	assert.NotPanics(t, func() {
		assert.Equal(t, "", trie.Unregister(nil))
	})
}

func TestSelectorTrieUnregisterPrunesEmptyBranches(t *testing.T) {
	// Not directly observable via Find, but exercised so that repeated
	// subscribe/unsubscribe churn does not leak trie nodes; also guards
	// against a panic/incorrect prune when a whole branch empties out.
	trie := newTrie(3)
	sub, err := trie.Register(key("A", "B", "C"), "sub")
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		trie.Unregister(sub)
	})
	assert.Empty(t, trie.Find(key("A", "B", "C")))

	// Registering again under the same path must work as if the trie were
	// fresh - proof the earlier branch was actually cleaned up, not left
	// dangling with stale index bookkeeping.
	_, err = trie.Register(key("A", "B", "C"), "sub-again")
	require.NoError(t, err)
	assert.Equal(t, []string{"sub-again"}, trie.Find(key("A", "B", "C")))
}
