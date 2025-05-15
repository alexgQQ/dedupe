package main

import (
	"dedupe/dhash"
	"dedupe/utils"
	"dedupe/vptree"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
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
	// dir := flag.String("target", "wallpapers", "target image directory")
	dir := flag.String("target", "images", "target image directory")
	recursive := flag.Bool("recursive", true, "")
	output := flag.String("output", "stdout", "")
	delete := flag.Bool("delete", false, "")
	move := flag.Bool("move", false, "")
	verbose := flag.Bool("verbose", false, "")

	flag.Parse()

	var logLevel = new(slog.LevelVar)
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(h))
	if *verbose {
		logLevel.Set(slog.LevelInfo)
	} else {
		logLevel.Set(slog.LevelWarn)
	}

	if err := validatePath(*dir); err != nil {
		log.Fatal(err)
	}
	files := utils.FindImages(*dir, *recursive)
	if len(files) < 1 {
		log.Fatal("no images found")
	}

	duplicates, err := findDuplicates(files)
	if err != nil {
		log.Fatal(err)
	}

	var w *csv.Writer
	if *output == "stdout" {
		w = csv.NewWriter(os.Stdout)
	} else {
		path, _ := filepath.Abs(*output)
		file, err := os.OpenFile(path, os.O_RDWR, 0666)
		if errors.Is(err, os.ErrNotExist) {
			file, _ = os.Create(path)
		}
		w = csv.NewWriter(file)
	}

	for _, group := range duplicates {
		if err := w.Write(group); err != nil {
			log.Fatalln("error writing record to csv:", err)
		}
	}
	w.Flush()

	if *move {
		for _, files := range duplicates {
			moveFiles(files, *dir)
		}
	} else if *delete {
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

	nWorkers := 2
	itemID := 1
	// work := make(chan image.Image)
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
					slog.Error("error loading image", "file", f, "error", err)
				} else {
					dhash := dhash.New(img)
					slog.Info("computed image hash", "file", f, "hash", dhash)
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

	slog.Info("gathering image hashes", "imageCount", len(files))
	for item := range results {
		items = append(items, item)
	}
	return vptree.New(items)
}

func findDuplicates(files []string) ([][]string, error) {
	tree := buildTree(files)
	duplicates := make(map[uint][]uint)
	threshold := 22.0

	for _, item := range tree.Items() {
		found, dist := tree.Within(item, threshold)
		if len(found) == 0 && len(dist) == 0 {
			continue
		}
		var ids []uint
		for _, r := range found {
			ids = append(ids, r.ID)
		}
		duplicates[item.ID] = ids
	}

	var skip []uint
	var filegroups [][]string
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
		for _, g := range group {
			filenames = append(filenames, files[g])
		}
		filegroups = append(filegroups, filenames)
	}

	return filegroups, nil
}
