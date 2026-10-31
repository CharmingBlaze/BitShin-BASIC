package runtime

import (
	"image"
	"image/color"
	"math"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"moonbasic/internal/interp"
	"moonbasic/internal/value"
)

type ebiImage struct {
	img   *ebiten.Image
	src   image.Image
	glTex uint32
	mid   bool
	w, h  int
}

type ebiSprite struct {
	img                 *ebiten.Image
	id                  int
	x, y                float64
	rot, sx, sy         float64
	visible             bool
	fw, fh              int
	frame               int
	animFrom, animCount int
	animFPS, animT      float64
	flipX, flipY        bool
	alpha               float64
	cr, cg, cb          float64
	layer               int
	order               int     // draw order inside a layer
	pvx, pvy            float64 // pivot offset from the centre, in sprite sizes
	blend               int     // blendAlpha, blendAdd, blendMultiply, blendScreen
	parent              int     // sprite this one is positioned relative to
	ux, uy, uw, uh      int     // source rectangle; uw 0 uses the frame
	occluder            bool    // blocks 2D light
	unlit               bool    // drawn after 2D lighting, at full brightness
}

type drawOp struct {
	kind           int // 0 image, 1 rect, 2 oval, 3 line, 4 text
	img            *ebiten.Image
	imgID          int
	glTex          uint32
	texW, texH     int
	x, y, w, h     float32
	x2, y2         float32
	sx, sy, sw, sh int
	r, g, b, a     uint8
	text           string
	filled         bool
	world          bool // x, y are world coordinates seen through the camera
	tm             *tileMap
	layer          int
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
		g.w.prof.scriptBegin()
		err := g.in.Run()
		g.w.prof.scriptEnd()
		if err != nil {
			return err
		}
	}
	g.first = false
	g.w.tickFX()
	g.w.tickSpriteAnims()
	g.w.tickCameras2D()
	g.w.tileClock += g.w.delta
	if g.in.Done() || g.w.quit {
		return ebiten.Termination
	}
	return nil
}

func (g *ebiGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{g.w.clsRGB[0], g.w.clsRGB[1], g.w.clsRGB[2], 255})
	p := &g.w.prof
	p.frameStart()
	p.renderStart = time.Now()
	p.sampleMemory(g.w.loopFrames)
	g.w.draw2D(screen)
	p.rend = smooth(p.rend, ms(time.Since(p.renderStart)))
	g.w.profilerOverlay2D(screen)
	g.w.captureShot2D(screen)
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

// draw2D draws the frame. With only camera 0 it keeps the old order: the
// queued draws, HudPrint, particles, fog, then sprites. With more cameras,
// each viewport gets the world (tiles, particles, sprites) and the
// screen-space draws go on top once.
func (w *World) draw2D(screen *ebiten.Image) {
	views := w.views2D()
	lit := w.lighting2DOn()
	if !lit && len(views) == 1 && views[0].id == 0 && views[0].vw == 0 {
		c := views[0]
		w.drawOps2D(screen, c, true, true)
		w.drawHudPrint2D(screen)
		w.drawParticles2D(screen, c)
		w.drawFog2D(screen)
		w.drawSprites2D(screen, c, spritesAll)
		return
	}
	for _, c := range views {
		x, y, vw, vh := c.viewport(w)
		dst := screen.SubImage(rectXYWH(x, y, vw, vh)).(*ebiten.Image)
		w.drawOps2D(dst, c, true, false)
		w.drawParticles2D(dst, c)
		if lit {
			w.drawSprites2D(dst, c, spritesLit)
			w.renderLight2D(dst, c)
			w.drawSprites2D(dst, c, spritesUnlit)
			continue
		}
		w.drawSprites2D(dst, c, spritesAll)
	}
	w.drawOps2D(screen, nil, false, true)
	w.drawHudPrint2D(screen)
	w.drawFog2D(screen)
}

// drawOps2D draws the queued draws in order: world ones through camera c,
// screen ones as given.
func (w *World) drawOps2D(dst *ebiten.Image, c *camera2D, world, scr bool) {
	var cm ebiten.GeoM
	var x0, y0, x1, y1 float64
	if c != nil {
		cm = c.geoM(w)
		x0, y0, x1, y1 = c.worldRect(w)
	}
	for _, op := range w.draws {
		if op.world && !world || !op.world && !scr {
			continue
		}
		switch op.kind {
		case 0:
			if op.img == nil {
				continue
			}
			if op.world && (float64(op.x+op.w) < x0 || float64(op.x) > x1 || float64(op.y+op.h) < y0 || float64(op.y) > y1) {
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
			if op.world {
				opt.GeoM.Concat(cm)
			}
			dst.DrawImage(src, opt)
		case 1:
			if op.world {
				// A world rect turns and zooms with the camera: a scaled white pixel.
				if op.filled {
					opt := &ebiten.DrawImageOptions{}
					opt.GeoM.Scale(float64(op.w), float64(op.h))
					opt.GeoM.Translate(float64(op.x), float64(op.y))
					opt.GeoM.Concat(cm)
					opt.ColorScale.Scale(float32(op.r)/255, float32(op.g)/255, float32(op.b)/255, 1)
					dst.DrawImage(w.whitePixel(), opt)
				}
				continue
			}
			c := color.RGBA{op.r, op.g, op.b, 255}
			if op.filled {
				vector.FillRect(dst, op.x, op.y, op.w, op.h, c, true)
			} else {
				vector.StrokeRect(dst, op.x, op.y, op.w, op.h, 1, c, true)
			}
		case 2:
			c := color.RGBA{op.r, op.g, op.b, 255}
			vector.FillCircle(dst, op.x+op.w/2, op.y+op.h/2, op.w/2, c, true)
		case 3:
			c := color.RGBA{op.r, op.g, op.b, 255}
			vector.StrokeLine(dst, op.x, op.y, op.x2, op.y2, 1, c, true)
		case 5:
			if c != nil {
				w.drawTileLayer(dst, c, op.tm, op.tm.layer(op.layer), float64(op.x), float64(op.y))
			}
		case 4:
			if !w.drawFont(dst, op) {
				ebitenutil.DebugPrintAt(dst, op.text, int(op.x), int(op.y))
			}
		}
	}
}

const (
	spritesAll = iota
	spritesLit
	spritesUnlit
)

func (w *World) drawSprites2D(dst *ebiten.Image, c *camera2D, which int) {
	list := make([]*ebiSprite, 0, len(w.sprites))
	for _, s := range w.sprites {
		if s == nil || !s.visible || s.img == nil {
			continue
		}
		if which == spritesLit && s.unlit || which == spritesUnlit && !s.unlit {
			continue
		}
		list = append(list, s)
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].layer != list[j].layer {
			return list[i].layer < list[j].layer
		}
		if list[i].order != list[j].order {
			return list[i].order < list[j].order
		}
		return list[i].id < list[j].id
	})
	cm := c.geoM(w)
	x0, y0, x1, y1 := c.worldRect(w)
	for _, s := range list {
		src, bw, bh := s.frameImage()
		x, y, rot, sx, sy := w.spriteWorld(s, 0)
		if r := float64(bw)*math.Abs(sx) + float64(bh)*math.Abs(sy); x+r < x0 || x-r > x1 || y+r < y0 || y-r > y1 {
			continue
		}
		opt := &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(-float64(bw)*(0.5+s.pvx), -float64(bh)*(0.5+s.pvy))
		if s.flipX {
			sx = -sx
		}
		if s.flipY {
			sy = -sy
		}
		opt.GeoM.Scale(sx, sy)
		opt.GeoM.Rotate(rot * math.Pi / 180)
		opt.GeoM.Translate(x, y)
		opt.GeoM.Concat(cm)
		opt.ColorScale.Scale(float32(s.cr), float32(s.cg), float32(s.cb), float32(s.alpha))
		opt.Blend = spriteBlend(s.blend)
		dst.DrawImage(src, opt)
	}
}

func (w *World) tickSpriteAnims() {
	dt := w.delta
	if dt <= 0 {
		return
	}
	for _, s := range w.sprites {
		if s == nil || s.animCount <= 1 || s.animFPS <= 0 {
			continue
		}
		s.animT += dt
		step := 1 / s.animFPS
		for s.animT >= step {
			s.animT -= step
			s.frame++
			if s.frame < s.animFrom || s.frame >= s.animFrom+s.animCount {
				s.frame = s.animFrom
			}
		}
	}
}

func rectXYWH(x, y, w, h int) image.Rectangle {
	return image.Rect(x, y, x+w, y+h)
}

func (w *World) drawHudPrint2D(screen *ebiten.Image) {
	if len(w.hudLines) == 0 {
		return
	}
	lineH := float32(16)
	if f := w.fonts[w.curFont]; f != nil && f.size > 0 {
		lineH = float32(f.size) + 4
	}
	for i := 0; i < len(w.hudLines); i++ {
		op := drawOp{kind: 4, x: 0, y: float32(i) * lineH, text: w.hudLines[i]}
		if !w.drawFont(screen, op) {
			ebitenutil.DebugPrintAt(screen, w.hudLines[i], 0, int(op.y))
		}
	}
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
