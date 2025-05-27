package hash

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// These tests are specific in validating the Hamming function can fulfill the
// requirements for being used as a metric in a vantage point tree

func TestZeroHamming(t *testing.T) {
	var a, b uint64
	a = 10
	b = 10
	if Hamming(a, b) != 0 {
		t.Error("The hamming distance between the same hash should be zero")
	}
}

func TestEqualHamming(t *testing.T) {
	var a, b uint64
	a = 0
	b = 15
	if Hamming(a, b) != Hamming(b, a) {
		t.Error("The hamming distance between two hashes should always be the same")
	}
}

func TestTriangleHamming(t *testing.T) {
	// Tests the triangle inequality as in for a triangle
	// no one side can be larger than the sum of the other two
	var a, b, c uint64
	a = 0
	b = 15
	c = 30
	if Hamming(a, c) > Hamming(a, b)+Hamming(b, c) {
		t.Error("The triangle inequality failed")
	}
}

func TestKnownHamming(t *testing.T) {
	var a, b uint64
	a = 0x0
	b = 0xf
	if Hamming(a, b) != 4 {
		t.Error("The hamming distance between 0x0 and 0xf should be 4")
	}
}

func TestMaxHamming(t *testing.T) {
	var a, b uint64
	a = 0
	b = math.MaxUint64
	if Hamming(a, b) != 64 {
		t.Error("The maximum possible hamming distance should be 64")
	}
}

func TestZeroDhash(t *testing.T) {
	width := 100
	height := 100
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	rowHash, colHash := Dhash(img)
	if rowHash != 0 {
		t.Error("The row hash for a uniform white image should be zero")
	}
	if colHash != 0 {
		t.Error("The column hash for a uniform white image should be zero")
	}
}
