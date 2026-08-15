package runtime

import (
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"bitshinbasic/internal/interp"
	"bitshinbasic/internal/value"
)

type ebiImage struct {
	img    *ebiten.Image
	mid    bool
	w, h   int
}

type ebiSprite struct {
	img           *ebiten.Image
	x, y          float64
	rot, sx, sy   float64
	visible       bool
}

type drawOp struct {
	kind       int // 0 image, 1 rect, 2 oval, 3 line, 4 text
	img        *ebiten.Image
	x, y, w, h float32
	x2, y2     float32
	sx, sy, sw, sh int
	r, g, b, a uint8
	text       string
	filled     bool
}

var ebiKeys = map[int]ebiten.Key{
	KeyEscape: ebiten.KeyEscape, KeySpace: ebiten.KeySpace, KeyEnter: ebiten.KeyEnter,
	KeyTab: ebiten.KeyTab, KeyBackspace: ebiten.KeyBackspace,
	KeyA: ebiten.KeyA, KeyB: ebiten.KeyB, KeyC: ebiten.KeyC, KeyD: ebiten.KeyD,
	KeyE: ebiten.KeyE, KeyF: ebiten.KeyF, KeyG: ebiten.KeyG, KeyH: ebiten.KeyH,
	KeyI: ebiten.KeyI, KeyJ: ebiten.KeyJ, KeyK: ebiten.KeyK, KeyL: ebiten.KeyL,
	KeyM: ebiten.KeyM, KeyN: ebiten.KeyN, KeyO: ebiten.KeyO, KeyP: ebiten.KeyP,
	KeyQ: ebiten.KeyQ, KeyR: ebiten.KeyR, KeyS: ebiten.KeyS, KeyT: ebiten.KeyT,
	KeyU: ebiten.KeyU, KeyV: ebiten.KeyV, KeyW: ebiten.KeyW, KeyX: ebiten.KeyX,
	KeyY: ebiten.KeyY, KeyZ: ebiten.KeyZ,
	Key0: ebiten.Key0, Key1: ebiten.Key1, Key2: ebiten.Key2, Key3: ebiten.Key3,
	Key4: ebiten.Key4, Key5: ebiten.Key5, Key6: ebiten.Key6, Key7: ebiten.Key7,
	Key8: ebiten.Key8, Key9: ebiten.Key9,
	KeyUp: ebiten.KeyUp, KeyDown: ebiten.KeyDown, KeyLeft: ebiten.KeyLeft, KeyRight: ebiten.KeyRight,
	KeyLShift: ebiten.KeyShift, KeyRShift: ebiten.KeyShiftRight,
	KeyLControl: ebiten.KeyControl, KeyRControl: ebiten.KeyControlRight,
}

type ebiGame struct {
	w     *World
	in    *interp.Interp
	first bool
}

func (g *ebiGame) Update() error {
	g.w.pollEbitenInput()
	if g.w.presented && !g.w.presentOK {
		g.w.latchHeldKeys()
		g.w.drainPhantomQuit()
		if g.w.heldAtFlip[KeyEscape] || g.w.keys[KeyEscape] {
			g.w.escapeLatch = true
			g.w.clearHeldKey(KeyEscape)
			g.w.heldAtFlip[KeyEscape] = true
		}
		g.w.presentOK = true
	}
	g.w.noteMouse()
	g.w.loopFrames++
	tps := ebiten.ActualTPS()
	if tps > 1 {
		g.w.tickDelta(time.Duration(float64(time.Second) / tps))
	} else {
		g.w.tickDelta(0)
	}
	if !g.first {
		g.w.draws = g.w.draws[:0]
	}
	if g.in == nil {
		return ebiten.Termination
	}
	if !g.first {
		if err := g.in.Run(); err != nil {
			return err
		}
	}
	g.first = false
	g.w.tickFX()
	if g.in.Done() || g.w.quit {
		return ebiten.Termination
	}
	return nil
}

func (g *ebiGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{g.w.clsRGB[0], g.w.clsRGB[1], g.w.clsRGB[2], 255})
	g.w.draw2D(screen)
}

func (g *ebiGame) Layout(ow, oh int) (int, int) {
	if g.w.scrW <= 0 {
		return 800, 600
	}
	return g.w.scrW, g.w.scrH
}

func (w *World) graphics2D(width, height int) (value.Value, error) {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	w.scrW, w.scrH = width, height
	w.mode2D = true
	w.ready = true
	if w.runner != nil {
		w.runner.MarkLive()
	}
	return value.Num(0), nil
}

func (w *World) loop2D(in *interp.Interp) error {
	ebiten.SetWindowSize(w.scrW, w.scrH)
	ebiten.SetWindowTitle(w.title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	g := &ebiGame{w: w, in: in, first: true}
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		return err
	}
	if in != nil {
		return in.Err()
	}
	return nil
}

func (w *World) pollEbitenInput() {
	for code, k := range ebiKeys {
		if (code == KeyEscape || code == KeySpace) && !w.presented {
			w.keys[code] = false
			w.hits[code] = false
			continue
		}
		down := ebiten.IsKeyPressed(k)
		if w.applyHeldLatch(code, down) {
			continue
		}
		if down && !w.keys[code] {
			w.hits[code] = true
		}
		w.keys[code] = down
	}
	if !w.presented {
		w.keys[KeyEscape] = false
		w.hits[KeyEscape] = false
		w.keys[KeySpace] = false
		w.hits[KeySpace] = false
	} else if ebiten.IsKeyPressed(ebiten.KeyEscape) && !w.heldAtFlip[KeyEscape] && !w.escapeLatch {
		w.keys[KeyEscape] = true
	}
	cx, cy := ebiten.CursorPosition()
	w.mx, w.my = float32(cx), float32(cy)
	for i, btn := range []ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonRight, ebiten.MouseButtonMiddle} {
		down := ebiten.IsMouseButtonPressed(btn)
		if down && !w.mouse[i+1] {
			w.mouseHits[i+1] = true
		}
		w.mouse[i+1] = down
	}
	_, wy := ebiten.Wheel()
	w.mz += float32(wy)
	_ = inpututil.IsKeyJustPressed
}

func (w *World) draw2D(screen *ebiten.Image) {
	for _, op := range w.draws {
		switch op.kind {
		case 0:
			if op.img == nil {
				continue
			}
			src := op.img
			if op.sw > 0 && op.sh > 0 {
				src = op.img.SubImage(rectXYWH(op.sx, op.sy, op.sw, op.sh)).(*ebiten.Image)
			}
			opt := &ebiten.DrawImageOptions{}
			b := src.Bounds()
			if op.w > 0 && op.h > 0 && (int(op.w) != b.Dx() || int(op.h) != b.Dy()) {
				opt.GeoM.Scale(float64(op.w)/float64(b.Dx()), float64(op.h)/float64(b.Dy()))
			}
			opt.GeoM.Translate(float64(op.x), float64(op.y))
			screen.DrawImage(src, opt)
		case 1:
			c := color.RGBA{op.r, op.g, op.b, 255}
			if op.filled {
				vector.FillRect(screen, op.x, op.y, op.w, op.h, c, true)
			} else {
				vector.StrokeRect(screen, op.x, op.y, op.w, op.h, 1, c, true)
			}
		case 2:
			c := color.RGBA{op.r, op.g, op.b, 255}
			vector.FillCircle(screen, op.x+op.w/2, op.y+op.h/2, op.w/2, c, true)
		case 3:
			c := color.RGBA{op.r, op.g, op.b, 255}
			vector.StrokeLine(screen, op.x, op.y, op.x2, op.y2, 1, c, true)
		case 4:
			if !w.drawFont(screen, op) {
				ebitenutil.DebugPrintAt(screen, op.text, int(op.x), int(op.y))
			}
		}
	}
	w.drawParticles2D(screen)
	w.drawFog2D(screen)
	for _, s := range w.sprites {
		if s == nil || !s.visible || s.img == nil {
			continue
		}
		opt := &ebiten.DrawImageOptions{}
		b := s.img.Bounds()
		opt.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
		opt.GeoM.Scale(s.sx, s.sy)
		opt.GeoM.Rotate(s.rot * 3.141592653589793 / 180)
		opt.GeoM.Translate(s.x, s.y)
		screen.DrawImage(s.img, opt)
	}
}

func rectXYWH(x, y, w, h int) image.Rectangle {
	return image.Rect(x, y, x+w, y+h)
}

func (w *World) drawFont(screen *ebiten.Image, op drawOp) bool {
	f := w.fonts[w.curFont]
	if f == nil || f.src == nil {
		return false
	}
	face := &text.GoTextFace{Source: f.src, Size: f.size}
	opt := &text.DrawOptions{}
	opt.GeoM.Translate(float64(op.x), float64(op.y))
	opt.ColorScale.ScaleWithColor(color.RGBA{w.drawRGB[0], w.drawRGB[1], w.drawRGB[2], 255})
	text.Draw(screen, op.text, face, opt)
	return true
}

func (w *World) solidImage(ww, hh int, r, g, b uint8) *ebiten.Image {
	img := ebiten.NewImage(ww, hh)
	img.Fill(color.RGBA{r, g, b, 255})
	return img
}
