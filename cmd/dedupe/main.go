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
Read images from a file listing and output any duplicates found in a csv like format
	cat images.txt | dedupe -o - > duplicates.csv`
		fmt.Fprintln(flag.CommandLine.Output(), "dedupe is a program for discovering duplicate images")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s [-r | -v | -m | -d | -o | -q | -hash] <images> [<images> ...] \n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), msg)
	}

	var targets []string
	var output bool
	var quiet bool
	var recursive bool
	var verbose bool
	var search bool
	var move string
	var copy string
	var delete bool
	var deleteAll bool
	var hashName string

	flag.BoolVar(&output, "output", false, "Suppress info output and only output results. Intended to be used for piping output to a file or process")
	flag.BoolVar(&output, "o", false, "alias for -output")

	flag.BoolVar(&quiet, "quiet", false, "Suppress all output")
	flag.BoolVar(&quiet, "q", false, "alias for -quiet")

	flag.BoolVar(&recursive, "recursive", false, "Search for images in subdirectories of any target directories")
	flag.BoolVar(&recursive, "r", false, "alias for -recursive")

	flag.BoolVar(&verbose, "verbose", false, "Run application with info logging")
	flag.BoolVar(&verbose, "v", false, "alias for -verbose")

	flag.StringVar(&move, "move", "", "Move duplicate images to a the provided directory. The provided path will be created if it doesn't exist")
	flag.StringVar(&move, "m", "", "alias for -move")
	flag.StringVar(&copy, "copy", "", "Same as move but will copy files instead")
	flag.StringVar(&copy, "c", "", "alias for -copy")

	flag.BoolVar(&delete, "delete", false, "Delete all secondary instances of duplicates found")
	flag.BoolVar(&delete, "d", false, "alias for -delete")
	flag.BoolVar(&deleteAll, "delete-all", false, "Delete all instances of duplicate images found")

	flag.BoolVar(&search, "search", false, "Force a search for any duplicates against the images provided")

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
			images := utils.FindImages(target, recursive)
			files = append(files, images...)
		}
	}

	var duplicates [][]string
	var total int
	if len(files) <= 0 {
		return errors.New("no image file were found")
	} else if imgTarget && !search {
		isDupe, results := dedupe.Compare(hashType, files[0], files[1:]...)
		if isDupe {
			duplicates = append(duplicates, results)
			total = len(results)
		}
	} else {
		duplicates, total, _ = dedupe.Duplicates(files, hashType)
	}

	defaultWriter := os.Stdout
	if output || quiet {
		// io.Discard seems to be of the proper type but does not compile
		// so I'm doing this instead
		defaultWriter, _ = os.Open(os.DevNull)
		defer defaultWriter.Close()
	}
	if total == 0 {
		fmt.Fprintln(defaultWriter, "No duplicate images found")
		// I think it makes sense to return an error so a return code can be
		// sent specifically for no duplicates case
		return nil
	}
	if imgTarget && !search {
		fmt.Fprintf(defaultWriter, "These %d images are duplicates of %s\n", total, files[0])
	} else {
		fmt.Fprintf(defaultWriter, "These %d images are duplicates\n", total)
	}

	var w *csv.Writer
	if quiet {
		w = csv.NewWriter(defaultWriter)
	} else {
		w = csv.NewWriter(os.Stdout)
	}
	for _, group := range duplicates {
		if err := w.Write(group); err != nil {
			slog.Error("Error writing record to csv", "error", err)
		}
	}
	w.Flush()

	if move != "" {
		for i, files := range duplicates {
			parent := filepath.Join(move, fmt.Sprintf("group%d", i))
			os.MkdirAll(parent, 0750)
			if err := utils.MoveFiles(files, parent); err != nil {
				slog.Error("Error moving files", "error", err)
			}
		}
	} else if copy != "" {
		for i, files := range duplicates {
			parent := filepath.Join(copy, fmt.Sprintf("group%d", i))
			os.MkdirAll(parent, 0750)
			if err := utils.CopyFiles(files, parent); err != nil {
				slog.Error("Error copying files", "error", err)
			}
		}
	} else if delete {
		for _, files := range duplicates {
			if !deleteAll {
				files = files[1:]
			}
			if err := utils.DeleteFiles(files); err != nil {
				slog.Error("Error deleting files", "error", err)
			}
		}
	}
	return nil
}
