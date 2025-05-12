package utils

import (
	"image"
)

type Hash uint64

// The method for getting a dhash is outlined here https://www.hackerfactor.com/blog/index.php?/archives/529-Kind-of-Like-That.html

func ImageHash(img image.Image) Hash {
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	// No explicit resizing but instead we'll split the image into
	// a grid of 9x9 segments to gather their pixel values
	var seg [9][8]int
	for i := 0; i < 9; i++ {
		left := bounds.Min.X + (width * i / 9)
		right := bounds.Min.X + (width * (i + 1) / 9)
		if right == left {
			right = left + 1
		}
		for j := 0; j < 8; j++ {
			top := bounds.Min.Y + (height * j / 8)
			bottom := bounds.Min.Y + (height * (j + 1) / 8)
			if bottom == top {
				bottom = top + 1
			}
			var total int64
			for x := left; x < right; x++ {
				for y := top; y < bottom; y++ {
					r, g, b, _ := img.At(x, y).RGBA()
					// Greyscale by averaging, I wonder if doing it by weights/luminosity has any meaningful impact
					total += int64((r + g + b) / 3)
				}
			}
			// Average across the number of pixels in the grid segment
			seg[i][j] = int(total / int64((right-left)*(bottom-top)))
		}
	}

	var bits [64]bool
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if seg[x][y] < seg[x+1][y] {
				bits[(y*8)+x] = true
			} else {
				bits[(y*8)+x] = false
			}
		}
	}

	var result Hash
	for i, bit := range bits {
		if bit {
			result |= 1 << i
		}
	}

	// test value: 3a6c6565498da525
	return result
}
