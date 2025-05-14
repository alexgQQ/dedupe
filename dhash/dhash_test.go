package dhash

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// These tests are specific in validating the Hamming function can fulfill the
// requirements for being used as a metric in a vantage point tree

func TestZeroHamming(t *testing.T) {
	hash := &DHash{10, 10}
	samehash := &DHash{10, 10}
	result := Hamming(hash, samehash)
	if result != 0 {
		t.Error("The hamming distance between the same hash should be zero")
	}
}

func TestEqualHamming(t *testing.T) {
	a := &DHash{0, 0}
	b := &DHash{0, 15}
	if Hamming(a, b) != Hamming(b, a) {
		t.Error("The hamming distance between two hashes should always be the same")
	}
}

func TestTriangleHamming(t *testing.T) {
	// Tests the triangle inequality as in for a triangle
	// no one side can be larger than the sum of the other two
	a := &DHash{0, 0}
	b := &DHash{0, 15}
	c := &DHash{15, 15}
	if Hamming(a, c) > Hamming(a, b)+Hamming(b, c) {
		t.Error("The triangle inequality failed")
	}
}

func TestKnownHamming(t *testing.T) {
	zerohash := &DHash{0x0, 0x0}
	fifteenhash := &DHash{0x0, 0xf}
	if Hamming(zerohash, fifteenhash) != 4 {
		t.Error("The hamming distance between 0x0 and 0xf should be 4")
	}
}

func TestMaxHamming(t *testing.T) {
	zerohash := &DHash{0, 0}
	maxhash := &DHash{math.MaxUint64, math.MaxUint64}
	if Hamming(zerohash, maxhash) != 128 {
		t.Error("The maximum possible hamming distance should be 128")
	}
}

func TestZeroDHash(t *testing.T) {
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
