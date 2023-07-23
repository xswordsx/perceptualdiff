# perceptualdiff

[![Go Reference](https://pkg.go.dev/badge/github.com/xswordsx/perceptualdiff.svg)](https://pkg.go.dev/github.com/xswordsx/perceptualdiff)

A program that compares two images using a perceptually based image metric.

This is a Go-port of the [perceptualdiff tool](https://github.com/myint/perceptualdiff).

**:warning: This is still in alpha! API changes may occur at any moment!**

## Usage

For a more detailed description refer to the [GoDoc](https://pkg.go.dev/github.com/xswordsx/perceptualdiff).

```go
package main

import (
	"image"
	"image/png"
	"fmt"
	"os"

	"github.com/xswordsx/perceptualdiff"
)

func main() {
	img_a_fd, _ := os.Open("image_a.jpg")
	img_b_fd, _ := os.Open("image_b.jpg")

	image_a, _, _ := image.Decode(img_a_fd)
	image_b, _, _ := image.Decode(img_b_fd)

	identical, result := perceptualdiff.YeeCompare(
		image_a, image_b,
		perceptualdiff.DefaultParameters, // or make your own
		os.Stdout,                        // passing nil discards logs
	)
	fmt.Printf("Identical: %v\n", identical)
	fmt.Printf("Results:   %+v\n", result)
	if !identical {
		file := io.Discard // your output file here
		_ = png.Encode(file, result.ImageDifference)
	}
}
```
