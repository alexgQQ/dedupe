package utils

import (
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"
)

func LoadImage(file string) (image.Image, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func matchesAnyExt(path string, extensions []string) bool {
	ext := filepath.Ext(path)
	for _, e := range extensions {
		if ext == e {
			return true
		}
	}
	return false
}

func isImage(filename string) bool {
	var ext = []string{".png", ".jpg", ".jpeg"}
	return matchesAnyExt(filename, ext)
}

// It would be interesting to do this in an iterator pattern
// instead of loading a array of strings that could be potentially very large
// With more recent go version we can do some things like generator patterns
// in python
// check out the rabbit hole https://github.com/golang/go/issues/64341
func FindImages(root string, subdirs bool) []string {
	var images []string
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if isImage(d.Name()) {
			images = append(images, s)
		}
		if !subdirs && root != s && d.IsDir() {
			return fs.SkipDir
		}
		return nil
	})
	return images
}

// Bubble up any errors without breaking the loop
func MoveFiles(files []string, dir string) (e error) {
	for _, src := range files {
		filename := filepath.Base(src)
		dst := filepath.Join(dir, filename)
		err := os.Rename(src, dst)
		e = errors.Join(e, err)
	}
	return
}

// Bubble up any errors without breaking the loop
func DeleteFiles(files []string) (e error) {
	for _, f := range files {
		err := os.Remove(f)
		e = errors.Join(e, err)
	}
	return
}

func ImageOrDir(path string) (abs string, isImg bool, isDir bool) {
	if path == "" {
		return
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}

	file, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	if file.IsDir() {
		isDir = true
	} else if isImage(file.Name()) {
		isImg = true
	}
	return
}
