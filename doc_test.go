package perceptualdiff_test

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"os"

	"github.com/xswordsx/perceptualdiff"
)

func Example_basic() {
	// All error-handling is omitted for the sake of brevity.

	img_a_fd, _ := os.Open("image_a.jpg")
	img_b_fd, _ := os.Open("image_b.jpg")

	image_a, _, _ := image.Decode(img_a_fd)
	image_b, _, _ := image.Decode(img_b_fd)

	identical, result := perceptualdiff.Compare(
		image_a, image_b,
		perceptualdiff.DefaultParameters, // or make your own
		os.Stdout,                        // passing nil discards logs
	)
	fmt.Printf("Identical: %v\n", identical)
	fmt.Printf("Results:   %+v\n", result)
	if !identical {
		file := os.Stdout
		_ = png.Encode(file, result.ImageDifference)
	}
}
