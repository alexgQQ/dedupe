package vptree

// max heap priority queue
type queue []*QueueItem

type QueueItem struct {
	item Item
	dist float64
}

func (q queue) Len() int { return len(q) }

func (q queue) Less(i, j int) bool {
	return q[i].dist > q[j].dist
}

func (q queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q *queue) Push(i any) {
	item := i.(*QueueItem)
	*q = append(*q, item)
}

func (q *queue) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	*q = old[0 : n-1]
	return item
}

func (q queue) Top() any {
	return q[0]
}
