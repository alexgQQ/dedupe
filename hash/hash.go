package hash

import (
	"dedupe/utils"
	"image"
	"math"
	"math/bits"
	"sort"
)

// The method for getting a dhash is outlined here https://www.hackerfactor.com/blog/index.php?/archives/529-Kind-of-Like-That.html

// For image resizing we really don't need a high quality process and can likely ignore samplers that optimize for upscaling
// the Linear and Box filter work fine but using the NearestNeighbor sacrifices some accuracy in our test case

func Dhash(img image.Image) (uint64, uint64) {
	size := 9
	img = utils.Resize(img, 9, 9, utils.Linear)

	var segments [9][9]float64
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			r, g, b, _ := img.At(i, j).RGBA()
			// Grayscale by averaging, I wonder if doing it by weights/luminosity has any meaningful impact
			segments[i][j] = float64((r + g + b) / 3)
		}
	}

	var row, col uint64

	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if segments[x][y] < segments[x+1][y] {
				row |= 1 << ((y * 8) + x)
			}
			if segments[x][y] < segments[x][y+1] {
				col |= 1 << ((y * 8) + x)
			}
		}
	}
	return row, col
}

// This is ported from https://github.com/azr/phash/blob/main/dtc.go
// I initially used it as a quick test so it may be worth refactoring
// but it works for now and that's enough

// DTC computes the perceptual hash for img using phash dtc image
// technique.
//
//  1. Reduce size to 32x32
//  2. Reduce color to greyscale
//  3. Compute the DCT.
//  4. Reduce the DCT to 8x8 in order to keep high frequencies.
//  5. Compute the median value of 8x8 dtc.
//  6. Further reduce the DCT into an uint64.
func DCT(img image.Image) uint64 {
	// if img == nil {
	// 	return nil
	// }

	var (
		dtcSizeBig = 32
		dtcSize    = 8
	)

	size := dtcSizeBig
	smallerSize := dtcSize

	// 1. Reduce size.
	// Like Average Hash, pHash starts with a small image. However,
	// the image is larger than 8x8; 32x32 is a good size. This is
	// really done to simplify the DCT computation and not because it
	// is needed to reduce the high frequencies.
	im := utils.Resize(img, size, size, utils.Linear)

	// 2. Reduce color.
	// The image is reduced to a grayscale just to further simplify
	// the number of computations.

	vals := make([]float64, size*size)
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			// vals[size*i+j] = colorToGreyScaleFloat64(im.At(i, j))
			r, g, b, _ := im.At(i, j).RGBA()
			vals[size*i+j] = float64((r + g + b) / 3)
		}
	}

	// 3. Compute the DCT.
	// The DCT separates the image into a collection of frequencies
	// and scalars. While JPEG uses an 8x8 DCT, this algorithm uses a
	// 32x32 DCT.

	applyDCT2 := func(N int, f []float64) []float64 {
		// initialize coefficients
		c := make([]float64, N)
		c[0] = 1 / math.Sqrt(2)
		for i := 1; i < N; i++ {
			c[i] = 1
		}

		// output goes here
		F := make([]float64, N*N)

		// construct a lookup table, because it's O(n^4)
		entries := (2 * N) * (N - 1)
		COS := make([]float64, entries)
		for i := 0; i < entries; i++ {
			COS[i] = math.Cos(float64(i) / float64(2*N) * math.Pi)
		}

		// the core loop inside a loop inside a loop...
		for u := 0; u < N; u++ {
			for v := 0; v < N; v++ {
				var sum float64
				for i := 0; i < N; i++ {
					for j := 0; j < N; j++ {
						sum += COS[(2*i+1)*u] *
							COS[(2*j+1)*v] *
							f[N*i+j]
					}
				}
				sum *= ((c[u] * c[v]) / 4)
				F[N*u+v] = sum
			}
		}
		return F
	}

	dctVals := applyDCT2(size, vals)

	// 4. Reduce the DCT.
	// While the DCT is 32x32, just keep the top-left 8x8. Those
	// represent the lowest frequencies in the picture.

	vals = make([]float64, 0, smallerSize*smallerSize)
	for x := 1; x <= smallerSize; x++ {
		for y := 1; y <= smallerSize; y++ {
			vals = append(vals, dctVals[size*x+y])
		}
	}

	// 5. Compute the median value.
	// Like the Average Hash, compute the mean DCT value (using only
	// the 8x8 DCT low-frequency values and excluding the first term
	// since the DC coefficient can be significantly different from
	// the other values and will throw off the average).

	sortedVals := make([]float64, smallerSize*smallerSize)
	copy(sortedVals, vals)
	sort.Float64s(sortedVals)
	median := sortedVals[smallerSize*smallerSize/2]

	// 6. Further reduce the DCT.
	// Set the 64 hash bits to 0 or 1 depending on whether each of the
	// 64 DCT values is above or below the average value. The result
	// doesn't tell us the actual low frequencies; it just tells us
	// the very-rough relative scale of the frequencies to the mean.
	// The result will not vary as long as the overall structure of
	// the image remains the same; this will survive gamma and color
	// histogram adjustments without a problem.

	var phash uint64
	for n, e := range vals {
		if e > median { // when frequency is higher than median
			phash ^= (1 << uint64(n)) // set nth bit to one
		}
	}
	return phash
}

func Hamming(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}
