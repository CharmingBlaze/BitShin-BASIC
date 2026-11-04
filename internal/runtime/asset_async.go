package runtime

import (
	"fmt"
	"image"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/g3n/engine/texture"

	"moonbasic/internal/value"
)

// Textures in the background. Decoding a large image is what stalls a game when it
// loads one mid-play, and it needs no GL, so it runs on a worker. The texture exists
// at once, as a grey pixel, and takes its picture on a later frame. Uploads happen one
// a frame on the GL thread, so several finishing together do not stall it either.

type asyncTexture struct {
	id   int
	file string
	img  *image.RGBA
	err  error
}

type assetLoader struct {
	done    chan asyncTexture
	slots   chan struct{}
	pending map[int]bool
	failed  map[int]bool
}

func (w *World) ensureLoader() *assetLoader {
	if w.assets == nil {
		n := runtime.NumCPU() - 1
		if n < 1 {
			n = 1
		}
		if n > 4 {
			n = 4
		}
		w.assets = &assetLoader{
			done:    make(chan asyncTexture, 256),
			slots:   make(chan struct{}, n),
			pending: map[int]bool{},
			failed:  map[int]bool{},
		}
	}
	return w.assets
}

func (w *World) loadTextureAsync(file string) (int, error) {
	path, err := w.openPath(file)
	if err != nil {
		return 0, err
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".dds", ".ktx", ".ktx2":
		return w.loadTextureFile(file) // already GPU data: little to decode
	}
	grey := image.NewRGBA(image.Rect(0, 0, 1, 1))
	grey.Pix[0], grey.Pix[1], grey.Pix[2], grey.Pix[3] = 128, 128, 128, 255
	id := w.nextTex
	w.nextTex++
	if w.texs == nil {
		w.texs = map[int]*texSlot{}
	}
	w.texs[id] = &texSlot{tex: texture.NewTexture2DFromRGBA(grey), path: file}
	l := w.ensureLoader()
	l.pending[id] = true
	limit := w.textureLimit()
	go func() {
		l.slots <- struct{}{}
		img, err := texture.DecodeImage(path)
		if err == nil {
			img = shrinkToLimit(img, limit)
		}
		<-l.slots
		l.done <- asyncTexture{id: id, file: file, img: img, err: err}
	}()
	return id, nil
}

// tickAssets takes one finished texture each frame.
func (w *World) tickAssets() {
	l := w.assets
	if l == nil || len(l.pending) == 0 {
		return
	}
	select {
	case r := <-l.done:
		delete(l.pending, r.id)
		slot := w.texs[r.id]
		if r.err != nil {
			l.failed[r.id] = true
			fmt.Printf("LoadTextureAsync: %s: %v\n", r.file, r.err)
			return
		}
		if slot == nil || slot.tex == nil {
			return // freed while it was loading
		}
		slot.tex.SetFromRGBA(r.img)
		b := r.img.Bounds()
		w.noteVRAM(b.Dx(), b.Dy(), 0)
	default:
	}
}

// textureLimit is the largest side a loaded texture keeps. Old GPUs share a little
// memory with everything else, and a 4K texture costs them more than it shows at 720p.
func (w *World) textureLimit() int {
	if w.texLimitSet {
		return w.texLimit
	}
	if !w.quality.set {
		return 0
	}
	return tiers[w.quality.tier].maxTexture
}

// shrinkToLimit halves an image, averaging, until it fits. Halving keeps it sharp
// and cheap, and it leaves images within the limit untouched.
func shrinkToLimit(img *image.RGBA, limit int) *image.RGBA {
	if limit <= 0 || img == nil {
		return img
	}
	for {
		b := img.Bounds()
		if (b.Dx() <= limit && b.Dy() <= limit) || b.Dx() < 2 || b.Dy() < 2 {
			return img
		}
		img = halveImage(img)
	}
}

func halveImage(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx()/2, b.Dy()/2
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := src.PixOffset(b.Min.X+x*2, b.Min.Y+y*2)
			c := src.PixOffset(b.Min.X+x*2, b.Min.Y+y*2+1)
			o := dst.PixOffset(x, y)
			for i := 0; i < 4; i++ {
				sum := int(src.Pix[a+i]) + int(src.Pix[a+4+i]) + int(src.Pix[c+i]) + int(src.Pix[c+4+i])
				dst.Pix[o+i] = uint8((sum + 2) / 4)
			}
		}
	}
	return dst
}

func (w *World) assetCommands(n func(func([]value.Value) (value.Value, error)) cmd, need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		// LoadTextureAsync(file$): like LoadTexture, without the wait. The texture
		// can be used at once; it shows grey until its picture arrives.
		"loadtextureasync": need(func(a []value.Value) (value.Value, error) {
			id, err := w.loadTextureAsync(argS(a, 0))
			if err != nil {
				fmt.Println("LoadTextureAsync:", err)
				return value.Num(0), nil
			}
			return value.Num(float64(id)), nil
		}),
		// TextureReady(tex): 1 once the picture is in, 0 while it loads, -1 if it failed.
		"textureready": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if l := w.assets; l != nil {
				if l.pending[id] {
					return value.Num(0), nil
				}
				if l.failed[id] {
					return value.Num(-1), nil
				}
			}
			return value.Num(float64(b01(w.texs[id] != nil))), nil
		}),
		// TexturesLoading(): how many are still on their way, for a loading screen.
		"texturesloading": n(func(a []value.Value) (value.Value, error) {
			if w.assets == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(len(w.assets.pending))), nil
		}),
		// SetTextureLimit(pixels): the largest side a texture keeps when loaded, over
		// what the quality tier would choose. 0 is no limit.
		"settexturelimit": n(func(a []value.Value) (value.Value, error) {
			w.texLimit = argI(a, 0, 0)
			w.texLimitSet = true
			return value.Num(float64(w.texLimit)), nil
		}),
	}
}
