package vptree

import (
	"container/heap"
	"dedupe/hash"
	"math"
	"math/rand"
)

// I gave a shot at using generics with this but it did not pan out how I would like
// mostly due to the need of the type instantiation with how it fits in the structs
// It seemed much simpler to just represent the hashes as an array and drop the extra types

type Item struct {
	FilePath string
	ID       uint
	Hashes   []uint64
}

type Node struct {
	Item      Item
	Threshold float64
	Left      *Node
	Right     *Node
}

type QueueItem struct {
	Item Item
	Dist float64
}

func distance(a, b Item) float64 {
	if len(a.Hashes) != len(b.Hashes) {
		panic("The hash sizes must be the same")
	}
	var dist int
	for i, h := range a.Hashes {
		dist += hash.Hamming(h, b.Hashes[i])
	}
	return float64(dist)
}

type VPTree struct {
	root *Node
}

func New(items []Item) *VPTree {
	t := &VPTree{}
	t.root = t.build(items)
	return t
}

func (vp *VPTree) Items() []Item {
	var items []Item
	var traverse func(n *Node)

	traverse = func(n *Node) {
		items = append(items, n.Item)
		if n.Left != nil {
			traverse(n.Left)
		}
		if n.Right != nil {
			traverse(n.Right)
		}
	}

	traverse(vp.root)
	return items
}

func (vp *VPTree) Search(target Item, k int) ([]Item, []float64) {
	var results []Item
	var distances []float64

	q := make(queue, 0, k)

	tau := math.MaxFloat64
	vp.search(vp.root, tau, target, k, &q)

	for q.Len() > 0 {
		hi := heap.Pop(&q)
		results = append(results, hi.(*QueueItem).Item)
		distances = append(distances, hi.(*QueueItem).Dist)
	}

	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
		distances[i], distances[j] = distances[j], distances[i]
	}
	return results, distances
}

func (vp *VPTree) Within(target Item, radius float64) ([]Item, []float64) {
	var results []Item
	var distances []float64

	q := make(queue, 0, 100)

	tau := radius
	vp.within(vp.root, tau, target, &q)

	for q.Len() > 0 {
		hi := heap.Pop(&q)
		item := hi.(*QueueItem).Item
		if item.ID != target.ID {
			results = append(results, item)
			distances = append(distances, hi.(*QueueItem).Dist)
		}
	}

	// For my use case I don't think I even need these sorted
	// for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
	// 	results[i], results[j] = results[j], results[i]
	// 	distances[i], distances[j] = distances[j], distances[i]
	// }
	return results, distances
}

func (vp *VPTree) build(items []Item) *Node {
	// Since this is called recursively there could be an empty slice that comes through here
	if len(items) == 0 {
		return nil
	}

	n := &Node{}
	idx := rand.Intn(len(items))
	n.Item = items[idx]
	items[idx], items = items[len(items)-1], items[:len(items)-1]

	if len(items) > 0 {
		median := len(items) / 2
		pivotDist := distance(items[median], n.Item)
		items[median], items[len(items)-1] = items[len(items)-1], items[median]

		storeIndex := 0
		for i := 0; i < len(items)-1; i++ {
			if distance(items[i], n.Item) <= pivotDist {
				items[storeIndex], items[i] = items[i], items[storeIndex]
				storeIndex++
			}
		}
		items[len(items)-1], items[storeIndex] = items[storeIndex], items[len(items)-1]
		median = storeIndex

		n.Threshold = pivotDist
		n.Left = vp.build(items[:median])
		n.Right = vp.build(items[median:])
	}
	return n
}

func (vp *VPTree) search(n *Node, tau float64, target Item, k int, q *queue) {
	// This comes through as nil when we've reached the end of a branch
	if n == nil {
		return
	}

	dist := distance(n.Item, target)

	if dist < tau {
		if q.Len() == k {
			heap.Pop(q)
		}
		heap.Push(q, &QueueItem{n.Item, dist})
		if q.Len() == k {
			tau = q.Top().(*QueueItem).Dist
		}
	}

	if n.Left == nil && n.Right == nil {
		return
	}

	if dist < n.Threshold {
		if dist-tau <= n.Threshold {
			vp.search(n.Left, tau, target, k, q)
		}

		if dist+tau >= n.Threshold {
			vp.search(n.Right, tau, target, k, q)
		}
	} else {
		if dist+tau >= n.Threshold {
			vp.search(n.Right, tau, target, k, q)
		}

		if dist-tau <= n.Threshold {
			vp.search(n.Left, tau, target, k, q)
		}
	}
}

func (vp *VPTree) within(n *Node, tau float64, target Item, q *queue) {
	// This comes through as nil when we've reached the end of a branch
	if n == nil {
		return
	}

	dist := distance(n.Item, target)

	if dist < tau {
		heap.Push(q, &QueueItem{n.Item, dist})
	}

	if n.Left == nil && n.Right == nil {
		return
	}

	if dist < n.Threshold {
		if dist-tau <= n.Threshold {
			vp.within(n.Left, tau, target, q)
		}

		if dist+tau >= n.Threshold {
			vp.within(n.Right, tau, target, q)
		}
	} else {
		if dist+tau >= n.Threshold {
			vp.within(n.Right, tau, target, q)
		}

		if dist-tau <= n.Threshold {
			vp.within(n.Left, tau, target, q)
		}
	}
}
