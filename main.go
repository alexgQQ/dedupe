package main

import (
	"dedupe/dhash"
	"dedupe/utils"
	"dedupe/vptree"
	"flag"
	"fmt"
	"os"
)

func main() {
	name := flag.String("name", "World", "Name to greet")
	verbose := flag.Bool("verbose", false, "Enable verbose output")

	flag.Parse()

	var items []vptree.Item
	var target vptree.Item
	files := utils.FindImages("images")
	for _, f := range files {
		img, _ := utils.LoadImage(f)
		dhash := dhash.New(img)
		item := vptree.Item{Path: f, Hash: dhash}
		items = append(items, item)
		if f == "images/kitten.jpg" {
			target = item
		}
	}

	tree := *vptree.New(items)
	results, dist := tree.Search(target, 7)

	for ix, r := range results {
		fmt.Printf("%s - %f\n", r.Path, dist[ix])
	}

	if err := run(*name, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func run(name string, verbose bool) error {
	if verbose {
		fmt.Println("Running in verbose mode")
	}
	fmt.Printf("Hello, %s!\n", name)
	return nil
}
