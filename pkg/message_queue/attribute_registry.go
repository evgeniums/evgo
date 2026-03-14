package message_queue

type RegistrySubscription struct {
	index uint64
	path  []Optional[string]
}

// AttributeRegistry defines the standard operations for a
// hierarchical, key-addressable Pub-Sub container.
type AttributeRegistry[T any] interface {
	// Register adds an object to the container based on its selector path.
	Register(item Matchable, obj T) (*RegistrySubscription, error)

	// Unregister object
	Unregister(subscription *RegistrySubscription)

	// Find retrieves all objects whose registered paths match
	// the provided lookup key (including subsumed/general matches).
	Find(key Matchable) []T
}

// Optional mimics std::optional to handle "Not Set" values without pointers.
type Optional[T any] struct {
	Value T
	IsSet bool
}

func Some[T any](v T) Optional[T] { return Optional[T]{Value: v, IsSet: true} }
func None[T any]() Optional[T]    { return Optional[T]{IsSet: false} }

// Matchable is the interface for both Objects (Subscribers) and Keys (Publishers).
type Matchable interface {
	GetSelectors() []Optional[string]
}

type node[T any] struct {
	children map[Optional[string]]*node[T]
	objects  map[uint64]T
}

func newNode[T any]() *node[T] {
	return &node[T]{children: make(map[Optional[string]]*node[T])}
}
