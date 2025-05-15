package main

import (
	"dedupe/dhash"
	"dedupe/utils"
	"dedupe/vptree"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
)

func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	fileInfo, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %w", err)
		}
		return fmt.Errorf("error checking path: %w", err)
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("path is not a directory")
	}

	return nil
}

func main() {
	var target string
	var output string
	var recursive bool
	var verbose bool
	var move bool
	var delete bool

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

	flag.Parse()

	var logLevel = new(slog.LevelVar)
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(h))
	if verbose {
		logLevel.Set(slog.LevelInfo)
	} else {
		logLevel.Set(slog.LevelWarn)
	}

	if target == "" {
		slog.Error("No target directory provided")
		os.Exit(1)
	} else if err := validatePath(target); err != nil {
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
	duplicates, total, err := findDuplicates(files)
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
		for _, files := range duplicates {
			moveFiles(files, target)
		}
	} else if delete {
		for _, files := range duplicates {
			deleteFiles(files)
		}
	}
}

func moveFiles(files []string, dir string) error {
	dupeDir := filepath.Join(dir, "duplicates")
	os.Mkdir(dupeDir, 0750)
	for _, src := range files {
		filename := filepath.Base(src)
		dst := filepath.Join(dir, "duplicates", filename)
		os.Rename(src, dst)
	}
	return nil
}

func deleteFiles(files []string) error {
	for _, src := range files {
		os.Remove(src)
	}
	return nil
}

func buildTree(files []string) *vptree.VPTree {
	var items []vptree.Item
	var wg sync.WaitGroup

	// Lets start with allocating available cpu as the worker count
	// it's hard to say what could be optimal outside of that
	nWorkers := runtime.NumCPU()
	itemID := 0
	work := make(chan string)
	results := make(chan vptree.Item)

	// Spin up nWorkers to process images concurrently
	for i := 0; i < nWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range work {
				img, err := utils.LoadImage(f)
				if err != nil {
					slog.Error("Error loading image", "file", f, "error", err)
				} else {
					dhash := dhash.New(img)
					slog.Info("Computed image hash", "file", f, "hash", dhash)
					item := vptree.Item{ID: uint(itemID), Hash: dhash}
					itemID++
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

	slog.Info("Gathering image hashes", "imageCount", len(files), "workerCount", nWorkers)
	for item := range results {
		items = append(items, item)
	}
	return vptree.New(items)
}

func findDuplicates(files []string) ([][]string, int, error) {
	tree := buildTree(files)
	duplicates := make(map[uint][]uint)
	// This is a bit of an arbitrary number, most duplicates will have a very low distance metric but let's cast a wide net
	threshold := 10.0

	for _, item := range tree.Items() {
		found, dist := tree.Within(item, threshold)
		if len(found) == 0 && len(dist) == 0 {
			continue
		}
		slog.Info("VPTree found results within an item", "item", item, "found", found, "dist", dist, "threshold", threshold)
		var ids []uint
		for _, r := range found {
			ids = append(ids, r.ID)
		}
		duplicates[item.ID] = ids
	}

	var skip []uint
	var filegroups [][]string
	total := 0
	// Make unique arrays of the found duplicates
	for key, value := range duplicates {
		if slices.Contains(skip, key) {
			continue
		}
		var filenames []string
		var group []uint = value
		for _, v := range value {
			group = append(group, duplicates[v]...)
			skip = append(skip, v)
		}
		slices.Sort(group)
		group = slices.Compact(group)
		total += len(group)
		for _, g := range group {
			filenames = append(filenames, files[g])
		}
		filegroups = append(filegroups, filenames)
	}
	slog.Info("Duplicate images found", "count", total)

	return filegroups, total, nil
}
