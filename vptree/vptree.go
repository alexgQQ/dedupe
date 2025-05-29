package vptree

import (
	"container/heap"
	"dedupe/hash"
	"iter"
	"math/rand"
	"sync"
)

// This keeps track of each filepath with it's corresponding Item.ID.
// It works with a mutex as we issue goroutines when gathering hashes from files
// and need the ID values to be unique and safe from race conditions.

type FileMapper struct {
	mu    sync.Mutex
	count uint
	files []string
}

func (c *FileMapper) addFile(file string) uint {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files = append(c.files, file)
	c.count++
	return c.count
}

func (c *FileMapper) ByID(id uint) (string, error) {
	// A panic may be more appropriate if the id is off then something is very wrong
	// if int(id) > len(c.files) {
	// 	return "", fmt.Errorf("integer %d is outside of item array", id)
	// }
	return c.files[id-1], nil
}

// 	Each Item represents a hash for an image file.
//	An unique ID is used in place of the file path and should be 1-indexed to avoid zero valued collisions.
// 	The hash is either a single 64 bit value (dct) or a 128 bit value (dhash) as two 64 bit values.

type Item struct {
	ID     uint
	Hashes []uint64
}

func NewItem(file string, fileMap *FileMapper, hashes ...uint64) *Item {
	item := Item{Hashes: hashes}
	item.ID = fileMap.addFile(file)
	return &item
}

type Node struct {
	item      Item
	threshold float64
	left      *Node
	right     *Node
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

func New(items []*Item) *VPTree {
	t := &VPTree{}
	t.root = t.build(items)
	return t
}

func (n *Node) walk(yield func(Item) bool) bool {
	if n == nil {
		return true
	}
	if !yield(n.item) {
		return false
	}
	if !n.left.walk(yield) {
		return false
	}
	return n.right.walk(yield)
}

func (vp *VPTree) All() iter.Seq[Item] {
	return func(yield func(Item) bool) {
		vp.root.walk(yield)
	}
}

func (vp *VPTree) Within(target Item, radius float64) ([]Item, []float64) {
	var results []Item
	var distances []float64

	q := make(queue, 0, 100)

	tau := radius
	vp.within(vp.root, tau, target, &q)

	for q.Len() > 0 {
		hi := heap.Pop(&q)
		item := hi.(*QueueItem).item
		if item.ID != target.ID {
			results = append(results, item)
			distances = append(distances, hi.(*QueueItem).dist)
		}
	}

	// For my use case I don't think I even need these sorted
	// for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
	// 	results[i], results[j] = results[j], results[i]
	// 	distances[i], distances[j] = distances[j], distances[i]
	// }
	return results, distances
}

func (vp *VPTree) build(items []*Item) *Node {
	// Since this is called recursively there could be an empty slice that comes through here
	if len(items) == 0 {
		return nil
	}

	n := &Node{}
	idx := rand.Intn(len(items))
	n.item = *items[idx]
	items[idx], items = items[len(items)-1], items[:len(items)-1]

	if len(items) > 0 {
		median := len(items) / 2
		pivotDist := distance(*items[median], n.item)
		items[median], items[len(items)-1] = items[len(items)-1], items[median]

		storeIndex := 0
		for i := 0; i < len(items)-1; i++ {
			if distance(*items[i], n.item) <= pivotDist {
				items[storeIndex], items[i] = items[i], items[storeIndex]
				storeIndex++
			}
		}
		items[len(items)-1], items[storeIndex] = items[storeIndex], items[len(items)-1]
		median = storeIndex

		n.threshold = pivotDist
		n.left = vp.build(items[:median])
		n.right = vp.build(items[median:])
	}
	return n
}

func (vp *VPTree) within(n *Node, tau float64, target Item, q *queue) {
	// This comes through as nil when we've reached the end of a branch
	if n == nil {
		return
	}

	dist := distance(n.item, target)

	if dist < tau {
		heap.Push(q, &QueueItem{n.item, dist})
	}

	if n.left == nil && n.right == nil {
		return
	}

	if dist < n.threshold {
		if dist-tau <= n.threshold {
			vp.within(n.left, tau, target, q)
		}

		if dist+tau >= n.threshold {
			vp.within(n.right, tau, target, q)
		}
	} else {
		if dist+tau >= n.threshold {
			vp.within(n.right, tau, target, q)
		}

		if dist-tau <= n.threshold {
			vp.within(n.left, tau, target, q)
		}
	}
}

// I don't actually need this functionality but it is a good example
// for using this as a k item search
// func (vp *VPTree) Search(target Item, k int) ([]Item, []float64) {
// 	var results []Item
// 	var distances []float64

// 	q := make(queue, 0, k)

// 	tau := math.MaxFloat64
// 	vp.search(vp.root, tau, target, k, &q)

// 	for q.Len() > 0 {
// 		hi := heap.Pop(&q)
// 		results = append(results, hi.(*QueueItem).Item)
// 		distances = append(distances, hi.(*QueueItem).Dist)
// 	}

// 	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
// 		results[i], results[j] = results[j], results[i]
// 		distances[i], distances[j] = distances[j], distances[i]
// 	}
// 	return results, distances
// }

// func (vp *VPTree) search(n *Node, tau float64, target Item, k int, q *queue) {
// 	// This comes through as nil when we've reached the end of a branch
// 	if n == nil {
// 		return
// 	}

// 	dist := distance(n.Item, target)

// 	if dist < tau {
// 		if q.Len() == k {
// 			heap.Pop(q)
// 		}
// 		heap.Push(q, &QueueItem{n.Item, dist})
// 		if q.Len() == k {
// 			tau = q.Top().(*QueueItem).Dist
// 		}
// 	}

// 	if n.Left == nil && n.Right == nil {
// 		return
// 	}

// 	if dist < n.Threshold {
// 		if dist-tau <= n.Threshold {
// 			vp.search(n.Left, tau, target, k, q)
// 		}

// 		if dist+tau >= n.Threshold {
// 			vp.search(n.Right, tau, target, k, q)
// 		}
// 	} else {
// 		if dist+tau >= n.Threshold {
// 			vp.search(n.Right, tau, target, k, q)
// 		}

// 		if dist-tau <= n.Threshold {
// 			vp.search(n.Left, tau, target, k, q)
// 		}
// 	}
// }
