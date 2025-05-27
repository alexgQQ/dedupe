// This has been ported and cherry picked from https://github.com/kovidgoyal/imaging/tree/master
// which is the most up to date fork for the popular imaging library
// There is a lot of good functionality there but I do not need the majority of it just the image resizing

package utils

import (
	"image"
	"image/color"
	"math"
)

type scanner struct {
	image   image.Image
	w, h    int
	palette []color.NRGBA
}

func newScanner(img image.Image) *scanner {
	s := &scanner{
		image: img,
		w:     img.Bounds().Dx(),
		h:     img.Bounds().Dy(),
	}
	if img, ok := img.(*image.Paletted); ok {
		s.palette = make([]color.NRGBA, max(256, len(img.Palette)))
		for i := 0; i < len(img.Palette); i++ {
			s.palette[i] = color.NRGBAModel.Convert(img.Palette[i]).(color.NRGBA)
		}
	}
	return s
}

// scan scans the given rectangular region of the image into dst.
func (s *scanner) scan(x1, y1, x2, y2 int, dst []uint8) {
	switch img := s.image.(type) {
	case *image.NRGBA:
		size := (x2 - x1) * 4
		j := 0
		i := y1*img.Stride + x1*4
		if size == 4 {
			for y := y1; y < y2; y++ {
				d := dst[j : j+4 : j+4]
				s := img.Pix[i : i+4 : i+4]
				d[0] = s[0]
				d[1] = s[1]
				d[2] = s[2]
				d[3] = s[3]
				j += size
				i += img.Stride
			}
		} else {
			for y := y1; y < y2; y++ {
				copy(dst[j:j+size], img.Pix[i:i+size])
				j += size
				i += img.Stride
			}
		}

	case *image.NRGBA64:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*8
			for x := x1; x < x2; x++ {
				s := img.Pix[i : i+8 : i+8]
				d := dst[j : j+4 : j+4]
				d[0] = s[0]
				d[1] = s[2]
				d[2] = s[4]
				d[3] = s[6]
				j += 4
				i += 8
			}
		}

	case *image.RGBA:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*4
			for x := x1; x < x2; x++ {
				d := dst[j : j+4 : j+4]
				a := img.Pix[i+3]
				switch a {
				case 0:
					d[0] = 0
					d[1] = 0
					d[2] = 0
					d[3] = a
				case 0xff:
					s := img.Pix[i : i+4 : i+4]
					d[0] = s[0]
					d[1] = s[1]
					d[2] = s[2]
					d[3] = a
				default:
					s := img.Pix[i : i+4 : i+4]
					r16 := uint16(s[0])
					g16 := uint16(s[1])
					b16 := uint16(s[2])
					a16 := uint16(a)
					d[0] = uint8(r16 * 0xff / a16)
					d[1] = uint8(g16 * 0xff / a16)
					d[2] = uint8(b16 * 0xff / a16)
					d[3] = a
				}
				j += 4
				i += 4
			}
		}

	case *image.RGBA64:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*8
			for x := x1; x < x2; x++ {
				s := img.Pix[i : i+8 : i+8]
				d := dst[j : j+4 : j+4]
				a := s[6]
				switch a {
				case 0:
					d[0] = 0
					d[1] = 0
					d[2] = 0
				case 0xff:
					d[0] = s[0]
					d[1] = s[2]
					d[2] = s[4]
				default:
					r32 := uint32(s[0])<<8 | uint32(s[1])
					g32 := uint32(s[2])<<8 | uint32(s[3])
					b32 := uint32(s[4])<<8 | uint32(s[5])
					a32 := uint32(s[6])<<8 | uint32(s[7])
					d[0] = uint8((r32 * 0xffff / a32) >> 8)
					d[1] = uint8((g32 * 0xffff / a32) >> 8)
					d[2] = uint8((b32 * 0xffff / a32) >> 8)
				}
				d[3] = a
				j += 4
				i += 8
			}
		}

	case *image.Gray:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1
			for x := x1; x < x2; x++ {
				c := img.Pix[i]
				d := dst[j : j+4 : j+4]
				d[0] = c
				d[1] = c
				d[2] = c
				d[3] = 0xff
				j += 4
				i++
			}
		}

	case *image.Gray16:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*2
			for x := x1; x < x2; x++ {
				c := img.Pix[i]
				d := dst[j : j+4 : j+4]
				d[0] = c
				d[1] = c
				d[2] = c
				d[3] = 0xff
				j += 4
				i += 2
			}
		}

	case *image.YCbCr:
		j := 0
		x1 += img.Rect.Min.X
		x2 += img.Rect.Min.X
		y1 += img.Rect.Min.Y
		y2 += img.Rect.Min.Y

		hy := img.Rect.Min.Y / 2
		hx := img.Rect.Min.X / 2
		for y := y1; y < y2; y++ {
			iy := (y-img.Rect.Min.Y)*img.YStride + (x1 - img.Rect.Min.X)

			var yBase int
			switch img.SubsampleRatio {
			case image.YCbCrSubsampleRatio444, image.YCbCrSubsampleRatio422:
				yBase = (y - img.Rect.Min.Y) * img.CStride
			case image.YCbCrSubsampleRatio420, image.YCbCrSubsampleRatio440:
				yBase = (y/2 - hy) * img.CStride
			}

			for x := x1; x < x2; x++ {
				var ic int
				switch img.SubsampleRatio {
				case image.YCbCrSubsampleRatio444, image.YCbCrSubsampleRatio440:
					ic = yBase + (x - img.Rect.Min.X)
				case image.YCbCrSubsampleRatio422, image.YCbCrSubsampleRatio420:
					ic = yBase + (x/2 - hx)
				default:
					ic = img.COffset(x, y)
				}

				yy1 := int32(img.Y[iy]) * 0x10101
				cb1 := int32(img.Cb[ic]) - 128
				cr1 := int32(img.Cr[ic]) - 128

				r := yy1 + 91881*cr1
				if uint32(r)&0xff000000 == 0 {
					r >>= 16
				} else {
					r = ^(r >> 31)
				}

				g := yy1 - 22554*cb1 - 46802*cr1
				if uint32(g)&0xff000000 == 0 {
					g >>= 16
				} else {
					g = ^(g >> 31)
				}

				b := yy1 + 116130*cb1
				if uint32(b)&0xff000000 == 0 {
					b >>= 16
				} else {
					b = ^(b >> 31)
				}

				d := dst[j : j+4 : j+4]
				d[0] = uint8(r)
				d[1] = uint8(g)
				d[2] = uint8(b)
				d[3] = 0xff

				iy++
				j += 4
			}
		}

	case *image.Paletted:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1
			for x := x1; x < x2; x++ {
				c := s.palette[img.Pix[i]]
				d := dst[j : j+4 : j+4]
				d[0] = c.R
				d[1] = c.G
				d[2] = c.B
				d[3] = c.A
				j += 4
				i++
			}
		}

	default:
		j := 0
		b := s.image.Bounds()
		x1 += b.Min.X
		x2 += b.Min.X
		y1 += b.Min.Y
		y2 += b.Min.Y
		for y := y1; y < y2; y++ {
			for x := x1; x < x2; x++ {
				r16, g16, b16, a16 := s.image.At(x, y).RGBA()
				d := dst[j : j+4 : j+4]
				switch a16 {
				case 0xffff:
					d[0] = uint8(r16 >> 8)
					d[1] = uint8(g16 >> 8)
					d[2] = uint8(b16 >> 8)
					d[3] = 0xff
				case 0:
					d[0] = 0
					d[1] = 0
					d[2] = 0
					d[3] = 0
				default:
					d[0] = uint8(((r16 * 0xffff) / a16) >> 8)
					d[1] = uint8(((g16 * 0xffff) / a16) >> 8)
					d[2] = uint8(((b16 * 0xffff) / a16) >> 8)
					d[3] = uint8(a16 >> 8)
				}
				j += 4
			}
		}
	}
}

type indexWeight struct {
	index  int
	weight float64
}

// clamp rounds and clamps float64 value to fit into uint8.
func clamp(x float64) uint8 {
	v := int64(x + 0.5)
	if v > 255 {
		return 255
	}
	if v > 0 {
		return uint8(v)
	}
	return 0
}

func precomputeWeights(dstSize, srcSize int, filter ResampleFilter) [][]indexWeight {
	du := float64(srcSize) / float64(dstSize)
	scale := du
	if scale < 1.0 {
		scale = 1.0
	}
	ru := math.Ceil(scale * filter.Support)

	out := make([][]indexWeight, dstSize)
	tmp := make([]indexWeight, 0, dstSize*int(ru+2)*2)

	for v := 0; v < dstSize; v++ {
		fu := (float64(v)+0.5)*du - 0.5

		begin := int(math.Ceil(fu - ru))
		if begin < 0 {
			begin = 0
		}
		end := int(math.Floor(fu + ru))
		if end > srcSize-1 {
			end = srcSize - 1
		}

		var sum float64
		for u := begin; u <= end; u++ {
			w := filter.Kernel((float64(u) - fu) / scale)
			if w != 0 {
				sum += w
				tmp = append(tmp, indexWeight{index: u, weight: w})
			}
		}
		if sum != 0 {
			for i := range tmp {
				tmp[i].weight /= sum
			}
		}

		out[v] = tmp
		tmp = tmp[len(tmp):]
	}

	return out
}

// Resize resizes the image to the specified width and height using the specified resampling
// filter and returns the transformed image. If one of width or height is 0, the image aspect
// ratio is preserved.
//
// Example:
//
//	dstImage := imaging.Resize(srcImage, 800, 600, imaging.Lanczos)
func Resize(img image.Image, width, height int, filter ResampleFilter) *image.NRGBA {
	dstW, dstH := width, height
	if dstW < 0 || dstH < 0 {
		return &image.NRGBA{}
	}
	if dstW == 0 && dstH == 0 {
		return &image.NRGBA{}
	}

	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()
	if srcW <= 0 || srcH <= 0 {
		return &image.NRGBA{}
	}

	// I do not need these for my use cases
	// If new width or height is 0 then preserve aspect ratio, minimum 1px.
	// if dstW == 0 {
	// 	tmpW := float64(dstH) * float64(srcW) / float64(srcH)
	// 	dstW = int(math.Max(1.0, math.Floor(tmpW+0.5)))
	// }
	// if dstH == 0 {
	// 	tmpH := float64(dstW) * float64(srcH) / float64(srcW)
	// 	dstH = int(math.Max(1.0, math.Floor(tmpH+0.5)))
	// }

	// if srcW == dstW && srcH == dstH {
	// 	return Clone(img)
	// }

	if filter.Support <= 0 {
		// Nearest-neighbor special case.
		return resizeNearest(img, dstW, dstH)
	}

	if srcW != dstW && srcH != dstH {
		return resizeVertical(resizeHorizontal(img, dstW, filter), dstH, filter)
	}
	if srcW != dstW {
		return resizeHorizontal(img, dstW, filter)
	}
	return resizeVertical(img, dstH, filter)

}

func resizeHorizontal(img image.Image, width int, filter ResampleFilter) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, width, src.h))
	weights := precomputeWeights(width, src.w, filter)
	scanLine := make([]uint8, src.w*4)
	for y := 0; y < src.h; y++ {
		src.scan(0, y, src.w, y+1, scanLine)
		j0 := y * dst.Stride
		for x := range weights {
			var r, g, b, a float64
			for _, w := range weights[x] {
				i := w.index * 4
				s := scanLine[i : i+4 : i+4]
				aw := float64(s[3]) * w.weight
				r += float64(s[0]) * aw
				g += float64(s[1]) * aw
				b += float64(s[2]) * aw
				a += aw
			}
			if a != 0 {
				aInv := 1 / a
				j := j0 + x*4
				d := dst.Pix[j : j+4 : j+4]
				d[0] = clamp(r * aInv)
				d[1] = clamp(g * aInv)
				d[2] = clamp(b * aInv)
				d[3] = clamp(a)
			}
		}
	}
	// Leaving these for now as I like the implementation of the parallel processing
	// in the imaging library https://github.com/kovidgoyal/imaging/blob/master/utils.go#L20
	// parallel(0, src.h, func(ys <-chan int) {
	// 	scanLine := make([]uint8, src.w*4)
	// 	for y := range ys {
	// 		src.scan(0, y, src.w, y+1, scanLine)
	// 		j0 := y * dst.Stride
	// 		for x := range weights {
	// 			var r, g, b, a float64
	// 			for _, w := range weights[x] {
	// 				i := w.index * 4
	// 				s := scanLine[i : i+4 : i+4]
	// 				aw := float64(s[3]) * w.weight
	// 				r += float64(s[0]) * aw
	// 				g += float64(s[1]) * aw
	// 				b += float64(s[2]) * aw
	// 				a += aw
	// 			}
	// 			if a != 0 {
	// 				aInv := 1 / a
	// 				j := j0 + x*4
	// 				d := dst.Pix[j : j+4 : j+4]
	// 				d[0] = clamp(r * aInv)
	// 				d[1] = clamp(g * aInv)
	// 				d[2] = clamp(b * aInv)
	// 				d[3] = clamp(a)
	// 			}
	// 		}
	// 	}
	// })
	return dst
}

func resizeVertical(img image.Image, height int, filter ResampleFilter) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, height))
	weights := precomputeWeights(height, src.h, filter)
	scanLine := make([]uint8, src.h*4)
	for x := 0; x < src.w; x++ {
		src.scan(x, 0, x+1, src.h, scanLine)
		for y := range weights {
			var r, g, b, a float64
			for _, w := range weights[y] {
				i := w.index * 4
				s := scanLine[i : i+4 : i+4]
				aw := float64(s[3]) * w.weight
				r += float64(s[0]) * aw
				g += float64(s[1]) * aw
				b += float64(s[2]) * aw
				a += aw
			}
			if a != 0 {
				aInv := 1 / a
				j := y*dst.Stride + x*4
				d := dst.Pix[j : j+4 : j+4]
				d[0] = clamp(r * aInv)
				d[1] = clamp(g * aInv)
				d[2] = clamp(b * aInv)
				d[3] = clamp(a)
			}
		}
	}
	// parallel(0, src.w, func(xs <-chan int) {
	// 	scanLine := make([]uint8, src.h*4)
	// 	for x := range xs {
	// 		src.scan(x, 0, x+1, src.h, scanLine)
	// 		for y := range weights {
	// 			var r, g, b, a float64
	// 			for _, w := range weights[y] {
	// 				i := w.index * 4
	// 				s := scanLine[i : i+4 : i+4]
	// 				aw := float64(s[3]) * w.weight
	// 				r += float64(s[0]) * aw
	// 				g += float64(s[1]) * aw
	// 				b += float64(s[2]) * aw
	// 				a += aw
	// 			}
	// 			if a != 0 {
	// 				aInv := 1 / a
	// 				j := y*dst.Stride + x*4
	// 				d := dst.Pix[j : j+4 : j+4]
	// 				d[0] = clamp(r * aInv)
	// 				d[1] = clamp(g * aInv)
	// 				d[2] = clamp(b * aInv)
	// 				d[3] = clamp(a)
	// 			}
	// 		}
	// 	}
	// })
	return dst
}

// Clone returns a copy of the given image.
func Clone(img image.Image) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, src.h))
	size := src.w * 4
	for y := 0; y < src.h; y++ {
		i := y * dst.Stride
		src.scan(0, y, src.w, y+1, dst.Pix[i:i+size])
	}
	return dst
}

func toNRGBA(img image.Image) *image.NRGBA {
	if img, ok := img.(*image.NRGBA); ok {
		return &image.NRGBA{
			Pix:    img.Pix,
			Stride: img.Stride,
			Rect:   img.Rect.Sub(img.Rect.Min),
		}
	}
	return Clone(img)
}

// There is only one resample filter that uses this and I'm not sure if I'll use it
// resizeNearest is a fast nearest-neighbor resize, no filtering.
func resizeNearest(img image.Image, width, height int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	dx := float64(img.Bounds().Dx()) / float64(width)
	dy := float64(img.Bounds().Dy()) / float64(height)

	if dx > 1 && dy > 1 {
		src := newScanner(img)
		for y := 0; y < height; y++ {
			srcY := int((float64(y) + 0.5) * dy)
			dstOff := y * dst.Stride
			for x := 0; x < width; x++ {
				srcX := int((float64(x) + 0.5) * dx)
				src.scan(srcX, srcY, srcX+1, srcY+1, dst.Pix[dstOff:dstOff+4])
				dstOff += 4
			}
		}
		// parallel(0, height, func(ys <-chan int) {
		// 	for y := range ys {
		// 		srcY := int((float64(y) + 0.5) * dy)
		// 		dstOff := y * dst.Stride
		// 		for x := 0; x < width; x++ {
		// 			srcX := int((float64(x) + 0.5) * dx)
		// 			src.scan(srcX, srcY, srcX+1, srcY+1, dst.Pix[dstOff:dstOff+4])
		// 			dstOff += 4
		// 		}
		// 	}
		// })
	} else {
		src := toNRGBA(img)
		for y := 0; y < height; y++ {
			srcY := int((float64(y) + 0.5) * dy)
			srcOff0 := srcY * src.Stride
			dstOff := y * dst.Stride
			for x := 0; x < width; x++ {
				srcX := int((float64(x) + 0.5) * dx)
				srcOff := srcOff0 + srcX*4
				copy(dst.Pix[dstOff:dstOff+4], src.Pix[srcOff:srcOff+4])
				dstOff += 4
			}
		}
		// parallel(0, height, func(ys <-chan int) {
		// 	for y := range ys {
		// 		srcY := int((float64(y) + 0.5) * dy)
		// 		srcOff0 := srcY * src.Stride
		// 		dstOff := y * dst.Stride
		// 		for x := 0; x < width; x++ {
		// 			srcX := int((float64(x) + 0.5) * dx)
		// 			srcOff := srcOff0 + srcX*4
		// 			copy(dst.Pix[dstOff:dstOff+4], src.Pix[srcOff:srcOff+4])
		// 			dstOff += 4
		// 		}
		// 	}
		// })
	}

	return dst
}

// ResampleFilter specifies a resampling filter to be used for image resizing.
//
//	General filter recommendations:
//
//	- Lanczos
//		A high-quality resampling filter for photographic images yielding sharp results.
//
//	- CatmullRom
//		A sharp cubic filter that is faster than Lanczos filter while providing similar results.
//
//	- MitchellNetravali
//		A cubic filter that produces smoother results with less ringing artifacts than CatmullRom.
//
//	- Linear
//		Bilinear resampling filter, produces a smooth output. Faster than cubic filters.
//
//	- Box
//		Simple and fast averaging filter appropriate for downscaling.
//		When upscaling it's similar to NearestNeighbor.
//
//	- NearestNeighbor
//		Fastest resampling filter, no antialiasing.
type ResampleFilter struct {
	Support float64
	Kernel  func(float64) float64
}

// NearestNeighbor is a nearest-neighbor filter (no anti-aliasing).
var NearestNeighbor ResampleFilter

// Box filter (averaging pixels).
var Box ResampleFilter

// Linear filter.
var Linear ResampleFilter

// Hermite cubic spline filter (BC-spline; B=0; C=0).
var Hermite ResampleFilter

// MitchellNetravali is Mitchell-Netravali cubic filter (BC-spline; B=1/3; C=1/3).
var MitchellNetravali ResampleFilter

// CatmullRom is a Catmull-Rom - sharp cubic filter (BC-spline; B=0; C=0.5).
var CatmullRom ResampleFilter

// BSpline is a smooth cubic filter (BC-spline; B=1; C=0).
var BSpline ResampleFilter

// Gaussian is a Gaussian blurring filter.
var Gaussian ResampleFilter

// Bartlett is a Bartlett-windowed sinc filter (3 lobes).
var Bartlett ResampleFilter

// Lanczos filter (3 lobes).
var Lanczos ResampleFilter

// Hann is a Hann-windowed sinc filter (3 lobes).
var Hann ResampleFilter

// Hamming is a Hamming-windowed sinc filter (3 lobes).
var Hamming ResampleFilter

// Blackman is a Blackman-windowed sinc filter (3 lobes).
var Blackman ResampleFilter

// Welch is a Welch-windowed sinc filter (parabolic window, 3 lobes).
var Welch ResampleFilter

// Cosine is a Cosine-windowed sinc filter (3 lobes).
var Cosine ResampleFilter

func bcspline(x, b, c float64) float64 {
	var y float64
	x = math.Abs(x)
	if x < 1.0 {
		y = ((12-9*b-6*c)*x*x*x + (-18+12*b+6*c)*x*x + (6 - 2*b)) / 6
	} else if x < 2.0 {
		y = ((-b-6*c)*x*x*x + (6*b+30*c)*x*x + (-12*b-48*c)*x + (8*b + 24*c)) / 6
	}
	return y
}

func sinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	return math.Sin(math.Pi*x) / (math.Pi * x)
}

// This is the only init function in the package so it should run before main
// but I need to look more into how to structure this or if I need to change anything
func init() {
	NearestNeighbor = ResampleFilter{
		Support: 0.0, // special case - not applying the filter
	}

	Box = ResampleFilter{
		Support: 0.5,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x <= 0.5 {
				return 1.0
			}
			return 0
		},
	}

	Linear = ResampleFilter{
		Support: 1.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 1.0 {
				return 1.0 - x
			}
			return 0
		},
	}

	Hermite = ResampleFilter{
		Support: 1.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 1.0 {
				return bcspline(x, 0.0, 0.0)
			}
			return 0
		},
	}

	MitchellNetravali = ResampleFilter{
		Support: 2.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 2.0 {
				return bcspline(x, 1.0/3.0, 1.0/3.0)
			}
			return 0
		},
	}

	CatmullRom = ResampleFilter{
		Support: 2.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 2.0 {
				return bcspline(x, 0.0, 0.5)
			}
			return 0
		},
	}

	BSpline = ResampleFilter{
		Support: 2.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 2.0 {
				return bcspline(x, 1.0, 0.0)
			}
			return 0
		},
	}

	Gaussian = ResampleFilter{
		Support: 2.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 2.0 {
				return math.Exp(-2 * x * x)
			}
			return 0
		},
	}

	Bartlett = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * (3.0 - x) / 3.0
			}
			return 0
		},
	}

	Lanczos = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * sinc(x/3.0)
			}
			return 0
		},
	}

	Hann = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * (0.5 + 0.5*math.Cos(math.Pi*x/3.0))
			}
			return 0
		},
	}

	Hamming = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * (0.54 + 0.46*math.Cos(math.Pi*x/3.0))
			}
			return 0
		},
	}

	Blackman = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * (0.42 - 0.5*math.Cos(math.Pi*x/3.0+math.Pi) + 0.08*math.Cos(2.0*math.Pi*x/3.0))
			}
			return 0
		},
	}

	Welch = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * (1.0 - (x * x / 9.0))
			}
			return 0
		},
	}

	Cosine = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * math.Cos((math.Pi/2.0)*(x/3.0))
			}
			return 0
		},
	}
}
