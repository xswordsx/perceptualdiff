/*
Metric math & color funcs
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

package perceptualdiff

import "math"

var white struct{ x, y, z float64 }

func init() {
	x, y, z := adobe_rgb_to_xyz(1, 1, 1)
	white.x = x
	white.y = y
	white.z = z
}

func to_radians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

func to_degrees(radians float64) float64 {
	return radians * 180.0 / math.Pi
}

// Given the adaptation luminance, this function returns the
// threshold of visibility in cd per m^2.
//
// TVI means Threshold vs Intensity function.
// This version comes from Ward Larson Siggraph 1997.
//
// Returns the threshold luminance given the adaptation luminance.
// Units are candelas per meter squared.
func tvi(adaptation_luminance float64) float64 {
	log_a := math.Log10(adaptation_luminance)

	var r float64
	if log_a < -3.94 {
		r = -2.86
	} else if log_a < -1.44 {
		r = math.Pow(0.405*log_a+1.6, 2.18) - 2.86
	} else if log_a < -0.0184 {
		r = log_a - 0.395
	} else if log_a < 1.9 {
		r = math.Pow(0.249*log_a+0.65, 2.7) - 0.72
	} else {
		r = log_a - 1.255
	}

	return math.Pow(10.0, r)
}

// computes the contrast sensitivity function (Barten SPIE 1989)
// given the cycles per degree (cpd) and luminance (lum)
func csf(cpd, lum float64) float64 {
	a := 440.0 * math.Pow((1.0+0.7/lum), -0.2)
	b := 0.3 * math.Pow((1.0+100.0/lum), 0.15)

	return a * cpd * math.Exp(-b*cpd) * math.Sqrt(1.0+0.06*math.Exp(b*cpd))
}

/*
 * Visual Masking Function
 * from Daly 1993
 */
func mask(contrast float64) float64 {
	a := math.Pow(392.498*contrast, 0.7)
	b := math.Pow(0.0153*a, 4.0)
	return math.Pow(1.0+b, 0.25)
}

// convert Adobe RGB (1998) with reference white D65 to XYZ
func adobe_rgb_to_xyz(r, g, b float64) (float64, float64, float64) {
	// matrix is from http://www.brucelindbloom.com/
	return r*0.576700 + g*0.185556 + b*0.188212,
		r*0.297361 + g*0.627355 + b*0.0752847,
		r*0.0270328 + g*0.0706879 + b*0.991248
}

func xyz_to_lab(x, y, z float64) (l, a, b float64) {
	const epsilon = 216.0 / 24389.0
	const kappa = 24389.0 / 27.0
	var r = [3]float64{
		x / white.x,
		y / white.y,
		z / white.z,
	}
	var f [3]float64
	for i := 0; i < 3; i++ {
		if r[i] > epsilon {
			f[i] = math.Pow(r[i], 1.0/3.0)
		} else {
			f[i] = (kappa*r[i] + 16.0) / 116.0
		}
	}
	l = 116.0*f[1] - 16.0
	a = 500.0 * (f[0] - f[1])
	b = 200.0 * (f[1] - f[2])
	return l, a, b
}

func adaptation(num_one_degree_pixels float64) int {
	num_pixels := 1.0
	adaptation_level := 0
	for i := 0; i < MAX_PYR_LEVELS; i++ {
		adaptation_level = i
		if num_pixels > num_one_degree_pixels {
			break
		}
		num_pixels *= 2
	}
	return adaptation_level
}
