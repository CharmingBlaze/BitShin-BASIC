package runtime

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// captureShot writes the finished frame (scene, post, HUD) to the path
// SaveScreenshot queued. It runs on the GL thread before the swap.
func (w *World) captureShot(fbw, fbh int) {
	path := w.shotPath
	if path == "" || fbw < 1 || fbh < 1 {
		return
	}
	w.shotPath = ""
	bindColorTarget(0)
	pix := make([]uint8, fbw*fbh*4)
	gl.PixelStorei(gl.PACK_ALIGNMENT, 1)
	gl.ReadBuffer(gl.BACK)
	gl.ReadPixels(0, 0, int32(fbw), int32(fbh), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix))
	drainGL("SaveScreenshot")
	img := image.NewRGBA(image.Rect(0, 0, fbw, fbh))
	row := fbw * 4
	for y := 0; y < fbh; y++ {
		// GL rows start at the bottom.
		copy(img.Pix[y*row:(y+1)*row], pix[(fbh-1-y)*row:(fbh-y)*row])
	}
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	if dir := filepath.Dir(path); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Println("SaveScreenshot:", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Println("SaveScreenshot:", err)
	}
}

func (w *World) screenshotCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// SaveScreenshot(path$) saves the next presented frame as a PNG.
		"savescreenshot": n(func(a []value.Value) (value.Value, error) {
			p := argS(a, 0)
			if p == "" {
				p = "screenshot.png"
			}
			if !filepath.IsAbs(p) {
				p = w.resolve(p)
			}
			w.shotPath = p
			return z()
		}),
	}
}
