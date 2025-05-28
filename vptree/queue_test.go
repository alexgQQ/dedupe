package vptree

import (
	"container/heap"
	"testing"
)

func TestMaxHeapQueue(t *testing.T) {
	max := 100.0
	q := make(queue, 0, 4)
	heap.Push(&q, &QueueItem{Item{}, 10.0})
	heap.Push(&q, &QueueItem{Item{}, max})
	heap.Push(&q, &QueueItem{Item{}, 50.0})
	if q.Top().(*QueueItem).dist != max {
		t.Error("Top of the queue should be the largest distance")
	}
	max = 110.0
	heap.Push(&q, &QueueItem{Item{}, max})
	if q.Top().(*QueueItem).dist != max {
		t.Error("Top of the queue should be the new largest distance")
	}
}
