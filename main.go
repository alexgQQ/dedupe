package main

import (
	"dedupe/dhash"
	"dedupe/utils"
	"dedupe/vptree"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	dir := flag.String("target", "images", "target image directory")

	flag.Parse()

	if err := validatePath(*dir); err != nil {
		log.Fatal(err)
	}
	files := utils.FindImages(*dir)
	if len(files) < 1 {
		log.Fatal("no images found")
	}

	if err := run(files); err != nil {
		log.Fatal(err)
	}
}

func run(files []string) error {
	var items []vptree.Item
	duplicates := make([][]int, len(files))

	for i, f := range files {
		img, _ := utils.LoadImage(f)
		dhash := dhash.New(img)
		item := vptree.Item{ID: uint(i), Hash: dhash}
		items = append(items, item)
	}

	tree := *vptree.New(items)
	threshold := 10.0

	for i, item := range items {
		var dupes []int
		// searching only works by neighbor and not by distance alone at the moment
		results, distances := tree.Search(item, len(files))

		for j, result := range results {
			// results are ordered by distance so we can break
			if distances[j] >= threshold {
				break
			}
			dupes = append(dupes, int(result.ID))
		}
		duplicates[i] = dupes
	}

	for i, dupes := range duplicates {
		if len(dupes) < 1 {
			continue
		}
		fmt.Printf("%s -\n", files[i])
		for _, d := range dupes {
			fmt.Printf("%s, ", files[d])
		}
		fmt.Print("\n")
	}

	return nil
}
