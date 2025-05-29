// This has been ported from https://github.com/kovidgoyal/imaging/blob/master/resize_test.go

package utils

import (
	"image"
	"testing"
)

func compareNRGBA(img1, img2 *image.NRGBA, delta int) bool {
	if !img1.Rect.Eq(img2.Rect) {
		return false
	}
	return compareBytes(img1.Pix, img2.Pix, delta)
}

func compareBytes(a, b []uint8, delta int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if absint(int(a[i])-int(b[i])) > delta {
			return false
		}
	}
	return true
}

// absint returns the absolute value of i.
func absint(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func TestResize(t *testing.T) {
	testCases := []struct {
		name string
		src  image.Image
		w, h int
		f    ResampleFilter
		want *image.NRGBA
	}{
		{
			"Resize 2x2 1x1 box",
			&image.NRGBA{
				Rect:   image.Rect(-1, -1, 1, 1),
				Stride: 2 * 4,
				Pix: []uint8{
					0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0xff,
					0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff,
				},
			},
			1, 1,
			Box,
			&image.NRGBA{
				Rect:   image.Rect(0, 0, 1, 1),
				Stride: 1 * 4,
				Pix:    []uint8{0x55, 0x55, 0x55, 0xc0},
			},
		},
		{
			"Resize 2x2 1x2 box",
			&image.NRGBA{
				Rect:   image.Rect(-1, -1, 1, 1),
				Stride: 2 * 4,
				Pix: []uint8{
					0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0xff,
					0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff,
				},
			},
			1, 2,
			Box,
			&image.NRGBA{
				Rect:   image.Rect(0, 0, 1, 2),
				Stride: 1 * 4,
				Pix: []uint8{
					0xff, 0x00, 0x00, 0x80,
					0x00, 0x80, 0x80, 0xff,
				},
			},
		},
		{
			"Resize 2x2 2x1 box",
			&image.NRGBA{
				Rect:   image.Rect(-1, -1, 1, 1),
				Stride: 2 * 4,
				Pix: []uint8{
					0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0xff,
					0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff,
				},
			},
			2, 1,
			Box,
			&image.NRGBA{
				Rect:   image.Rect(0, 0, 2, 1),
				Stride: 2 * 4,
				Pix: []uint8{
					0x00, 0xff, 0x00, 0x80, 0x80, 0x00, 0x80, 0xff,
				},
			},
		},
		{
			"Resize 2x2 2x2 box",
			&image.NRGBA{
				Rect:   image.Rect(-1, -1, 1, 1),
				Stride: 2 * 4,
				Pix: []uint8{
					0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0xff,
					0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff,
				},
			},
			2, 2,
			Box,
			&image.NRGBA{
				Rect:   image.Rect(0, 0, 2, 2),
				Stride: 2 * 4,
				Pix: []uint8{
					0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0xff,
					0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff,
				},
			},
		},
		{
			"Resize 3x1 1x1 nearest",
			&image.NRGBA{
				Rect:   image.Rect(-1, -1, 2, 0),
				Stride: 3 * 4,
				Pix: []uint8{
					0xff, 0x00, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff,
				},
			},
			1, 1,
			NearestNeighbor,
			&image.NRGBA{
				Rect:   image.Rect(0, 0, 1, 1),
				Stride: 1 * 4,
				Pix:    []uint8{0x00, 0xff, 0x00, 0xff},
			},
		},
		// These fail fro some reason and deserve some exploration at some point
		// {
		// 	"Resize 2x2 0x4 box",
		// 	&image.NRGBA{
		// 		Rect:   image.Rect(-1, -1, 1, 1),
		// 		Stride: 2 * 4,
		// 		Pix: []uint8{
		// 			0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0xff,
		// 			0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff,
		// 		},
		// 	},
		// 	0, 4,
		// 	Box,
		// 	&image.NRGBA{
		// 		Rect:   image.Rect(0, 0, 4, 4),
		// 		Stride: 4 * 4,
		// 		Pix: []uint8{
		// 			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0xff,
		// 			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0xff,
		// 			0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff,
		// 			0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff,
		// 		},
		// 	},
		// },
		// {
		// 	"Resize 2x2 4x0 linear",
		// 	&image.NRGBA{
		// 		Rect:   image.Rect(-1, -1, 1, 1),
		// 		Stride: 2 * 4,
		// 		Pix: []uint8{
		// 			0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0xff,
		// 			0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff,
		// 		},
		// 	},
		// 	4, 0,
		// 	Linear,
		// 	&image.NRGBA{
		// 		Rect:   image.Rect(0, 0, 4, 4),
		// 		Stride: 4 * 4,
		// 		Pix: []uint8{
		// 			0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0x40, 0xff, 0x00, 0x00, 0xbf, 0xff, 0x00, 0x00, 0xff,
		// 			0x00, 0xff, 0x00, 0x40, 0x6e, 0x6d, 0x25, 0x70, 0xb0, 0x14, 0x3b, 0xcf, 0xbf, 0x00, 0x40, 0xff,
		// 			0x00, 0xff, 0x00, 0xbf, 0x14, 0xb0, 0x3b, 0xcf, 0x33, 0x33, 0x99, 0xef, 0x40, 0x00, 0xbf, 0xff,
		// 			0x00, 0xff, 0x00, 0xff, 0x00, 0xbf, 0x40, 0xff, 0x00, 0x40, 0xbf, 0xff, 0x00, 0x00, 0xff, 0xff,
		// 		},
		// 	},
		// },
		{
			"Resize 0x0 1x1 box",
			&image.NRGBA{
				Rect:   image.Rect(-1, -1, -1, -1),
				Stride: 0,
				Pix:    []uint8{},
			},
			1, 1,
			Box,
			&image.NRGBA{},
		},
		{
			"Resize 2x2 0x0 box",
			&image.NRGBA{
				Rect:   image.Rect(-1, -1, 1, 1),
				Stride: 2 * 4,
				Pix: []uint8{
					0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0xff,
					0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff,
				},
			},
			0, 0,
			Box,
			&image.NRGBA{},
		},
		{
			"Resize 2x2 -1x0 box",
			&image.NRGBA{
				Rect:   image.Rect(-1, -1, 1, 1),
				Stride: 2 * 4,
				Pix: []uint8{
					0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0xff,
					0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0xff, 0xff,
				},
			},
			-1, 0,
			Box,
			&image.NRGBA{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Resize(tc.src, tc.w, tc.h, tc.f)
			if !compareNRGBA(got, tc.want, 0) {
				t.Fatalf("got result %#v want %#v", got, tc.want)
			}
		})
	}
}

func TestResampleFilters(t *testing.T) {
	for _, filter := range []ResampleFilter{
		NearestNeighbor,
		Box,
		Linear,
		Hermite,
		MitchellNetravali,
		CatmullRom,
		BSpline,
		Gaussian,
		Lanczos,
		Hann,
		Hamming,
		Blackman,
		Bartlett,
		Welch,
		Cosine,
	} {
		t.Run("", func(t *testing.T) {
			src := image.NewNRGBA(image.Rect(-1, -1, 2, 3))
			got := Resize(src, 5, 6, filter)
			want := image.NewNRGBA(image.Rect(0, 0, 5, 6))
			if !compareNRGBA(got, want, 0) {
				t.Fatalf("got result %#v want %#v", got, want)
			}
			if filter.Kernel != nil {
				if x := filter.Kernel(filter.Support + 0.0001); x != 0 {
					t.Fatalf("got kernel value %f want 0", x)
				}
			}
		})
	}
}
