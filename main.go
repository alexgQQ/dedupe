package main

import (
	"dedupe/hash"
	"dedupe/utils"
	"dedupe/vptree"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
)

func main() {
	var target string
	var output string
	var recursive bool
	var verbose bool
	var move bool
	var delete bool
	var hashName string

	flag.StringVar(&target, "target", "", "The directory to search for image duplicates")
	flag.StringVar(&target, "t", "", "alias for -target")

	flag.StringVar(&output, "output", "stdout", "Set to a file for a csv output, otherwise it goes to stdout")
	flag.StringVar(&output, "o", "stdout", "alias for -output")

	flag.BoolVar(&recursive, "recursive", false, "Search for images in subdirectories of the target directory")
	flag.BoolVar(&recursive, "r", false, "alias for -recursive")

	flag.BoolVar(&verbose, "verbose", false, "Run application with info logging")
	flag.BoolVar(&verbose, "v", false, "alias for -verbose")

	flag.BoolVar(&move, "move", false, "Move duplicate images to a `duplicates` directory under the target directory")
	flag.BoolVar(&move, "m", false, "alias for -move")

	flag.BoolVar(&delete, "delete", false, "Delete duplicate images")
	flag.BoolVar(&delete, "d", false, "alias for -delete")

	hashTypes := slices.Sorted(maps.Keys(hash.HashTypes))
	opts := strings.Join(hashTypes, ", ")
	flag.StringVar(&hashName, "hash", "dct", fmt.Sprintf("Which type of hash to use for searching. Available options are %s", opts))

	flag.Parse()

	var logLevel = new(slog.LevelVar)
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(h))
	if verbose {
		logLevel.Set(slog.LevelInfo)
	} else {
		logLevel.Set(slog.LevelWarn)
	}

	var hashType hash.HashType
	if !slices.Contains(hashTypes, hashName) {
		slog.Error("Invalid hash type provided", "hashName", hashName)
		hashType = hash.DCT
	} else {
		hashType = hash.HashTypes[hashName]
	}

	if err := utils.PathIsDir(target); err != nil {
		slog.Error("Invalid target path", "path", target, "error", err)
		os.Exit(1)
	}
	files := utils.FindImages(target, recursive)
	if len(files) < 1 {
		slog.Error("No images found at target path", "path", target)
		os.Exit(1)
	}
	target, _ = filepath.Abs(target)

	fmt.Printf("Scanning %s for duplicate images...\n", target)
	duplicates, total, err := Duplicates(files, hashType)

	if err != nil {
		slog.Error("Error occurred while finding duplicates", "error", err)
		os.Exit(1)
	}
	if total == 0 {
		fmt.Print("No duplicate images found")
		os.Exit(0)
	}
	fmt.Printf("%d duplicate images found:\n", total)
	var w *csv.Writer
	if output == "stdout" {
		w = csv.NewWriter(os.Stdout)
	} else {
		path, _ := filepath.Abs(output)
		file, err := os.OpenFile(path, os.O_RDWR, 0666)
		if errors.Is(err, os.ErrNotExist) {
			file, _ = os.Create(path)
		}
		w = csv.NewWriter(file)
	}

	for _, group := range duplicates {
		if err := w.Write(group); err != nil {
			slog.Error("Error writing record to csv", "error", err)
		}
	}
	w.Flush()

	if move {
		dupeDir := filepath.Join(target, "duplicates")
		os.Mkdir(dupeDir, 0750)
		for _, files := range duplicates {
			if err := utils.MoveFiles(files, dupeDir); err != nil {
				slog.Error("Error moving files", "error", err)
			}
		}
	} else if delete {
		for _, files := range duplicates {
			if err := utils.DeleteFiles(files); err != nil {
				slog.Error("Error deleting files", "error", err)
			}
		}
	}
}

func buildTree(files []string, hashType hash.HashType) (*vptree.VPTree, *vptree.FileMapper) {
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
					slog.Error("Error loading image", "file", f, "error", err)
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
		found, dist := tree.Within(item, threshold)
		if len(found) <= 0 {
			continue
		}
		slog.Info("VPTree found results within item", "item", item, "results", found, "distances", dist, "threshold", threshold)
		group = append(group, item.ID)

		// I've gone back and forth with myself on this
		// do I need to do this reciprocal search?
		// Basically anything subsequently found here would be at max
		// 2x the threshold from the first item, which isn't necessarily wrong
		for _, i := range found {
			group = append(group, i.ID)
			f, d := tree.Within(i, threshold)
			for _, F := range f {
				group = append(group, F.ID)
			}
			slog.Info("VPTree found results within item", "item", item, "results", f, "distances", d, "threshold", threshold)
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
