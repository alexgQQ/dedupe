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
	duplicates := make(map[uint][]uint)

	for i, f := range files {
		img, _ := utils.LoadImage(f)
		dhash := dhash.New(img)
		item := vptree.Item{ID: uint(i), Hash: dhash}
		items = append(items, item)
	}

	tree := *vptree.New(items)
	threshold := 10.0
	for _, item := range items {
		found, _ := tree.Within(item, threshold)
		if len(found) == 0 {
			continue
		}
		var ids []uint
		for _, r := range found {
			ids = append(ids, r.ID)
		}
		duplicates[item.ID] = ids
	}

	for key, value := range duplicates {
		fmt.Printf("%s: ", files[key])
		for _, v := range value {
			fmt.Printf("%s, ", files[v])
		}
		fmt.Println("")
	}

	return nil
}
