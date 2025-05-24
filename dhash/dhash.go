package dhash

import (
	"fmt"
	"image"
	"math/bits"

	"github.com/kovidgoyal/imaging"
)

// The method for getting a dhash is outlined here https://www.hackerfactor.com/blog/index.php?/archives/529-Kind-of-Like-That.html

type DHash struct {
	row    uint64
	column uint64
}

func (d *DHash) String() string {
	return fmt.Sprintf("0x%x%x", d.row, d.column)
}

func New(img image.Image) *DHash {
	// bounds := img.Bounds()
	// width := bounds.Max.X - bounds.Min.X
	// height := bounds.Max.Y - bounds.Min.Y

	// My cruddy implementation is definitely slower than those offered by third parties
	// however I do not want to use a fork of a popular imaging library
	// so I'll want to port that code over
	size := 9
	img = imaging.Resize(img, 9, 9, imaging.Lanczos)

	// vals := make([]float64, size*size)
	var segments [9][9]float64
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			r, g, b, _ := img.At(i, j).RGBA()
			segments[i][j] = float64((r + g + b) / 3)
		}
	}

	// No explicit resizing but instead we'll split the image into
	// a grid of 9x9 segments to gather their pixel values
	// var segments [9][9]uint
	// for i := 0; i < 9; i++ {
	// 	left_bound := bounds.Min.X + (width * i / 9)
	// 	right_bound := bounds.Min.X + (width * (i + 1) / 9)

	// 	for j := 0; j < 9; j++ {
	// 		top_bound := bounds.Min.Y + (height * j / 9)
	// 		bottom_bound := bounds.Min.Y + (height * (j + 1) / 9)

	// 		var total uint64
	// 		for x := left_bound; x < right_bound; x++ {
	// 			for y := top_bound; y < bottom_bound; y++ {
	// 				r, g, b, _ := img.At(x, y).RGBA()
	// 				// Grayscale by averaging, I wonder if doing it by weights/luminosity has any meaningful impact
	// 				total += uint64((r + g + b) / 3)
	// 			}
	// 		}
	// 		// Average across the number of pixels in the grid segment
	// 		segments[i][j] = uint(total / uint64((right_bound-left_bound)*(bottom_bound-top_bound)))
	// 	}
	// }

	var dhash *DHash = &DHash{}

	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if segments[x][y] < segments[x+1][y] {
				dhash.row |= 1 << ((y * 8) + x)
			}
			if segments[x][y] < segments[x][y+1] {
				dhash.column |= 1 << ((y * 8) + x)
			}
		}
	}

	return dhash

}

func NewFromValues(row uint64, col uint64) *DHash {
	return &DHash{row, col}
}

func Hamming(dhash1 *DHash, dhash2 *DHash) int {
	return bits.OnesCount64(dhash1.row^dhash2.row) + bits.OnesCount64(dhash1.column^dhash2.column)
}
