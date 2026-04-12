// Package terminal provides PTY terminal management.
package terminal

import "strings"

// LimitQueue is a fixed-capacity FIFO ring buffer of strings.
// When the buffer is full the oldest entry is evicted — identical behaviour to
// the original Node.js LimitQueue used as a terminal output ring buffer.
type LimitQueue struct {
	items    []string
	capacity int
	start    int
	count    int
}

// NewLimitQueue creates a new LimitQueue with the given capacity.
func NewLimitQueue(capacity int) *LimitQueue {
	return &LimitQueue{
		items:    make([]string, capacity),
		capacity: capacity,
	}
}

// Push appends an item, evicting the oldest entry if full.
func (q *LimitQueue) Push(item string) {
	if q.count == q.capacity {
		// Overwrite oldest slot.
		q.items[q.start] = item
		q.start = (q.start + 1) % q.capacity
	} else {
		q.items[(q.start+q.count)%q.capacity] = item
		q.count++
	}
}

// Len returns the number of items currently stored.
func (q *LimitQueue) Len() int { return q.count }

// Join concatenates all buffered strings in insertion order.
func (q *LimitQueue) Join() string {
	if q.count == 0 {
		return ""
	}
	var sb strings.Builder
	for i := 0; i < q.count; i++ {
		sb.WriteString(q.items[(q.start+i)%q.capacity])
	}
	return sb.String()
}
