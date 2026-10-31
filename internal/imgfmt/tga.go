package imgfmt

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
)

// TGA (Truevision Targa) has no magic number. The registered prefixes are the
// colour-map flag and image type of every kind the decoder reads:
// uncompressed and RLE colour-mapped (1, 9), truecolor (2, 10), and
// greyscale (3, 11).
func init() {
	for _, m := range []string{
		"?\x01\x01", "?\x01\x09",
		"?\x00\x02", "?\x00\x0a",
		"?\x00\x03", "?\x00\x0b",
	} {
		image.RegisterFormat("tga", m, DecodeTGA, DecodeTGAConfig)
	}
}

type tgaHeader struct {
	idLen     uint8
	cmapType  uint8
	imgType   uint8
	cmapFirst uint16
	cmapLen   uint16
	cmapDepth uint8
	w, h      int
	depth     uint8
	desc      uint8
}

func readTGAHeader(r io.Reader) (tgaHeader, error) {
	var b [18]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return tgaHeader{}, fmt.Errorf("tga: %w", err)
	}
	h := tgaHeader{
		idLen:     b[0],
		cmapType:  b[1],
		imgType:   b[2],
		cmapFirst: binary.LittleEndian.Uint16(b[3:5]),
		cmapLen:   binary.LittleEndian.Uint16(b[5:7]),
		cmapDepth: b[7],
		w:         int(binary.LittleEndian.Uint16(b[12:14])),
		h:         int(binary.LittleEndian.Uint16(b[14:16])),
		depth:     b[16],
		desc:      b[17],
	}
	if h.w == 0 || h.h == 0 {
		return h, errors.New("tga: empty image")
	}
	switch h.imgType &^ 8 {
	case 1:
		if h.cmapType != 1 || (h.depth != 8 && h.depth != 16) {
			return h, errors.New("tga: colour-mapped image needs an 8- or 16-bit index")
		}
	case 2:
		if h.depth != 15 && h.depth != 16 && h.depth != 24 && h.depth != 32 {
			return h, fmt.Errorf("tga: unsupported truecolor depth %d", h.depth)
		}
	case 3:
		if h.depth != 8 && h.depth != 16 {
			return h, fmt.Errorf("tga: unsupported greyscale depth %d", h.depth)
		}
	default:
		return h, fmt.Errorf("tga: unsupported image type %d", h.imgType)
	}
	return h, nil
}

// DecodeTGAConfig reads the size of a TGA image.
func DecodeTGAConfig(r io.Reader) (image.Config, error) {
	h, err := readTGAHeader(r)
	if err != nil {
		return image.Config{}, err
	}
	return image.Config{ColorModel: color.NRGBAModel, Width: h.w, Height: h.h}, nil
}

// DecodeTGA reads uncompressed and RLE TGA images: 15/16/24/32-bit truecolor,
// 8-bit greyscale (16-bit with alpha), and 8/16-bit colour-mapped. The origin
// bits in the descriptor are honoured, so bottom-up files come out upright.
func DecodeTGA(r io.Reader) (image.Image, error) {
	br := bufio.NewReader(r)
	h, err := readTGAHeader(br)
	if err != nil {
		return nil, err
	}
	if _, err := br.Discard(int(h.idLen)); err != nil {
		return nil, fmt.Errorf("tga: %w", err)
	}
	attrBits := h.desc & 0x0f
	var palette []color.NRGBA
	if h.cmapType == 1 {
		eb := (int(h.cmapDepth) + 7) / 8
		raw := make([]byte, int(h.cmapLen)*eb)
		if _, err := io.ReadFull(br, raw); err != nil {
			return nil, fmt.Errorf("tga: colour map: %w", err)
		}
		if h.imgType&^8 == 1 {
			palette = make([]color.NRGBA, h.cmapLen)
			for i := range palette {
				palette[i] = tgaTrueColor(raw[i*eb:(i+1)*eb], h.cmapDepth, attrBits)
			}
		}
	}
	bpp := (int(h.depth) + 7) / 8
	n := h.w * h.h
	pix := make([]byte, n*bpp)
	if h.imgType&8 != 0 {
		if err := tgaUnRLE(br, pix, bpp); err != nil {
			return nil, err
		}
	} else if _, err := io.ReadFull(br, pix); err != nil {
		return nil, fmt.Errorf("tga: pixels: %w", err)
	}

	img := image.NewNRGBA(image.Rect(0, 0, h.w, h.h))
	topDown := h.desc&0x20 != 0
	rightToLeft := h.desc&0x10 != 0
	anyAlpha := false
	for i := 0; i < n; i++ {
		px := pix[i*bpp : (i+1)*bpp]
		var c color.NRGBA
		switch h.imgType &^ 8 {
		case 1:
			idx := int(px[0])
			if bpp == 2 {
				idx |= int(px[1]) << 8
			}
			idx -= int(h.cmapFirst)
			if idx >= 0 && idx < len(palette) {
				c = palette[idx]
			} else {
				c = color.NRGBA{A: 255}
			}
		case 2:
			c = tgaTrueColor(px, h.depth, attrBits)
		case 3:
			c = color.NRGBA{px[0], px[0], px[0], 255}
			if bpp == 2 {
				c.A = px[1]
			}
		}
		if c.A != 0 {
			anyAlpha = true
		}
		x, y := i%h.w, i/h.w
		if !topDown {
			y = h.h - 1 - y
		}
		if rightToLeft {
			x = h.w - 1 - x
		}
		o := img.PixOffset(x, y)
		img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = c.R, c.G, c.B, c.A
	}
	// Many writers put zeros in the alpha byte of 32-bit files they mean as
	// opaque. A fully transparent image is never what they meant.
	if !anyAlpha {
		for i := 3; i < len(img.Pix); i += 4 {
			img.Pix[i] = 255
		}
	}
	return img, nil
}

// tgaTrueColor reads one BGR(A) pixel of 15, 16, 24, or 32 bits.
func tgaTrueColor(px []byte, depth, attrBits uint8) color.NRGBA {
	switch depth {
	case 15, 16:
		v := uint16(px[0]) | uint16(px[1])<<8
		ex := func(b uint16) uint8 { return uint8(b<<3 | b>>2) }
		c := color.NRGBA{ex(v >> 10 & 31), ex(v >> 5 & 31), ex(v & 31), 255}
		if depth == 16 && attrBits == 1 && v&0x8000 == 0 {
			c.A = 0
		}
		return c
	case 24:
		return color.NRGBA{px[2], px[1], px[0], 255}
	case 32:
		return color.NRGBA{px[2], px[1], px[0], px[3]}
	}
	return color.NRGBA{A: 255}
}

// tgaUnRLE expands run-length packets: a header byte whose high bit marks a
// run of one repeated pixel, otherwise a count of literal pixels.
func tgaUnRLE(r *bufio.Reader, pix []byte, bpp int) error {
	o := 0
	for o < len(pix) {
		hdr, err := r.ReadByte()
		if err != nil {
			return fmt.Errorf("tga: rle: %w", err)
		}
		count := int(hdr&0x7f) + 1
		if o+count*bpp > len(pix) {
			count = (len(pix) - o) / bpp
		}
		if hdr&0x80 != 0 {
			var one [4]byte
			if _, err := io.ReadFull(r, one[:bpp]); err != nil {
				return fmt.Errorf("tga: rle: %w", err)
			}
			for i := 0; i < count; i++ {
				copy(pix[o:o+bpp], one[:bpp])
				o += bpp
			}
		} else {
			if _, err := io.ReadFull(r, pix[o:o+count*bpp]); err != nil {
				return fmt.Errorf("tga: rle: %w", err)
			}
			o += count * bpp
		}
	}
	return nil
}
