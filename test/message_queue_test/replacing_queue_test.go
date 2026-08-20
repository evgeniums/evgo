package message_queue_test

import (
	"testing"

	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/stretchr/testify/assert"
)

func TestReplacingQueueEmpty(t *testing.T) {
	q := message_queue.NewReplacingQueue[string, string]()

	_, ok := q.Front()
	assert.False(t, ok)
	_, ok = q.Dequeue()
	assert.False(t, ok)
	_, ok = q.Get("missing")
	assert.False(t, ok)
	assert.False(t, q.Update("missing", "x"))
	assert.False(t, q.Remove("missing"))
	assert.Equal(t, 0, q.Depth())
	assert.NotPanics(t, q.DropFront)
	assert.NotPanics(t, q.Clear)
}

func TestReplacingQueueFIFOOrder(t *testing.T) {
	q := message_queue.NewReplacingQueue[string, string]()

	replaced, depth := q.Enqueue("a", "1")
	assert.False(t, replaced)
	assert.Equal(t, 1, depth)

	_, depth = q.Enqueue("b", "2")
	assert.Equal(t, 2, depth)
	_, depth = q.Enqueue("c", "3")
	assert.Equal(t, 3, depth)

	v, ok := q.Front()
	assert.True(t, ok)
	assert.Equal(t, "1", v, "Front must peek without removing")
	assert.Equal(t, 3, q.Depth(), "Front must not change depth")

	v, ok = q.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "1", v)
	assert.Equal(t, 2, q.Depth())

	v, ok = q.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "2", v)

	v, ok = q.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "3", v)

	_, ok = q.Dequeue()
	assert.False(t, ok)
}

func TestReplacingQueueEnqueueExistingKeyReplacesInPlace(t *testing.T) {
	q := message_queue.NewReplacingQueue[string, string]()
	q.Enqueue("a", "1")
	q.Enqueue("b", "2")
	q.Enqueue("c", "3")

	replaced, depth := q.Enqueue("b", "2-updated")
	assert.True(t, replaced)
	assert.Equal(t, 3, depth, "depth must not grow on replace")

	// Position must be unchanged (still 2nd), not moved to the back.
	v, _ := q.Dequeue()
	assert.Equal(t, "1", v)
	v, _ = q.Dequeue()
	assert.Equal(t, "2-updated", v)
	v, _ = q.Dequeue()
	assert.Equal(t, "3", v)
}

func TestReplacingQueueGetUpdateRemove(t *testing.T) {
	q := message_queue.NewReplacingQueue[string, string]()
	q.Enqueue("a", "1")
	q.Enqueue("b", "2")

	v, ok := q.Get("a")
	assert.True(t, ok)
	assert.Equal(t, "1", v)
	assert.Equal(t, 2, q.Depth(), "Get must not remove")

	assert.True(t, q.Update("a", "1-updated"))
	v, _ = q.Get("a")
	assert.Equal(t, "1-updated", v)

	assert.True(t, q.Remove("a"))
	_, ok = q.Get("a")
	assert.False(t, ok)
	assert.Equal(t, 1, q.Depth())

	// "b" must still be reachable and dequeue first, since "a" was removed
	// from the middle, not just marked absent.
	v, ok = q.Front()
	assert.True(t, ok)
	assert.Equal(t, "2", v)
}

func TestReplacingQueueDropFront(t *testing.T) {
	q := message_queue.NewReplacingQueue[string, string]()
	q.Enqueue("a", "1")
	q.Enqueue("b", "2")

	q.DropFront()
	assert.Equal(t, 1, q.Depth())
	_, ok := q.Get("a")
	assert.False(t, ok, "DropFront must also remove the key from random-access lookup")

	v, ok := q.Front()
	assert.True(t, ok)
	assert.Equal(t, "2", v)
}

func TestReplacingQueueClear(t *testing.T) {
	q := message_queue.NewReplacingQueue[string, string]()
	q.Enqueue("a", "1")
	q.Enqueue("b", "2")

	q.Clear()
	assert.Equal(t, 0, q.Depth())
	_, ok := q.Front()
	assert.False(t, ok)

	// Must be reusable after Clear.
	replaced, depth := q.Enqueue("a", "1-again")
	assert.False(t, replaced)
	assert.Equal(t, 1, depth)
}
