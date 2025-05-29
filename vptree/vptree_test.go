package vptree

import (
	"dedupe/hash"
	"math/rand"
	"slices"
	"testing"
)

func TestVPTreeWithin(t *testing.T) {
	var samples []*Item
	var expected []uint

	// Create some sample points and find the ones that should be
	// returned from the tree. Points are from 0-255 so the max hamming
	// distance between hashes is 8
	threshold := float64(rand.Intn(6-3+1) + 3)
	i := 0
	for i <= 0xff {
		hashes := make([]uint64, 1)
		hashes[0] = uint64(i)
		item := Item{ID: uint(i), Hashes: hashes}
		samples = append(samples, &item)
		i++
	}

	target := samples[rand.Intn(len(samples))]

	for _, item := range samples {
		if item.ID == target.ID {
			continue
		}
		dist := hash.Hamming(target.Hashes[0], item.Hashes[0])
		if dist < int(threshold) {
			expected = append(expected, item.ID)
		}
	}

	tree := *New(samples)
	found, distances := tree.Within(*target, threshold)

	if len(found) != len(expected) {
		t.Errorf("Within returned %d results but %d were expected", len(found), len(expected))
	}
	for i, result := range found {
		dist := hash.Hamming(target.Hashes[0], result.Hashes[0])
		if dist != int(distances[i]) {
			t.Error("Within returned an item with an unexpected hamming distance")
		}
		if !slices.Contains(expected, result.ID) {
			t.Error("Within returned an unexpected item")
		}
	}
}
