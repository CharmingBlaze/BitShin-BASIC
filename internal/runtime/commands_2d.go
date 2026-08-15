package runtime

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"bitshinbasic/internal/value"
)

func (w *World) twoDCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	need2 := func(fn func([]value.Value) (value.Value, error)) cmd {
		return func(a []value.Value) (value.Value, error) {
			if !w.ready {
				return value.Value{}, w.require()
			}
			return fn(a)
		}
	}
	return map[string]cmd{
		"graphics2d": n(func(a []value.Value) (value.Value, error) {
			return w.graphics2D(argI(a, 0, 800), argI(a, 1, 600))
		}),
		"graphics": n(func(a []value.Value) (value.Value, error) {
			return w.graphics2D(argI(a, 0, 800), argI(a, 1, 600))
		}),
		"loadimage": need2(func(a []value.Value) (value.Value, error) {
			path, err := w.openPath(argS(a, 0))
			if err != nil {
				return value.Value{}, err
			}
			img, _, err := ebitenutil.NewImageFromFile(path)
			if err != nil {
				return value.Value{}, err
			}
			id := w.nextImg
			w.nextImg++
			b := img.Bounds()
			w.images[id] = &ebiImage{img: img, w: b.Dx(), h: b.Dy()}
			return value.Num(float64(id)), nil
		}),
		"createimage": need2(func(a []value.Value) (value.Value, error) {
			ww, hh := argI(a, 0, 32), argI(a, 1, 32)
			if ww < 1 {
				ww = 1
			}
			if hh < 1 {
				hh = 1
			}
			img := w.solidImage(ww, hh, w.drawRGB[0], w.drawRGB[1], w.drawRGB[2])
			id := w.nextImg
			w.nextImg++
			w.images[id] = &ebiImage{img: img, w: ww, h: hh}
			return value.Num(float64(id)), nil
		}),
		"drawimage": need2(func(a []value.Value) (value.Value, error) {
			im := w.images[argI(a, 0, 0)]
			if im == nil {
				return z()
			}
			w.draws = append(w.draws, drawOp{kind: 0, img: im.img, x: float32(argN(a, 1, 0)), y: float32(argN(a, 2, 0))})
			return z()
		}),
		"createsprite": need2(func(a []value.Value) (value.Value, error) {
			var img *ebiten.Image
			if im := w.images[argI(a, 0, 0)]; im != nil {
				img = im.img
			} else {
				img = w.solidImage(32, 32, w.drawRGB[0], w.drawRGB[1], w.drawRGB[2])
			}
			id := w.nextID
			w.nextID++
			w.sprites[id] = &ebiSprite{img: img, sx: 1, sy: 1, visible: true}
			return value.Num(float64(id)), nil
		}),
		"positionsprite": need2(func(a []value.Value) (value.Value, error) {
			if s := w.sprites[argI(a, 0, 0)]; s != nil {
				s.x, s.y = argN(a, 1, 0), argN(a, 2, 0)
			}
			return z()
		}),
		"movesprite": need2(func(a []value.Value) (value.Value, error) {
			if s := w.sprites[argI(a, 0, 0)]; s != nil {
				s.x += argN(a, 1, 0)
				s.y += argN(a, 2, 0)
			}
			return z()
		}),
		"rotatesprite": need2(func(a []value.Value) (value.Value, error) {
			if s := w.sprites[argI(a, 0, 0)]; s != nil {
				s.rot = argN(a, 1, 0)
			}
			return z()
		}),
		"scalesprite": need2(func(a []value.Value) (value.Value, error) {
			if s := w.sprites[argI(a, 0, 0)]; s != nil {
				s.sx, s.sy = argN(a, 1, 1), argN(a, 2, 1)
			}
			return z()
		}),
		"spritex": need2(func(a []value.Value) (value.Value, error) {
			if s := w.sprites[argI(a, 0, 0)]; s != nil {
				return value.Num(s.x), nil
			}
			return value.Num(0), nil
		}),
		"spritey": need2(func(a []value.Value) (value.Value, error) {
			if s := w.sprites[argI(a, 0, 0)]; s != nil {
				return value.Num(s.y), nil
			}
			return value.Num(0), nil
		}),
		"hidesprite": need2(func(a []value.Value) (value.Value, error) {
			if s := w.sprites[argI(a, 0, 0)]; s != nil {
				s.visible = false
			}
			return z()
		}),
		"showsprite": need2(func(a []value.Value) (value.Value, error) {
			if s := w.sprites[argI(a, 0, 0)]; s != nil {
				s.visible = true
			}
			return z()
		}),
		"rect": need2(func(a []value.Value) (value.Value, error) {
			w.draws = append(w.draws, drawOp{
				kind: 1, x: float32(argN(a, 0, 0)), y: float32(argN(a, 1, 0)),
				w: float32(argN(a, 2, 10)), h: float32(argN(a, 3, 10)),
				r: w.drawRGB[0], g: w.drawRGB[1], b: w.drawRGB[2],
				filled: argI(a, 4, 1) != 0,
			})
			return z()
		}),
		"oval": need2(func(a []value.Value) (value.Value, error) {
			w.draws = append(w.draws, drawOp{
				kind: 2, x: float32(argN(a, 0, 0)), y: float32(argN(a, 1, 0)),
				w: float32(argN(a, 2, 16)), h: float32(argN(a, 3, 16)),
				r: w.drawRGB[0], g: w.drawRGB[1], b: w.drawRGB[2], filled: true,
			})
			return z()
		}),
		"line": need2(func(a []value.Value) (value.Value, error) {
			w.draws = append(w.draws, drawOp{
				kind: 3, x: float32(argN(a, 0, 0)), y: float32(argN(a, 1, 0)),
				x2: float32(argN(a, 2, 0)), y2: float32(argN(a, 3, 0)),
				r: w.drawRGB[0], g: w.drawRGB[1], b: w.drawRGB[2],
			})
			return z()
		}),
		"clscolor": n(func(a []value.Value) (value.Value, error) {
			w.clsRGB = rgbBytes(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0))
			return z()
		}),
		"rectsoverlap": n(func(a []value.Value) (value.Value, error) {
			if rectsOverlap(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0), argN(a, 4, 0), argN(a, 5, 0), argN(a, 6, 0), argN(a, 7, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"spritesoverlap": need2(func(a []value.Value) (value.Value, error) {
			s1, s2 := w.sprites[argI(a, 0, 0)], w.sprites[argI(a, 1, 0)]
			if s1 == nil || s2 == nil {
				return value.Num(0), nil
			}
			w1, h1 := spriteSize(s1)
			w2, h2 := spriteSize(s2)
			if rectsOverlap(s1.x-w1/2, s1.y-h1/2, w1, h1, s2.x-w2/2, s2.y-h2/2, w2, h2) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"imagesoverlap": need2(func(a []value.Value) (value.Value, error) {
			im1, im2 := w.images[argI(a, 0, 0)], w.images[argI(a, 3, 0)]
			if im1 == nil || im2 == nil {
				return value.Num(0), nil
			}
			if rectsOverlap(argN(a, 1, 0), argN(a, 2, 0), float64(im1.w), float64(im1.h), argN(a, 4, 0), argN(a, 5, 0), float64(im2.w), float64(im2.h)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"imagescollide": need2(func(a []value.Value) (value.Value, error) {
			im1, im2 := w.images[argI(a, 0, 0)], w.images[argI(a, 3, 0)]
			if im1 == nil || im2 == nil {
				return value.Num(0), nil
			}
			if rectsOverlap(argN(a, 1, 0), argN(a, 2, 0), float64(im1.w), float64(im1.h), argN(a, 4, 0), argN(a, 5, 0), float64(im2.w), float64(im2.h)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
	}
}

func rectsOverlap(x1, y1, w1, h1, x2, y2, w2, h2 float64) bool {
	return x1 < x2+w2 && x1+w1 > x2 && y1 < y2+h2 && y1+h1 > y2
}

func spriteSize(s *ebiSprite) (float64, float64) {
	if s == nil {
		return 32, 32
	}
	if s.img == nil {
		return 32 * s.sx, 32 * s.sy
	}
	b := s.img.Bounds()
	return float64(b.Dx()) * s.sx, float64(b.Dy()) * s.sy
}
