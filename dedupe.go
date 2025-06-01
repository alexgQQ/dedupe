package dedupe

import (
	"runtime"
	"slices"
	"sync"

	"github.com/alexgQQ/dedupe/hash"
	"github.com/alexgQQ/dedupe/utils"
	"github.com/alexgQQ/dedupe/vptree"
)

var (
	DHASH hash.HashType = hash.DHASH
	DCT   hash.HashType = hash.DCT
)

func imageHash(hashType hash.HashType, img image.Image) (hashes []uint64) {
	if hashType.Equal(hash.DCT) {
		hash := hash.Dct(img)
		hashes = append(hashes, hash)
	} else if hashType.Equal(hash.DHASH) {
		rHash, cHash := hash.Dhash(img)
		hashes = append(hashes, rHash)
		hashes = append(hashes, cHash)
	}
	return
}

func buildTree(files []string, hashType hash.HashType) (*vptree.VPTree, *vptree.FileMapper, error) {
	var wg sync.WaitGroup
	var fileMap vptree.FileMapper

	// By default this will be the runtime.NumCPU but will be GOMAXPROCS if set in the environment
	nWorkers := runtime.GOMAXPROCS(0)
	work := make(chan string)
	results := make(chan *vptree.Item)

	// Spin up nWorkers to process images concurrently
	for _ = range nWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range work {
				img, err := utils.LoadImage(f)
				if err != nil {
					// slog.Error("Error loading image", "file", f, "error", err)
				} else {
					var item *vptree.Item
					switch hashType {
					case hash.DCT:
						item = vptree.NewItem(f, &fileMap, hash.Dct(img))
					case hash.DHASH:
						rHash, cHash := hash.Dhash(img)
						item = vptree.NewItem(f, &fileMap, rHash, cHash)
					}
					results <- item
				}
			}
		}()
	}

	// Handle shifting images onto the worker queue and synchronizing
	go func() {
		for _, f := range files {
			work <- f
		}
		close(work)
		wg.Wait()
		close(results)
	}()

	// Accumulate the computed hashes to build the vptree
	var items []*vptree.Item
	for item := range results {
		items = append(items, item)
	}
	return vptree.New(items), &fileMap
}

func gatherDuplicateIds(tree *vptree.VPTree, threshold float64) ([][]uint, int, error) {
	var total int = 0
	var skip []uint
	var ids [][]uint

	for item := range tree.All() {
		var group []uint
		if slices.Contains(skip, item.ID) {
			continue
		}
		found, _ := tree.Within(item, threshold)
		if len(found) <= 0 {
			continue
		}
		// slog.Info("VPTree found results within item", "item", item, "results", found, "distances", dist, "threshold", threshold)
		group = append(group, item.ID)

		// I've gone back and forth with myself on this
		// do I need to do this reciprocal search?
		// Basically anything subsequently found here would be at max
		// 2x the threshold from the first item, which isn't necessarily wrong
		for _, i := range found {
			group = append(group, i.ID)
			f, _ := tree.Within(i, threshold)
			for _, F := range f {
				group = append(group, F.ID)
			}
			// slog.Info("VPTree found results within item", "item", item, "results", f, "distances", d, "threshold", threshold)
		}
		//
		slices.Sort(group)
		group = slices.Compact(group)
		total += len(group)
		skip = append(skip, group...)
		ids = append(ids, group)
	}

	return ids, total, nil
}

// This could be the major entrypoint if using as a package
// How does that look?
func Duplicates(files []string, hashType hash.HashType) ([][]string, int, error) {
	tree, fileMap := buildTree(files, hashType)
	ids, total, _ := gatherDuplicateIds(tree, hashType.Threshold)

	filegroups := make([][]string, len(ids))
	for i, group := range ids {
		paths := make([]string, len(group))
		for j, id := range group {
			filepath, _ := fileMap.ByID(id)
			paths[j] = filepath
		}
		filegroups[i] = paths
	}
	return filegroups, total, nil
}

func hashVals(hashType hash.HashType, file string) []uint64 {
	img, _ := utils.LoadImage(file)
	vals := make([]uint64, 0)
	switch hashType {
	case hash.DCT:
		hash := hash.Dct(img)
		vals = append(vals, hash)
	case hash.DHASH:
		rHash, cHash := hash.Dhash(img)
		vals = append(vals, rHash)
		vals = append(vals, cHash)
	}
	return vals
}

// I like the variadic syntax of this but it changes the consistency of the hashType arg
func Compare(hashType hash.HashType, target string, files ...string) (found bool, results []string) {
	targetHash := hashVals(hashType, target)
	// There potential to speed this up and run with goroutines here
	for _, file := range files {
		if target == file {
			continue
		}
		checkHash := hashVals(hashType, file)
		switch hashType {
		case hash.DCT:
			if hash.Hamming(targetHash[0], checkHash[0]) < int(hashType.Threshold) {
				found = true
				results = append(results, file)
			}
		case hash.DHASH:
			dist := hash.Hamming(targetHash[0], checkHash[0]) + hash.Hamming(targetHash[1], checkHash[1])
			if dist < int(hashType.Threshold) {
				found = true
				results = append(results, file)
			}
		}
	}
	return found, results
}
