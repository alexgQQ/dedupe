package utils

import (
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

func FindImages(root string, subdirs bool) []string {
	var images []string
	var ext = []string{".png", ".jpg", ".jpeg"}
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if matchesAnyExt(d.Name(), ext) {
			images = append(images, s)
		}
		if !subdirs && root != s && d.IsDir() {
			return fs.SkipDir
		}
		return nil
	})
	return images
}
