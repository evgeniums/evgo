package message_queue

// Code is generated with help of Google AI basing on my explicit and thorough description and refinings of the algorithm and optimizations

// LevelTrie implements a hierarchical multi-attribute matching tree.
// Partial searching can be done in LevelTrie, e.g.
// ("A",Null,"C") will be found by ("A","B","C")
// (Null,Null,"C") will be found by ("A","B","C")
// ("A","B") will be found by ("A","B","C")
type LevelTrie[T any] struct {
	SelectorTrie[T]
}

func NewLevelTrie[T any](maxSelectors int) *LevelTrie[T] {
	return &LevelTrie[T]{
		SelectorTrie[T]{
			maxSelectors: maxSelectors,
			root:         newNode[T](),
		},
	}
}

// Find retrieves all matching objects using a branching recursive search.
// This ensures we catch specific matches AND general (subsuming) matches.
func (lt *LevelTrie[T]) Find(key Matchable) []T {
	selectors := key.GetSelectors()
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	var results []T

	var search func(n *node[T], depth int)
	search = func(n *node[T], depth int) {
		// 1. Collect objects at this node (they match the prefix so far).
		if len(n.objects) > 0 {
			results = appendValues(results, n.objects)
		}

		// 2. Base case: end of tree or key.
		if depth >= lt.maxSelectors || depth >= len(selectors) {
			return
		}

		lookupVal := selectors[depth]

		// 3. Path A: Check for the specific branch if the lookup has a value.
		if lookupVal.IsSet {
			if next, ok := n.children[lookupVal]; ok {
				search(next, depth+1)
			}
		}

		// 4. Path B: Always check the 'None' branch (Subsumption).
		// This finds subscribers who didn't set a selector at this level.
		if next, ok := n.children[None[string]()]; ok {
			search(next, depth+1)
		}
	}

	search(lt.root, 0)
	return results
}
