package message_queue

// RandomAccessQueue defines the behavior for a thread-safe,
// key-addressable FIFO storage.
type RandomAccessQueue[K comparable, V any] interface {
	// Enqueue adds an item to the back.
	// Returns (replaced_existing,depth)
	Enqueue(key K, value V) (bool, int)

	// Dequeue removes and returns the front item.
	Dequeue() (V, bool)

	// Get retrieves a value by its key without removing it.
	Get(key K) (V, bool)

	// Update modifies an existing key's value in place.
	Update(key K, value V) bool

	// Remove deletes an item by key from any position.
	Remove(key K) bool

	// Clear empties the queue.
	Clear()

	// Depth returns the current number of elements.
	Depth() int

	Front() (V, bool)
	DropFront()
}
