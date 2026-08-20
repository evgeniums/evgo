package message_queue

import (
	"sync"
	"sync/atomic"
)

// Code is generated with help of Google AI basing on my explicit and thorough description and refinings of the algorithm and optimizations

// LevelTrie implements a hierarchical partial matching.
// Partial search by prefix can be done in LevelTrie, examples:
// ("A","B") will be found by ("A","B","C")
// (Null,"B") will be found by (Null,"B","C")
// ("A",Null,"C") will be found by ("A","B","C") only if there is no other object starting with ("A","B")
// (Null,Null,"C") will not be found by ("A","B","C")
// Container is suited only for prefix search and might work unstable with Null selectors of registered objects depending on existence of other objects in container.
// For more generic and stable search use LevelTrie
// SelectorTrie is faster for prefix search: O(K) in SelectorTrie vs O(2^K) worse case in LevelTrie
// Use it only for exact prefix search with support of Null selectors at the same level both in object and lookup keys.
// Do not use it for wildcard search.
type SelectorTrie[T any] struct {
	mu           sync.RWMutex
	maxSelectors int
	root         *node[T]
	indexCounter atomic.Uint64
}

func NewSelectorTrie[T any](maxSelectors int) *SelectorTrie[T] {
	return &SelectorTrie[T]{
		maxSelectors: maxSelectors,
		root:         newNode[T](),
	}
}

// anyRemainingSet checks if there are any 'Some' selectors from 'start' onwards.
func (st *SelectorTrie[T]) anyRemainingSet(s []Optional[string], start int) bool {
	for i := start; i < len(s); i++ {
		if s[i].IsSet {
			return true
		}
	}
	return false
}

// Register adds an object to the trie. It stops at the last 'Some' selector for efficiency.
func (st *SelectorTrie[T]) Register(item Matchable, obj T) (*RegistrySubscription, error) {
	selectors := item.GetSelectors()
	st.mu.Lock()
	defer st.mu.Unlock()

	// path records only the prefix actually descended, so that Unregister can
	// retrace exactly the same nodes Register visited.
	subscription := &RegistrySubscription{index: st.indexCounter.Add(1)}

	curr := st.root
	for i := 0; i < st.maxSelectors && i < len(selectors); i++ {
		if !st.anyRemainingSet(selectors, i) {
			break
		}
		s := selectors[i]
		subscription.path = append(subscription.path, s)
		child, ok := curr.children[s]
		if !ok {
			child = newNode[T]()
			curr.children[s] = child
		}
		curr = child
	}

	curr.objects[subscription.index] = obj
	return subscription, nil
}

// pathStep records the parent node and the selector used to descend from it,
// so that Unregister can prune the exact nodes it walked through.
type pathStep[T any] struct {
	parent   *node[T]
	selector Optional[string]
}

// Unregister removes a previously registered object from the trie and returns it
// (the zero value of T if the subscription is nil, unknown, or already removed).
// It retraces exactly the path Register descended (see pathStep) and then prunes
// bottom-up any node left with no objects and no children.
func (st *SelectorTrie[T]) Unregister(subscription *RegistrySubscription) T {
	var zero T
	if subscription == nil {
		return zero
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	path := make([]pathStep[T], 0, len(subscription.path))
	curr := st.root
	for _, selector := range subscription.path {
		next, ok := curr.children[selector]
		if !ok {
			// Path diverges from what Register created - nothing to remove.
			return zero
		}
		path = append(path, pathStep[T]{parent: curr, selector: selector})
		curr = next
	}

	found, ok := curr.objects[subscription.index]
	if !ok {
		return zero
	}
	delete(curr.objects, subscription.index)

	// Prune bottom-up: a node is removed from its parent only while it is
	// completely empty (no objects of its own and no remaining children).
	for i := len(path) - 1; i >= 0; i-- {
		if len(curr.objects) != 0 || len(curr.children) != 0 {
			break
		}
		parent := path[i].parent
		delete(parent.children, path[i].selector)
		curr = parent
	}

	return found
}

func appendValues[T any](results []T, m map[uint64]T) []T {
	for _, v := range m {
		results = append(results, v)
	}
	return results
}

// Find retrieves objects. Logic:
// 1. Reg(Specific) + Lookup(General) = NO MATCH
// 2. Reg(General) + Lookup(Specific) = MATCH (Subsumption)
func (st *SelectorTrie[T]) Find(key Matchable) []T {
	selectors := key.GetSelectors()
	st.mu.RLock()
	defer st.mu.RUnlock()

	var results []T
	curr := st.root
	results = appendValues(results, curr.objects)

	for i := 0; i < st.maxSelectors && i < len(selectors); i++ {
		s := selectors[i]

		// 1. Try Specific Path (If Lookup has 'Some', check if Reg has 'Some')
		if s.IsSet {
			if next, ok := curr.children[s]; ok {
				curr = next
				results = appendValues(results, curr.objects)
				continue
			}
		}

		// 2. Fallback to General Path (If Reg has 'None', it accepts any Lookup value)
		if next, ok := curr.children[None[string]()]; ok {
			curr = next
			results = appendValues(results, curr.objects)
			continue
		}

		// 3. No path matches (Lookup is missing a required specific selector)
		break
	}
	return results
}

// --- Example Usage ---

// type Entry struct {
// 	name      string
// 	selectors []Optional[string]
// }

// func (e *Entry) GetSelectors() []Optional[string] { return e.selectors }

// func main() {
// 	trie := NewSelectorTrie[string](3)

// 	// Reg: US -> Auth -> Prod
// 	trie.Register(&Entry{"Specific-Sub", []Optional[string]{Some("US"), Some("Auth"), Some("Prod")}}, "Sub-1")
// 	// Reg: US -> None -> Prod
// 	trie.Register(&Entry{"General-Sub", []Optional[string]{Some("US"), None[string](), Some("Prod")}}, "Sub-2")

// 	// Case 1: Specific Lookup
// 	// Should match BOTH Sub-1 (Exact) and Sub-2 (Subsumed)
// 	lookup1 := &Entry{selectors: []Optional[string]{Some("US"), Some("Auth"), Some("Prod")}}
// 	fmt.Println("Lookup Specific:", trie.Find(lookup1)) // [Sub-1 Sub-2]

// 	// Case 2: General Lookup
// 	// Should NOT match Sub-1 (missing 'Auth'), but matches Sub-2 (None path)
// 	lookup2 := &Entry{selectors: []Optional[string]{Some("US"), None[string](), Some("Prod")}}
// 	fmt.Println("Lookup General:", trie.Find(lookup2)) // [Sub-2]
// }
