package message_queue_test

import (
	"testing"

	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevelTrieBranchingSubsumption(t *testing.T) {
	trie := message_queue.NewLevelTrie[string](3)

	_, err := trie.Register(key("A", null, "C"), "prefix-AC")
	require.NoError(t, err)
	_, err = trie.Register(key("A", "B"), "prefix-AB")
	require.NoError(t, err)
	_, err = trie.Register(key(null, null, "C"), "wildcard-C")
	require.NoError(t, err)

	// From the package doc comment: ("A",Null,"C") is found by ("A","B","C"),
	// as is ("A","B") (a shorter registration matching a longer lookup), and
	// (Null,Null,"C") - none of these require backtracking-free luck the way
	// SelectorTrie does.
	got := trie.Find(key("A", "B", "C"))
	assert.ElementsMatch(t, []string{"prefix-AC", "prefix-AB", "wildcard-C"}, got)
}

func TestLevelTrieNoFalsePositive(t *testing.T) {
	trie := message_queue.NewLevelTrie[string](2)
	_, err := trie.Register(key("A", "B"), "sub")
	require.NoError(t, err)

	assert.Empty(t, trie.Find(key("A", "Y")))
	assert.Empty(t, trie.Find(key("X", "B")))
}

func TestLevelTrieNoDuplicates(t *testing.T) {
	trie := message_queue.NewLevelTrie[string](2)
	_, err := trie.Register(key(null, null), "catch-all")
	require.NoError(t, err)

	got := trie.Find(key("A", "B"))
	assert.Equal(t, []string{"catch-all"}, got, "must not collect the same object twice across branches")
}

// TestLevelTrieFindsWhatSelectorTrieMisses is the concrete case the package
// comment warns about: "(A,Null,C) will be found by (A,B,C) only if there is
// no other object starting with (A,B)". Registering an unrelated sibling
// under (A,B,*) creates a Some("B") child at the "A" node; SelectorTrie's
// greedy, non-backtracking walk then always takes that specific branch over
// the None branch "general" lives under, so a (A,B,C) lookup never reaches
// it. LevelTrie explores both branches and does not have this limitation.
func TestLevelTrieFindsWhatSelectorTrieMisses(t *testing.T) {
	selTrie := message_queue.NewSelectorTrie[string](3)
	lvlTrie := message_queue.NewLevelTrie[string](3)

	for _, reg := range []func(message_queue.Matchable, string){
		func(k message_queue.Matchable, obj string) { _, _ = selTrie.Register(k, obj) },
		func(k message_queue.Matchable, obj string) { _, _ = lvlTrie.Register(k, obj) },
	} {
		reg(key("A", null, "C"), "general")     // matches (A, *, C)
		reg(key("A", "B", "X"), "unrelated-B-X") // creates a competing Some("B") child at "A"
	}

	lookup := key("A", "B", "C")
	assert.Empty(t, selTrie.Find(lookup),
		"SelectorTrie's greedy Some(B) branch shadows the None branch \"general\" is registered under")
	assert.Equal(t, []string{"general"}, lvlTrie.Find(lookup),
		"LevelTrie must find \"general\" via its None branch even though a Some(B) sibling also exists")
}
