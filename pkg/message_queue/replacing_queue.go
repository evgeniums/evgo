package message_queue

import (
	"container/list"
)

// entry wraps the key and value to allow back-referencing from list elements to map keys
type entry[K comparable, V any] struct {
	key   K
	value V
}

// ReplacingQueue is a thread-safe, generic, random-access queue.
// It keeps map of enqueued items and replaces existing item on conflict.
type ReplacingQueue[K comparable, V any] struct {
	list  *list.List
	table map[K]*list.Element
}

// New initializes a new ReplacingQueue.
func NewReplacingQueue[K comparable, V any]() *ReplacingQueue[K, V] {
	return &ReplacingQueue[K, V]{
		list:  list.New(),
		table: make(map[K]*list.Element),
	}
}

// Enqueue adds an item to the back of the queue.
// If the key already exists, it updates the value and moves the item to the back.
// If element with such key exists it will be replaced in current position in the queue and true will be returned
// Returnes (replaced,depth)
func (q *ReplacingQueue[K, V]) Enqueue(key K, value V) (bool, int) {
	if el, ok := q.table[key]; ok {
		el.Value = entry[K, V]{key, value}
		return true, len(q.table)
	}

	el := q.list.PushBack(entry[K, V]{key, value})
	q.table[key] = el

	return false, len(q.table)
}

// Dequeue removes and returns the item from the front: O(1).
func (q *ReplacingQueue[K, V]) Dequeue() (V, bool) {

	front := q.list.Front()
	if front == nil {
		var zero V
		return zero, false
	}

	e := front.Value.(entry[K, V])
	q.list.Remove(front)
	delete(q.table, e.key)
	return e.value, true
}

func (q *ReplacingQueue[K, V]) Front() (V, bool) {

	front := q.list.Front()
	if front == nil {
		var zero V
		return zero, false
	}

	e := front.Value.(entry[K, V])
	return e.value, true
}

// Get performs a random access lookup by key: O(1).
func (q *ReplacingQueue[K, V]) Get(key K) (V, bool) {

	if el, ok := q.table[key]; ok {
		return el.Value.(entry[K, V]).value, true
	}
	var zero V
	return zero, false
}

// Update modifies the value for an existing key without changing its position: O(1).
func (q *ReplacingQueue[K, V]) Update(key K, value V) bool {

	if el, ok := q.table[key]; ok {
		el.Value = entry[K, V]{key, value}
		return true
	}
	return false
}

// Remove deletes a specific key from anywhere in the queue: O(1).
func (q *ReplacingQueue[K, V]) Remove(key K) bool {

	if el, ok := q.table[key]; ok {
		q.list.Remove(el)
		delete(q.table, key)
		return true
	}
	return false
}

// Clear empties the entire queue.
func (q *ReplacingQueue[K, V]) Clear() {

	q.list.Init()
	q.table = make(map[K]*list.Element)
}

// Depth returns the current number of elements in the queue.
func (q *ReplacingQueue[K, V]) Depth() int {
	return len(q.table)
}

func (q *ReplacingQueue[K, V]) DropFront() {

	front := q.list.Front()
	if front == nil {
		return
	}

	// 1. Get the key from the list element to clean up the map
	e := front.Value.(entry[K, V])
	delete(q.table, e.key)

	// 2. Remove from the list
	q.list.Remove(front)
}
