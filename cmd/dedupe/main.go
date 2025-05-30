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
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {

	flag.Usage = func() {
		msg := `
Example usage:
Compare two images
	dedupe duplicate/image.jpg duplicate/image-copy.jpg
Find duplicates of target/image.jpg in path/to/images
	dedupe target/image.jpg path/to/images
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
		return errors.New("no arguments provided")
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

	var files []string
	noDirs := true
	imgTarget := false
	for i, target := range targets {
		_, isImg, isDir := utils.ImageOrDir(target)
		if !isImg && !isDir {
			continue
		} else if isImg {
			if i == 0 {
				imgTarget = true
			}
			files = append(files, target)
		} else if isDir {
			noDirs = false
			images := utils.FindImages(target, recursive)
			files = append(files, images...)
		}
	}

	var duplicates [][]string
	var total int
	if len(files) <= 0 {
		return errors.New("no image file were found")
	} else if noDirs && len(files) == 2 {
		isDupe, results := dedupe.Compare(hashType, files[0], files[1])
		if isDupe {
			results = append(results, files[0])
			duplicates = append(duplicates, results)
			total = 2
		}
	} else if imgTarget {
		isDupe, results := dedupe.Compare(hashType, files[0], files[1:]...)
		if isDupe {
			duplicates = append(duplicates, results)
			total = len(results)
		}
	} else {
		duplicates, total, _ = dedupe.Duplicates(files, hashType)
	}

	if total == 0 {
		fmt.Println("No duplicate images found")
		return nil
	}

	fmt.Printf("Found %d duplicate images\n", total)
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
		dupeDir := "duplicates"
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
	return nil
}
