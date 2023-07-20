/*
Laplacian Pyramid
Copyright (C) 2006-2011 Yangli Hector Yee
Copyright (C) 2011-2016 Steven Myint, Jeff Terrace
Copyright (C) 2023 Ivan Latunov

This program is free software; you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation; either version 2 of the License, or (at your option) any later
version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY
WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with
this program; if not, write to the Free Software Foundation, Inc., 59 Temple
Place, Suite 330, Boston, MA 02111-1307 USA
*/

package perceptualdiff

// The maximum amount of pyramid levels to construct.
const MAX_PYR_LEVELS = 8

// pyramid is a Laplacian pyramid.
type pyramid struct {
	width  int
	height int
	// Successively blurred versions of the original image.
	levels [MAX_PYR_LEVELS][]float64
}

func newPyramid(image []float64, width, height int) *pyramid {
	l := &pyramid{
		width:  int(width),
		height: int(height),
	}

	for i := 0; i < MAX_PYR_LEVELS; i++ {
		if i == 0 || width*height <= 1 {
			l.levels[i] = image
		} else {
			l.levels[i] = make([]float64, l.width*l.height)
			l.convolve(l.levels[i], l.levels[i-1])
		}
	}

	return l
}

func (l *pyramid) get_value(x, y, level int) float64 {
	index := x + y*int(l.width)
	// assert(level < MAX_PYR_LEVELS)
	return l.levels[level][index]
}

func (l *pyramid) convolve(a, b []float64) {
	if len(a) == 0 || len(b) == 0 {
		panic("empty source or destination")
	}
	for y := 0; y < l.height; y++ {
		for x := 0; x < l.width; x++ {
			index := y*l.width + x
			var result float64
			for i := -2; i <= 2; i++ {
				for j := -2; j <= 2; j++ {
					nx := x + i
					ny := y + j
					nx = abs(nx)
					ny = abs(ny)
					if nx >= l.width {
						nx = 2*l.width - nx - 1
					}
					if ny >= l.height {
						ny = 2*l.height - ny - 1
					}

					kernel := []float64{0.05, 0.25, 0.4, 0.25, 0.05}

					result +=
						kernel[i+2] * kernel[j+2] * b[ny*l.width+nx]
				}
			}
			a[index] = result
		}
	}
}

func abs(x int) int {
	if x >= 0 {
		return x
	}
	return -x
}
