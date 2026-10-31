package runtime

import (
	"fmt"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"moonbasic/internal/value"
)

const (
	blendAlpha = iota
	blendAdd
	blendMultiply
	blendScreen
)

var (
	// Source is premultiplied: multiply is src*dst + dst*(1-srcA), screen
	// is src + dst*(1-src).
	blendMul = ebiten.Blend{
		BlendFactorSourceRGB:        ebiten.BlendFactorDestinationColor,
		BlendFactorSourceAlpha:      ebiten.BlendFactorDestinationAlpha,
		BlendFactorDestinationRGB:   ebiten.BlendFactorOneMinusSourceAlpha,
		BlendFactorDestinationAlpha: ebiten.BlendFactorOneMinusSourceAlpha,
		BlendOperationRGB:           ebiten.BlendOperationAdd,
		BlendOperationAlpha:         ebiten.BlendOperationAdd,
	}
	blendScr = ebiten.Blend{
		BlendFactorSourceRGB:        ebiten.BlendFactorOne,
		BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
		BlendFactorDestinationRGB:   ebiten.BlendFactorOneMinusSourceColor,
		BlendFactorDestinationAlpha: ebiten.BlendFactorOneMinusSourceAlpha,
		BlendOperationRGB:           ebiten.BlendOperationAdd,
		BlendOperationAlpha:         ebiten.BlendOperationAdd,
	}
)

func spriteBlend(mode int) ebiten.Blend {
	switch mode {
	case blendAdd:
		return ebiten.BlendLighter
	case blendMultiply:
		return blendMul
	case blendScreen:
		return blendScr
	}
	return ebiten.BlendSourceOver
}

func blendMode(name string, n int) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "alpha", "normal", "0":
		return blendAlpha, true
	case "add", "additive", "light", "1":
		return blendAdd, true
	case "multiply", "mul", "2":
		return blendMultiply, true
	case "screen", "3":
		return blendScreen, true
	case "":
		if n >= blendAlpha && n <= blendScreen {
			return n, true
		}
	}
	return 0, false
}

// frameImage is the part of the sprite's image it draws: its source
// rectangle, else its sheet frame, else the whole image.
func (s *ebiSprite) frameImage() (*ebiten.Image, int, int) {
	src := s.img
	bw, bh := src.Bounds().Dx(), src.Bounds().Dy()
	if s.uw > 0 && s.uh > 0 {
		return src.SubImage(rectXYWH(s.ux, s.uy, s.uw, s.uh)).(*ebiten.Image), s.uw, s.uh
	}
	if s.fw > 0 && s.fh > 0 {
		cols := bw / s.fw
		if cols < 1 {
			cols = 1
		}
		fr := s.frame
		if fr < 0 {
			fr = 0
		}
		sx, sy := (fr%cols)*s.fw, (fr/cols)*s.fh
		if sx+s.fw <= bw && sy+s.fh <= bh {
			return src.SubImage(rectXYWH(sx, sy, s.fw, s.fh)).(*ebiten.Image), s.fw, s.fh
		}
	}
	return src, bw, bh
}

// spriteWorld is the sprite's position, rotation, and scale in the world.
// A child's position is in its parent's space: it moves, turns, and scales
// with the parent.
func (w *World) spriteWorld(s *ebiSprite, depth int) (x, y, rot, sx, sy float64) {
	sx, sy = s.sx, s.sy
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	p := w.sprites[s.parent]
	if s.parent == 0 || p == nil || p == s || depth > 16 {
		return s.x, s.y, s.rot, sx, sy
	}
	px, py, prot, psx, psy := w.spriteWorld(p, depth+1)
	lx, ly := s.x*psx, s.y*psy
	sin, cos := math.Sincos(prot * math.Pi / 180)
	return px + lx*cos - ly*sin, py + lx*sin + ly*cos, prot + s.rot, sx * psx, sy * psy
}

func (w *World) whitePixel() *ebiten.Image {
	if w.white2D == nil {
		w.white2D = w.solidImage(1, 1, 255, 255, 255)
	}
	return w.white2D
}

func (w *World) sprite2DCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	sprite := func(a []value.Value, where string) (*ebiSprite, error) {
		s := w.sprites[argI(a, 0, 0)]
		if s == nil {
			return nil, fmt.Errorf("%s: %d is not a sprite", where, argI(a, 0, 0))
		}
		return s, nil
	}
	set := func(where string, fn func(s *ebiSprite, a []value.Value) error) cmd {
		return n(func(a []value.Value) (value.Value, error) {
			s, err := sprite(a, where)
			if err != nil {
				return value.Value{}, err
			}
			if err := fn(s, a); err != nil {
				return value.Value{}, err
			}
			return z()
		})
	}
	return map[string]cmd{
		// SetSpritePivot s, px, py: 0..1 across the sprite; 0.5, 0.5 is the
		// centre (the default), 0.5, 1 the feet.
		"setspritepivot": set("SetSpritePivot", func(s *ebiSprite, a []value.Value) error {
			s.pvx, s.pvy = argN(a, 1, 0.5)-0.5, argN(a, 2, 0.5)-0.5
			return nil
		}),
		// SetSpriteBlend s, "alpha" | "add" | "multiply" | "screen"
		"setspriteblend": set("SetSpriteBlend", func(s *ebiSprite, a []value.Value) error {
			m, ok := blendMode(argS(a, 1), argI(a, 1, 0))
			if !ok {
				return fmt.Errorf("SetSpriteBlend: want alpha, add, multiply, or screen, got %q", argS(a, 1))
			}
			s.blend = m
			return nil
		}),
		// SetSpriteParent child, parent [, keepWorld]: 0 detaches. With
		// keepWorld the child stays where it is on screen.
		"setspriteparent": set("SetSpriteParent", func(s *ebiSprite, a []value.Value) error {
			pid := argI(a, 1, 0)
			for p, depth := w.sprites[pid], 0; p != nil && depth < 32; p, depth = w.sprites[p.parent], depth+1 {
				if p == s {
					return fmt.Errorf("SetSpriteParent: %d is already above %d", s.id, pid)
				}
				if p.parent == 0 {
					break
				}
			}
			if argI(a, 2, 0) != 0 {
				x, y, rot, sx, sy := w.spriteWorld(s, 0)
				s.parent = 0
				if p := w.sprites[pid]; p != nil {
					px, py, prot, psx, psy := w.spriteWorld(p, 0)
					sin, cos := math.Sincos(-prot * math.Pi / 180)
					dx, dy := x-px, y-py
					x, y = (dx*cos-dy*sin)/psx, (dx*sin+dy*cos)/psy
					rot -= prot
					sx, sy = sx/psx, sy/psy
				}
				s.x, s.y, s.rot, s.sx, s.sy = x, y, rot, sx, sy
			}
			if w.sprites[pid] == nil {
				pid = 0
			}
			s.parent = pid
			return nil
		}),
		// SetSpriteRect s, x, y, w, h draws only that part of the image;
		// SetSpriteRect s alone goes back to the whole image or sheet frame.
		"setspriterect": set("SetSpriteRect", func(s *ebiSprite, a []value.Value) error {
			if len(a) < 5 {
				s.uw, s.uh = 0, 0
				return nil
			}
			s.ux, s.uy, s.uw, s.uh = argI(a, 1, 0), argI(a, 2, 0), argI(a, 3, 0), argI(a, 4, 0)
			return nil
		}),
		// SetSpriteOrder s, n: draw order inside the sprite's layer.
		"setspriteorder": set("SetSpriteOrder", func(s *ebiSprite, a []value.Value) error {
			s.order = argI(a, 1, 0)
			return nil
		}),
		"spriteworldx": n(func(a []value.Value) (value.Value, error) {
			s, err := sprite(a, "SpriteWorldX")
			if err != nil {
				return value.Value{}, err
			}
			x, _, _, _, _ := w.spriteWorld(s, 0)
			return value.Num(x), nil
		}),
		"spriteworldy": n(func(a []value.Value) (value.Value, error) {
			s, err := sprite(a, "SpriteWorldY")
			if err != nil {
				return value.Value{}, err
			}
			_, y, _, _, _ := w.spriteWorld(s, 0)
			return value.Num(y), nil
		}),
	}
}
