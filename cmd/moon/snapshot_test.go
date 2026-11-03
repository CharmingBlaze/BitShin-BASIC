package main

import (
	"image"
	"image/color"
	"testing"
)

func flatImage(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestCompareImagesIgnoresSmallChanges(t *testing.T) {
	a := flatImage(10, 10, color.RGBA{100, 100, 100, 255})
	b := flatImage(10, 10, color.RGBA{100 + snapshotPixelTol, 100, 100 - snapshotPixelTol, 255})
	share, _, err := compareImages(a, b)
	if err != nil || share != 0 {
		t.Fatalf("within the tolerance: share %v, err %v", share, err)
	}
}

func TestCompareImagesCountsChangedPixels(t *testing.T) {
	a := flatImage(10, 10, color.RGBA{100, 100, 100, 255})
	b := flatImage(10, 10, color.RGBA{100, 100, 100, 255})
	b.SetRGBA(3, 4, color.RGBA{100, 200, 100, 255})
	b.SetRGBA(5, 6, color.RGBA{0, 100, 100, 255})
	share, diff, err := compareImages(a, b)
	if err != nil || share != 0.02 {
		t.Fatalf("two of 100 pixels: share %v, err %v", share, err)
	}
	if diff.RGBAAt(3, 4) != (color.RGBA{255, 0, 255, 255}) {
		t.Fatalf("changed pixel not marked: %v", diff.RGBAAt(3, 4))
	}
}

func TestCompareImagesRejectsOtherSizes(t *testing.T) {
	a := flatImage(10, 10, color.RGBA{})
	b := flatImage(12, 10, color.RGBA{})
	if _, _, err := compareImages(a, b); err == nil {
		t.Fatal("a different size must fail")
	}
}
