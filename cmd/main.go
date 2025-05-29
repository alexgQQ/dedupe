package main

import (
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
