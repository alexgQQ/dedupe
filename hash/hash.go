package hash

import (
	"image"
	"image/color"
	"math"
	"math/bits"
	"sort"

	"github.com/alexgQQ/dedupe/utils"
)

type HashType struct {
	name      string
	Threshold float64
}

// Based on some of the initial documentation,
// https://www.hackerfactor.com/blog/index.php?/archives/529-Kind-of-Like-That.html
// https://phash.org/docs/design.html
// The dhash implementation should work with 10 and the dct should work with 22

var DHASH HashType = HashType{
	name:      "dhash",
	Threshold: 10.0,
}

var DCT HashType = HashType{
	name:      "dct",
	Threshold: 22.0,
}

var HashTypes = map[string]HashType{
	DHASH.name: DHASH,
	DCT.name:   DCT,
}

// Convert to greyscale with the luminosity approximation
func colorToGrey(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
}

// The method for getting a dhash is outlined here https://www.hackerfactor.com/blog/index.php?/archives/529-Kind-of-Like-That.html
func Dhash(img image.Image) (row, col uint64) {
	// For image resizing we really don't need a high quality process and can likely ignore samplers that optimize for upscaling
	// the Linear and Box filter work fine but using the NearestNeighbor sacrifices some accuracy in our test case
	// TODO: there should be a way to account for images that are smaller than the target size
	size := 9
	img = utils.Resize(img, size, size, utils.Linear)

	grey := make([]float64, size*size)
	for x := range size {
		for y := range size {
			grey[size*x+y] = colorToGrey(img.At(x, y))
		}
	}

	for y := range size - 1 {
		for x := range size - 1 {
			if grey[size*x+y] < grey[size*(x+1)+y] {
				row |= 1 << ((y * 8) + x)
			}
			if grey[size*x+y] < grey[size*x+(y+1)] {
				col |= 1 << ((y * 8) + x)
			}
		}
	}
	return
}

// This is ported from https://github.com/azr/phash/blob/main/dtc.go
func Dct(img image.Image) (phash uint64) {
	// For image resizing we really don't need a high quality process and can likely ignore samplers that optimize for upscaling
	// the Linear and Box filter work fine but using the NearestNeighbor sacrifices some accuracy in our test case
	// TODO: there should be a way to account for images that are smaller than the target size
	size := 32
	im := utils.Resize(img, size, size, utils.Linear)

	// Convert the image to grayscale
	gray := make([]float64, size*size)
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			gray[size*i+j] = colorToGrey(im.At(i, j))
		}
	}

	applyDCT2 := func(N int, f []float64) []float64 {
		// I'm not entirely sure why these coefficient values are these specifically but they work
		c := make([]float64, N)
		c[0] = 1 / math.Sqrt(2)
		for i := 1; i < N; i++ {
			c[i] = 1
		}

		// Compute the dct2 as the sum of the cosine products
		// A good reference can be seen here https://www.mathworks.com/help/images/discrete-cosine-transform.html
		entries := (2 * N) * (N - 1)
		COS := make([]float64, entries)
		for i := range entries {
			COS[i] = math.Cos(float64(i) / float64(2*N) * math.Pi)
		}

		F := make([]float64, N*N)
		for u := range N {
			for v := range N {
				var sum float64
				for i := range N {
					for j := range N {
						sum += COS[(2*i+1)*u] * COS[(2*j+1)*v] * f[N*i+j]
					}
				}
				sum *= ((c[u] * c[v]) / 4)
				F[N*u+v] = sum
			}
		}
		return F
	}

	dctVals := applyDCT2(size, gray)

	// The hashing only needs the top left 8x8 domain as these contain the lower frequencies
	regionSize := 8
	freqs := make([]float64, regionSize*regionSize)
	for x := range regionSize {
		for y := range regionSize {
			// Exclude the first term from dct as it's coefficient can throw off the average
			xN := x + 1
			yN := y + 1
			freqs[regionSize*x+y] = dctVals[size*xN+yN]
		}
	}

	// Get the median value from the low frequency band
	sorted := make([]float64, regionSize*regionSize)
	copy(sorted, freqs)
	sort.Float64s(sorted)
	median := sorted[regionSize*regionSize/2]

	// Compute the hash by a relative scale from the median that enables accurate and resilient comparison.
	for n, f := range freqs {
		if f > median {
			phash ^= (1 << uint64(n))
		}
	}
	return
}

func Hamming(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}
