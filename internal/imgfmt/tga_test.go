package imgfmt

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"testing"

	"golang.org/x/image/bmp"
)

func tgaHdr(cmapType, imgType byte, cmapLen uint16, cmapDepth byte, w, h int, depth, desc byte) []byte {
	b := make([]byte, 18)
	b[1], b[2] = cmapType, imgType
	binary.LittleEndian.PutUint16(b[5:], cmapLen)
	b[7] = cmapDepth
	binary.LittleEndian.PutUint16(b[12:], uint16(w))
	binary.LittleEndian.PutUint16(b[14:], uint16(h))
	b[16], b[17] = depth, desc
	return b
}

func decode(t *testing.T, b []byte) image.Image {
	t.Helper()
	img, format, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if format != "tga" {
		t.Fatalf("format %q", format)
	}
	return img
}

func nrgba(img image.Image, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}

func TestTGAUncompressed24BottomUp(t *testing.T) {
	// 2x2, bottom-left origin: the first row in the file is the bottom row.
	b := tgaHdr(0, 2, 0, 0, 2, 2, 24, 0)
	b = append(b,
		0, 0, 255, 0, 255, 0, // bottom: red, green
		255, 0, 0, 255, 255, 255, // top: blue, white
	)
	img := decode(t, b)
	if c := nrgba(img, 0, 1); c != (color.NRGBA{255, 0, 0, 255}) {
		t.Fatalf("bottom-left %v", c)
	}
	if c := nrgba(img, 0, 0); c != (color.NRGBA{0, 0, 255, 255}) {
		t.Fatalf("top-left %v", c)
	}
	if c := nrgba(img, 1, 0); c != (color.NRGBA{255, 255, 255, 255}) {
		t.Fatalf("top-right %v", c)
	}
}

func TestTGARLE32TopDown(t *testing.T) {
	b := tgaHdr(0, 10, 0, 0, 3, 1, 32, 0x28)
	b = append(b,
		0x81, 10, 20, 30, 128, // run of 2
		0x00, 1, 2, 3, 4, // 1 literal
	)
	img := decode(t, b)
	if c := nrgba(img, 1, 0); c != (color.NRGBA{30, 20, 10, 128}) {
		t.Fatalf("run pixel %v", c)
	}
	if c := nrgba(img, 2, 0); c != (color.NRGBA{3, 2, 1, 4}) {
		t.Fatalf("literal pixel %v", c)
	}
}

func TestTGAZeroAlphaIsOpaque(t *testing.T) {
	b := tgaHdr(0, 2, 0, 0, 1, 1, 32, 0x20)
	b = append(b, 1, 2, 3, 0)
	if c := nrgba(decode(t, b), 0, 0); c.A != 255 {
		t.Fatalf("alpha %d", c.A)
	}
}

func TestTGAGreyRLEAndColourMapped(t *testing.T) {
	b := tgaHdr(0, 11, 0, 0, 4, 1, 8, 0x20)
	b = append(b, 0x83, 77)
	if c := nrgba(decode(t, b), 3, 0); c != (color.NRGBA{77, 77, 77, 255}) {
		t.Fatalf("grey %v", c)
	}

	b = tgaHdr(1, 1, 2, 24, 2, 1, 8, 0x20)
	b = append(b, 0, 0, 255, 0, 255, 0) // palette: red, green
	b = append(b, 1, 0)
	img := decode(t, b)
	if c := nrgba(img, 0, 0); c != (color.NRGBA{0, 255, 0, 255}) {
		t.Fatalf("index 1 %v", c)
	}
	if c := nrgba(img, 1, 0); c != (color.NRGBA{255, 0, 0, 255}) {
		t.Fatalf("index 0 %v", c)
	}
}

func TestTGAConfigAndTruncated(t *testing.T) {
	b := tgaHdr(0, 2, 0, 0, 5, 7, 24, 0)
	cfg, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || format != "tga" || cfg.Width != 5 || cfg.Height != 7 {
		t.Fatalf("config %v %q %v", cfg, format, err)
	}
	if _, _, err := image.Decode(bytes.NewReader(b)); err == nil {
		t.Fatal("truncated pixels should fail")
	}
}

func TestBMPRegistered(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	src.Set(2, 1, color.NRGBA{9, 8, 7, 255})
	var buf bytes.Buffer
	if err := bmp.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	img, format, err := image.Decode(&buf)
	if err != nil || format != "bmp" {
		t.Fatalf("%q %v", format, err)
	}
	if c := nrgba(img, 2, 1); c != (color.NRGBA{9, 8, 7, 255}) {
		t.Fatalf("pixel %v", c)
	}
}

// A 1x1 lossless WebP.
var tinyWebP = []byte("RIFF\x1a\x00\x00\x00WEBPVP8L\x0d\x00\x00\x00\x2f\x00\x00\x00\x10\x07\x10\x11\x11\x88\x88\xfe\x07\x00")

func TestWebPRegistered(t *testing.T) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(tinyWebP))
	if err != nil || format != "webp" {
		t.Fatalf("%q %v", format, err)
	}
	if cfg.Width != 1 || cfg.Height != 1 {
		t.Fatalf("size %dx%d", cfg.Width, cfg.Height)
	}
	if _, _, err := image.Decode(bytes.NewReader(tinyWebP)); err != nil {
		t.Fatal(err)
	}
}
