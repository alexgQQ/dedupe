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

func FindImages(root string) []string {
	// This will do a recursive search through subdirectories
	var a []string
	var ext = []string{".png", ".jpg", ".jpeg"}
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if matchesAnyExt(d.Name(), ext) {
			a = append(a, s)
		}
		return nil
	})
	return a
}
