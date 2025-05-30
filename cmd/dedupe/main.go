package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/alexgQQ/dedupe"
	"github.com/alexgQQ/dedupe/hash"
	"github.com/alexgQQ/dedupe/utils"
)

func main() {

	flag.Usage = func() {
		msg := `
Example usage:
Compare two images
	dedupe duplicate/image.jpg potential/duplicate/image.jpg 
Output duplicate images found in path/to/images and other/path/to/images
	dedupe path/to/images other/path/to/images
Find and delete duplicate images in path/to/images and any of it's subdirectories
	dedupe -recursive -delete path/to/images
Read images from a file listing and output any duplicates found
	cat images.txt | dedupe -`
		fmt.Fprintln(flag.CommandLine.Output(), "dedupe is a program for discovering duplicate images")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s [-r | -v | -m | -d | -o | -hash] <images> [<images> ...] \n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), msg)
	}

	var targets []string
	var output string
	var recursive bool
	var verbose bool
	var move bool
	var delete bool
	var hashName string

	flag.StringVar(&output, "output", "stdout", "Set to a file for a csv output, otherwise it goes to stdout")
	flag.StringVar(&output, "o", "stdout", "alias for -output")

	flag.BoolVar(&recursive, "recursive", false, "Search for images in subdirectories of any target directories")
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
	args := flag.Args()
	if len(args) <= 0 {
		slog.Error("No images provided")
		os.Exit(1)
	} else if slices.Contains(args, "-") {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			// This could be problematic if filepaths have spaces, a bit of an edge case
			// so I won't worry for now
			targets = append(targets, strings.Split(line, " ")...)
		}
	} else {
		targets = args
	}

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
	duplicates, total, err := dedupe.Duplicates(files, hashType)

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
