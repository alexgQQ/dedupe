package dhash

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestMinHamming(t *testing.T) {
	hash := &DHash{0, 0}
	result := Hamming(hash, hash)
	if result != 0 {
		t.Error("The distance between the same hash should be zero")
	}
}

func TestKnownHamming(t *testing.T) {
	// a 0000 against 1111 case
	zerohash := &DHash{0, 0}
	hash := &DHash{0, 15}
	result := Hamming(zerohash, hash)
	if result != 4 {
		t.Error("The distance between 0x0 and 0xf should be 4")
	}
}

func TestMaxHamming(t *testing.T) {
	zerohash := &DHash{0, 0}
	maxhash := &DHash{math.MaxUint64, math.MaxUint64}
	result := Hamming(zerohash, maxhash)
	if result != 128 {
		t.Error("The distance between the lowest and highest hash should be 128")
	}
}

func TestNewDHash(t *testing.T) {
	width := 100
	height := 100
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	hash := New(img)
	if hash.row != 0 {
		t.Error("The row hash for a uniform white image should be zero")
	}
	if hash.column != 0 {
		t.Error("The column hash for a uniform white image should be zero")
	}
}
