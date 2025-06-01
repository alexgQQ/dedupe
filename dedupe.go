package dedupe

import (
	"errors"
	"fmt"
	"image"
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
	errs := make(chan error, nWorkers)

	// Spin up nWorkers to process images concurrently
	for range nWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range work {
				img, err := utils.LoadImage(f)
				if err != nil {
					errs <- fmt.Errorf("unable to load %s %w", f, err)
					continue
				}
				var item *vptree.Item
				hashes := imageHash(hashType, img)
				item = vptree.NewItem(f, &fileMap, hashes...)
				results <- item
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
		close(errs)
		close(results)
	}()

	// Accumulate the computed hashes to build the vptree
	var items []*vptree.Item
	for item := range results {
		items = append(items, item)
	}
	// Gather any errors during processing
	var err error
	for e := range errs {
		err = errors.Join(err, e)
	}
	return vptree.New(items), &fileMap, err
}

// Find groups of duplicate images from a list of given images
// hashTypes determines the hashing method and can be either dedupe.DCT or dedupe.DHASH
func Duplicates(hashType hash.HashType, files []string) (duplicates [][]string, total int, err error) {
	var skip []uint
	tree, fileMap, err := buildTree(files, hashType)
	for item := range tree.All() {
		if slices.Contains(skip, item.ID) {
			continue
		}
		found, _ := tree.Within(item, hashType.Threshold)
		if len(found) <= 0 {
			continue
		}
		group := make([]string, len(found)+1)
		skip = append(skip, item.ID)
		group[0] = fileMap.ByID(item.ID)
		for i, item := range found {
			group[i+1] = fileMap.ByID(item.ID)
			skip = append(skip, item.ID)
		}
		total += len(group)
		duplicates = append(duplicates, group)
	}
	return
}

// Find any duplicate images of the target image from given image files
// hashTypes determines the hashing method and can be either dedupe.DCT or dedupe.DHASH
func Compare(hashType hash.HashType, target string, files ...string) (filenames []string, err error) {
	// It should be noted that for a few amount of files building the tree might be overkill
	// but I'd rather have it consistent
	img, err := utils.LoadImage(target)
	if err != nil {
		return
	}
	tree, fileMap, err := buildTree(files, hashType)
	hashes := imageHash(hashType, img)
	item := vptree.NewItem(target, fileMap, hashes...)
	results, _ := tree.Within(*item, hashType.Threshold)
	if len(results) <= 0 {
		return
	}
	filenames = make([]string, len(results))
	for i, r := range results {
		filenames[i] = fileMap.ByID(r.ID)
	}
	return
}
