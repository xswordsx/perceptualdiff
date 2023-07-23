/*
Metric
Copyright (C) 2006-2011 Yangli Hector Yee
Copyright (C) 2011-2016 Steven Myint, Jeff Terrace
Copyright (C) 2023 Ivan Latunov

This program is free software; you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation; either version 2 of the License, or (at your option) any later
version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY
WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
PARTICULAR PURPOSE.  See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with
this program; if not, write to the Free Software Foundation, Inc., 59 Temple
Place, Suite 330, Boston, MA 02111-1307 USA
*/

// Package perceptualdiff is a Go port of
// https://github.com/myint/perceptualdiff.
package perceptualdiff

import (
	"image"
	"image/color"
	"io"
	"math"
	"sync"
	"sync/atomic"
)

const (
	ReasonDimensionMismatch = "Image dimensions do not match"
	ReasonBinaryIdentical   = "Images are binary identical"
	ReasonIndistinguishable = "Images are perceptually indistinguishable"
	ReasonVisiblyDifferent  = "Images are visibly different"
)

// Parameters are the available parameters for image comparison.
type Parameters struct {
	// Only consider luminance; ignore chroma channels in the comparison.
	LuminanceOnly bool

	// Field of view in degrees. Range is [0.1, 89.9].
	FieldOfView float64

	// The Gamma to convert to linear color space.
	Gamma float64

	// White luminance.
	Luminance float64

	// How many pixels different to ignore.
	ThresholdPixels uint

	// How much color to use in the metric.
	//   - 0.0 is the same as ``LuminanceOnly'' = true,
	//   - 1.0 means full strength.
	ColorFactor float64
}

// CompareResult is the result of a comparison between two images.
type CompareResult struct {
	Reason          string      // Reason for the response.
	NumPixelsFailed uint64      // Number of pixels that failed the perceptual check.
	ErrorSum        float64     // Sum of the deltas of all pixels.
	ImageDifference *image.RGBA // Bitmask that shows which pixels failed the check.
}

var (
	// DefaultParameters are the default parameters for the [Yee_compare] func.
	DefaultParameters Parameters

	global_white struct{ x, y, z float64 }
)

func init() {
	x, y, z := adobe_rgb_to_xyz(1, 1, 1)
	global_white.x = x
	global_white.y = y
	global_white.z = z

	DefaultParameters = Parameters{
		LuminanceOnly:   false,
		FieldOfView:     45.0,
		Gamma:           2.2,
		Luminance:       100.0,
		ThresholdPixels: 100,
		ColorFactor:     1.0,
	}
}

// Image comparison metric using Yee's method.
//
// References: A Perceptual Metric for Production Testing, Hector Yee,
// Journal of Graphics Tools 2004
func YeeCompare(image_a, image_b image.Image, args Parameters, output_verbose io.Writer) (
	perceptually_identical bool,
	output CompareResult,
) {
	if output_verbose == nil {
		output_verbose = io.Discard
	}

	a_size := image_a.Bounds().Size()
	b_size := image_b.Bounds().Size()

	if a_size != b_size {
		return false, CompareResult{Reason: ReasonDimensionMismatch}
	}

	w := a_size.X
	h := a_size.Y
	dim := w * h

	bounds := image_a.Bounds()
	identical := true
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ar, ag, ab, aa := image_a.At(x, y).RGBA()
			br, bg, bb, ba := image_b.At(x, y).RGBA()
			if !(ar == br && ag == bg && ab == bb && aa == ba) {
				identical = false
				// force break
				x = bounds.Max.X
				y = bounds.Max.Y
			}
		}
	}
	if identical {
		return true, CompareResult{Reason: ReasonBinaryIdentical}
	}

	// Assuming colorspaces are in Adobe RGB (1998) convert to XYZ.
	a_lum := make([]float64, dim)
	b_lum := make([]float64, dim)

	a_a := make([]float64, dim)
	b_a := make([]float64, dim)
	a_b := make([]float64, dim)
	b_b := make([]float64, dim)

	_, _ = output_verbose.Write([]byte("Converting RGB to XYZ\n"))

	gamma := args.Gamma
	luminance := args.Luminance

	wg := sync.WaitGroup{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		wg.Add(1)
		go func(y int) {
			defer wg.Done()

			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				i := x + y*w

				// The RGBA func returns alpha-premultiplied values in the [0, 0xffff] range.
				a_color_R, a_color_G, a_color_B, _ := image_a.At(x, y).RGBA()
				const maxValue = float64(0xffff)

				a_x, a_y, a_z := adobe_rgb_to_xyz(
					math.Pow(float64(a_color_R)/maxValue, gamma),
					math.Pow(float64(a_color_G)/maxValue, gamma),
					math.Pow(float64(a_color_B)/maxValue, gamma),
				)
				_, a_a[i], a_b[i] = xyz_to_lab(a_x, a_y, a_z)

				b_color_R, b_color_G, b_color_B, _ := image_b.At(x, y).RGBA()

				b_x, b_y, b_z := adobe_rgb_to_xyz(
					math.Pow(float64(b_color_R)/maxValue, gamma),
					math.Pow(float64(b_color_G)/maxValue, gamma),
					math.Pow(float64(b_color_B)/maxValue, gamma),
				)
				_, b_a[i], b_b[i] = xyz_to_lab(b_x, b_y, b_z)

				a_lum[i] = a_y * luminance
				b_lum[i] = b_y * luminance
			}
		}(y)
	}

	wg.Wait()

	num_one_degree_pixels := to_degrees(2 * math.Tan(args.FieldOfView*to_radians(.5)))
	pixels_per_degree := float64(w) / num_one_degree_pixels

	_, _ = output_verbose.Write([]byte("Performing test\n"))

	adaptation_level := adaptation(num_one_degree_pixels)

	cpd := [MAX_PYR_LEVELS]float64{}
	cpd[0] = 0.5 * pixels_per_degree
	for i := 1; i < MAX_PYR_LEVELS; i++ {
		cpd[i] = 0.5 * cpd[i-1]
	}
	csf_max := csf(3.248, 100.0)

	// Omit static assert

	f_freq := [MAX_PYR_LEVELS - 2]float64{}
	for i := 0; i < MAX_PYR_LEVELS-2; i++ {
		f_freq[i] = csf_max / csf(cpd[i], 100.0)
	}

	var pixels_failed atomic.Uint64
	var error_sum uint64 // will be used with the atomic* funcs as a float64

	_, _ = output_verbose.Write([]byte("Constructing Laplacian Pyramids\n"))

	diffImg := image.NewRGBA(image_a.Bounds())

	la := newPyramid(a_lum, w, h)
	lb := newPyramid(b_lum, w, h)

	wg = sync.WaitGroup{}
	for y := 0; y < h; y++ {
		wg.Add(1)
		go func(y int) {
			defer wg.Done()
			for x := 0; x < w; x++ {
				index := y*w + x

				adapt := math.Max(
					(la.get_value(x, y, adaptation_level)+lb.get_value(x, y, adaptation_level))*0.5,
					1e-5)

				var (
					sum_contrast float64
					factor       float64
				)

				for i := 0; i < MAX_PYR_LEVELS-2; i++ {
					n1 := math.Abs(la.get_value(x, y, i) - la.get_value(x, y, i+1))
					n2 := math.Abs(lb.get_value(x, y, i) - lb.get_value(x, y, i+1))

					numerator := math.Max(n1, n2)
					d1 := math.Abs(la.get_value(x, y, i+2))
					d2 := math.Abs(lb.get_value(x, y, i+2))
					denominator := math.Max(math.Max(d1, d2), 1e-5)
					contrast := numerator / denominator
					f_mask := mask(contrast * csf(cpd[i], adapt))
					factor += contrast * f_freq[i] * f_mask
					sum_contrast += contrast
				}
				sum_contrast = math.Max(sum_contrast, 1e-5)
				factor /= sum_contrast
				factor = math.Min(math.Max(factor, 1.0), 10.0)
				delta := math.Abs(la.get_value(x, y, 0) -
					lb.get_value(x, y, 0))
				atomicAddFloat64(&error_sum, delta)
				pass := true

				// Pure luminance test.
				if delta > factor*tvi(adapt) {
					pass = false
				}

				if !args.LuminanceOnly {
					// CIE delta E test with modifications.
					color_scale := args.ColorFactor

					// Ramp down the color test in scotopic regions.
					if adapt < 10.0 {
						// Don't do color test at all.
						color_scale = 0.0
					}

					da := a_a[index] - b_a[index]
					db := a_b[index] - b_b[index]
					delta_e := (da*da + db*db) * color_scale
					atomicAddFloat64(&error_sum, delta_e)
					if delta_e > factor {
						pass = false
					}
				}

				if pass {
					diffImg.SetRGBA(int(x), int(y), color.RGBA{0, 0, 0, 255})
				} else {
					pixels_failed.Add(1)
					diffImg.SetRGBA(int(x), int(y), color.RGBA{0, 0, 255, 255})
				}
			}
		}(y)
	}

	wg.Wait()

	var (
		perceptuallyIdentical bool = uint(pixels_failed.Load()) < args.ThresholdPixels
		reason                string
	)
	if perceptuallyIdentical {
		reason = ReasonIndistinguishable
	} else {
		reason = ReasonVisiblyDifferent
	}

	return perceptuallyIdentical, CompareResult{
		Reason:          reason,
		NumPixelsFailed: pixels_failed.Load(),
		ErrorSum:        atomicLoadFloat64(&error_sum),
		ImageDifference: diffImg,
	}
}

func atomicLoadFloat64(v *uint64) float64 {
	return math.Float64frombits(atomic.LoadUint64(v))
}

func atomicAddFloat64(v *uint64, delta float64) (new float64) {
	for {
		old := atomic.LoadUint64(v)
		new := math.Float64frombits(old) + delta

		if atomic.CompareAndSwapUint64(v, old, math.Float64bits(new)) {
			return new
		}
	}
}
