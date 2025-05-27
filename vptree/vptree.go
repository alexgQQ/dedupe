package vptree

import (
	"container/heap"
	"dedupe/dhash"
	"dedupe/phash"
	"math"
	"math/rand"
)

// I gave a shot at using generics with this but it did not pan out how I would like
// so maybe I should revisit that at some point but for now I'd rather make some progress

// To summarize the issue, I can define a generic hash type that is constrained to the DHash and PHash
// type Item[H phash.PHash | dhash.DHash] struct {
// 		...
//		Hash H
// }
// Subsequent struct (Node, QueueItem, VPTree) then need the hash type instantiated on the 
// underlying Item, like so
// type Node struct {
//		...
// 		Item Item[dhash.DHash]
//		...
// }
// This has to happen on any instantiation of the Item class, so this propegates to the Vptree functions
// the QueueItem and so on and so forth, making me basically have to redefine these for each hash anyway
// I am not totally sure how to approach this properly, and I don't know enough about generics to fix it for now

type Item struct {
	FilePath string
	ID       uint
	Dhash	*dhash.DHash
	Phash	*phash.PHash
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

func distance(a Item, b Item) float64 {
	if a.Dhash != nil {
		return float64(a.Dhash.Hamming(b.Dhash))
	} else {
		return float64(a.Phash.Hamming(b.Phash))
	}
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
