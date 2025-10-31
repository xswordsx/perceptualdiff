package perceptualdiff_test

import (
	"fmt"
	"image"
	_ "image/png"

	_ "golang.org/x/image/tiff"

	"os"
	"testing"

	"github.com/xswordsx/perceptualdiff"
)

// cases presume that the files are located in the "testdata" folder.
var cases []struct {
	imageA     string
	imageB     string
	shouldPass bool
} = []struct {
	imageA     string
	imageB     string
	shouldPass bool
}{
	{
		shouldPass: false,
		imageA:     "Bug1102605_ref.tif",
		imageB:     "Bug1102605.tif",
	},
	{
		shouldPass: true,
		imageA:     "Bug1471457_ref.tif",
		imageB:     "Bug1471457.tif",
	},
	{
		shouldPass: true,
		imageA:     "cam_mb_ref.tif",
		imageB:     "cam_mb.tif",
	},
	{
		shouldPass: false,
		imageA:     "fish2.png",
		imageB:     "fish1.png",
	},
	{
		shouldPass: false,
		imageA:     "square.png",
		imageB:     "square_scaled.png",
	},
	{
		shouldPass: false,
		imageA:     "Aqsis_vase.png",
		imageB:     "Aqsis_vase_ref.png",
	},
	{
		shouldPass: false,
		imageA:     "alpha1.png",
		imageB:     "alpha2.png",
	},
}

func readImage(filename string) (image.Image, error) {
	imageFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not load image %q: %w", filename, err)
	}
	defer imageFile.Close()
	imageData, _, err := image.Decode(imageFile)
	if err != nil {
		return nil, fmt.Errorf("could not decode image %q: %w", filename, err)
	}
	return imageData, err
}

func TestYeeCompare(t *testing.T) {
	for _, tc := range cases {
		t.Run(tc.imageA, func(t *testing.T) {
			imgA, err := readImage("./testdata/" + tc.imageA)
			if err != nil {
				t.Fatal(err)
			}
			imgB, err := readImage("./testdata/" + tc.imageB)
			if err != nil {
				t.Fatal(err)
			}

			identical, _ := perceptualdiff.Compare(imgA, imgB, perceptualdiff.DefaultParameters, nil)
			if identical != tc.shouldPass {
				not := ""
				if !tc.shouldPass {
					not = "not "
				}
				t.Errorf("expected %q images to %sbe perceptually identical", tc.imageA, not)
			}
		})
	}
}

func BenchmarkYeeCompare(b *testing.B) {
	for _, tc := range cases {
		b.Run(tc.imageA, func(b *testing.B) {
			b.StopTimer()
			imgA, err := readImage("./testdata/" + tc.imageA)
			if err != nil {
				b.Fatal(err)
			}
			imgB, err := readImage("./testdata/" + tc.imageB)
			if err != nil {
				b.Fatal(err)
			}

			b.StartTimer()
			for i := 0; i < b.N; i++ {
				perceptualdiff.Compare(imgA, imgB, perceptualdiff.DefaultParameters, nil)
			}
		})
	}
}
