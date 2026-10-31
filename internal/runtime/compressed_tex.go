package runtime

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"path/filepath"
	"strings"

	"github.com/g3n/engine/texture"
	"github.com/klauspost/compress/zstd"

	_ "moonbasic/internal/imgfmt" // BMP, TGA, WebP for every image.Decode
)

// Desktop GL 3.3 uploads these through EXT_texture_compression_s3tc.
// KTX2 Zstandard (scheme 2) and ZLIB (scheme 3) are inflated back to BC1–BC3.
// BasisLZ (scheme 1) is a different codec and is not transcoded.
const (
	glCompressedRGBS3TCDXT1  int32 = 0x83F0
	glCompressedRGBAS3TCDXT1 int32 = 0x83F1
	glCompressedRGBAS3TCDXT3 int32 = 0x83F2
	glCompressedRGBAS3TCDXT5 int32 = 0x83F3
)

type gpuImage struct {
	w, h       int
	compressed bool
	format     int32
	data       []byte
	rgba       *image.RGBA
}

func (w *World) loadTextureFile(file string) (int, error) {
	path, err := w.openPath(file)
	if err != nil {
		return 0, err
	}
	var tex *texture.Texture2D
	switch strings.ToLower(filepath.Ext(path)) {
	case ".dds", ".ktx", ".ktx2":
		b, err := readFileBytes(path)
		if err != nil {
			return 0, err
		}
		img, err := parseGPUImage(b)
		if err != nil {
			return 0, err
		}
		if img.compressed {
			tex = texture.NewTexture2DFromCompressedData(img.w, img.h, img.format, int32(len(img.data)), img.data)
		} else if img.rgba != nil {
			tex = texture.NewTexture2DFromRGBA(img.rgba)
		}
	default:
		var err error
		tex, err = texture.NewTexture2DFromImage(path)
		if err != nil {
			return 0, err
		}
	}
	if tex == nil {
		return 0, fmt.Errorf("texture: empty image")
	}
	id := w.nextTex
	w.nextTex++
	if w.texs == nil {
		w.texs = map[int]*texSlot{}
	}
	w.texs[id] = &texSlot{tex: tex, path: file}
	return id, nil
}

func parseGPUImage(b []byte) (gpuImage, error) {
	if len(b) >= 4 && string(b[:4]) == "DDS " {
		return parseDDS(b)
	}
	if len(b) >= 12 && string(b[1:7]) == "KTX 11" {
		return parseKTX1(b)
	}
	if len(b) >= 12 && string(b[1:7]) == "KTX 20" {
		return parseKTX2(b)
	}
	return gpuImage{}, fmt.Errorf("texture: not a DDS or KTX image")
}

func parseDDS(b []byte) (gpuImage, error) {
	if len(b) < 128 {
		return gpuImage{}, fmt.Errorf("dds: truncated header")
	}
	h := int(binary.LittleEndian.Uint32(b[12:16]))
	w := int(binary.LittleEndian.Uint32(b[16:20]))
	if w <= 0 || h <= 0 {
		return gpuImage{}, fmt.Errorf("dds: bad size")
	}
	fourcc := string(b[84:88])
	flags := binary.LittleEndian.Uint32(b[80:84])
	off := 128
	switch fourcc {
	case "DXT1":
		return dxtImage(b, off, w, h, 8, glCompressedRGBS3TCDXT1)
	case "DXT3":
		return dxtImage(b, off, w, h, 16, glCompressedRGBAS3TCDXT3)
	case "DXT5":
		return dxtImage(b, off, w, h, 16, glCompressedRGBAS3TCDXT5)
	case "DX10":
		return gpuImage{}, fmt.Errorf("dds: DX10 header is not loaded (use DXT1, DXT3, or DXT5)")
	}
	// Uncompressed 32-bit BGRA.
	if flags&0x40 == 0 {
		return gpuImage{}, fmt.Errorf("dds: unsupported format %q", fourcc)
	}
	bitCount := binary.LittleEndian.Uint32(b[88:92])
	if bitCount != 32 {
		return gpuImage{}, fmt.Errorf("dds: need 32-bit uncompressed or DXT")
	}
	n := w * h * 4
	if off+n > len(b) {
		return gpuImage{}, fmt.Errorf("dds: truncated pixels")
	}
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	src := b[off : off+n]
	for i := 0; i < n; i += 4 {
		rgba.Pix[i] = src[i+2]
		rgba.Pix[i+1] = src[i+1]
		rgba.Pix[i+2] = src[i]
		rgba.Pix[i+3] = src[i+3]
	}
	return gpuImage{w: w, h: h, rgba: rgba}, nil
}

func dxtImage(b []byte, off, w, h, block int, format int32) (gpuImage, error) {
	n, err := dxtSize(w, h, block)
	if err != nil {
		return gpuImage{}, err
	}
	if off+n > len(b) {
		return gpuImage{}, fmt.Errorf("dds: truncated block data")
	}
	return gpuImage{w: w, h: h, compressed: true, format: format, data: append([]byte(nil), b[off:off+n]...)}, nil
}

func dxtSize(w, h, block int) (int, error) {
	if w <= 0 || h <= 0 {
		return 0, fmt.Errorf("dds: bad size")
	}
	bw := (w + 3) / 4
	bh := (h + 3) / 4
	return bw * bh * block, nil
}

func parseKTX1(b []byte) (gpuImage, error) {
	if len(b) < 64 {
		return gpuImage{}, fmt.Errorf("ktx: truncated header")
	}
	if binary.LittleEndian.Uint32(b[12:16]) != 0x04030201 {
		return gpuImage{}, fmt.Errorf("ktx: big-endian files are not loaded")
	}
	glType := binary.LittleEndian.Uint32(b[16:20])
	internal := int32(binary.LittleEndian.Uint32(b[28:32]))
	w := int(binary.LittleEndian.Uint32(b[36:40]))
	h := int(binary.LittleEndian.Uint32(b[40:44]))
	kv := int(binary.LittleEndian.Uint32(b[60:64]))
	off := 64 + kv
	if off+4 > len(b) || w <= 0 || h <= 0 {
		return gpuImage{}, fmt.Errorf("ktx: bad header")
	}
	size := int(binary.LittleEndian.Uint32(b[off:]))
	off += 4
	if off+size > len(b) {
		return gpuImage{}, fmt.Errorf("ktx: truncated image")
	}
	payload := b[off : off+size]
	if glType == 0 {
		format, err := s3tcFormat(internal)
		if err != nil {
			return gpuImage{}, err
		}
		return gpuImage{w: w, h: h, compressed: true, format: format, data: append([]byte(nil), payload...)}, nil
	}
	return gpuImage{}, fmt.Errorf("ktx: uncompressed KTX is not loaded; use PNG, JPEG, or S3TC")
}

func parseKTX2(b []byte) (gpuImage, error) {
	// Header is 80 bytes, then levelCount index entries of 24 bytes.
	if len(b) < 80 {
		return gpuImage{}, fmt.Errorf("ktx2: truncated header")
	}
	vk := binary.LittleEndian.Uint32(b[12:16])
	w := int(binary.LittleEndian.Uint32(b[20:24]))
	h := int(binary.LittleEndian.Uint32(b[24:28]))
	levels := int(binary.LittleEndian.Uint32(b[40:44]))
	super := binary.LittleEndian.Uint32(b[44:48])
	if levels <= 0 || w <= 0 || h <= 0 {
		return gpuImage{}, fmt.Errorf("ktx2: bad header")
	}
	format, err := vkToS3TC(vk)
	if err != nil {
		return gpuImage{}, err
	}
	// The last level index is mip 0 (largest).
	entry := 80 + (levels-1)*24
	if entry+24 > len(b) {
		return gpuImage{}, fmt.Errorf("ktx2: truncated level index")
	}
	off := int(binary.LittleEndian.Uint64(b[entry:]))
	size := int(binary.LittleEndian.Uint64(b[entry+8:]))
	uncomp := int(binary.LittleEndian.Uint64(b[entry+16:]))
	if off < 0 || size <= 0 || off+size > len(b) {
		return gpuImage{}, fmt.Errorf("ktx2: bad level")
	}
	payload := b[off : off+size]
	switch super {
	case 0:
	case 1:
		return gpuImage{}, fmt.Errorf("ktx2: BasisLZ supercompression is not transcoded")
	case 2:
		payload, err = decodeZstd(payload, uncomp)
	case 3:
		payload, err = decodeZlib(payload)
	default:
		err = fmt.Errorf("ktx2: supercompression scheme %d is not decoded", super)
	}
	if err != nil {
		return gpuImage{}, err
	}
	return gpuImage{w: w, h: h, compressed: true, format: format, data: append([]byte(nil), payload...)}, nil
}

func decodeZlib(b []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("ktx2: zlib: %w", err)
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("ktx2: zlib: %w", err)
	}
	return out, nil
}

func decodeZstd(b []byte, want int) ([]byte, error) {
	r, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("ktx2: zstd: %w", err)
	}
	defer r.Close()
	out, err := r.DecodeAll(b, nil)
	if err != nil {
		return nil, fmt.Errorf("ktx2: zstd: %w", err)
	}
	if want > 0 && len(out) != want {
		return nil, fmt.Errorf("ktx2: zstd size %d, want %d", len(out), want)
	}
	return out, nil
}

func s3tcFormat(internal int32) (int32, error) {
	switch internal {
	case glCompressedRGBS3TCDXT1, glCompressedRGBAS3TCDXT1, glCompressedRGBAS3TCDXT3, glCompressedRGBAS3TCDXT5:
		return internal, nil
	default:
		return 0, fmt.Errorf("ktx: internal format 0x%X is not S3TC", uint32(internal))
	}
}

func vkToS3TC(vk uint32) (int32, error) {
	switch vk {
	case 131, 132: // BC1 RGB
		return glCompressedRGBS3TCDXT1, nil
	case 133, 134: // BC1 RGBA
		return glCompressedRGBAS3TCDXT1, nil
	case 135, 136: // BC2
		return glCompressedRGBAS3TCDXT3, nil
	case 137, 138: // BC3
		return glCompressedRGBAS3TCDXT5, nil
	default:
		return 0, fmt.Errorf("ktx2: vkFormat %d is not BC1/BC2/BC3", vk)
	}
}
