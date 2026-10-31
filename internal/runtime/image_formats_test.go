package runtime

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/g3n/engine/texture"
	"golang.org/x/image/bmp"
)

// LoadTexture and LoadImage both end in image.Decode, so the formats
// registered by internal/imgfmt reach G3N's texture decoder.
func TestTextureDecoderReadsTGAAndBMP(t *testing.T) {
	dir := t.TempDir()
	tga := []byte{0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 24, 0x20, 0, 0, 255}
	tgaPath := filepath.Join(dir, "red.tga")
	if err := os.WriteFile(tgaPath, tga, 0o644); err != nil {
		t.Fatal(err)
	}
	rgba, err := texture.DecodeImage(tgaPath)
	if err != nil {
		t.Fatal(err)
	}
	if c := rgba.RGBAAt(0, 0); c != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("tga pixel %v", c)
	}

	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.Set(1, 1, color.RGBA{0, 0, 255, 255})
	var buf bytes.Buffer
	if err := bmp.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	bmpPath := filepath.Join(dir, "blue.bmp")
	if err := os.WriteFile(bmpPath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	rgba, err = texture.DecodeImage(bmpPath)
	if err != nil {
		t.Fatal(err)
	}
	if c := rgba.RGBAAt(1, 1); c.B != 255 {
		t.Fatalf("bmp pixel %v", c)
	}
}
