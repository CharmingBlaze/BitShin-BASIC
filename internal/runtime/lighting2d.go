package runtime

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"moonbasic/internal/value"
)

// 2D lighting. When it is on, each camera view gets a light map: the
// ambient colour, plus every light's radial falloff added on top, with hard
// shadows cut where occluders block the light. The light map multiplies the
// world (tiles, particles, sprites); the screen-space draws (Text, Rect,
// HudPrint) go on top unlit. Sprites marked unlit draw after the multiply,
// so they glow in the dark.
type light2D struct {
	x, y      float64
	radius    float64
	r, g, b   float64 // 0..1
	intensity float64
	shadows   bool
	parent    int     // sprite the light follows, 0 none
	dir       float64 // cone direction, degrees
	cone      float64 // cone full angle, degrees; 0 is all round
	visible   bool
}

type occluder2D struct {
	x, y, w, h float64
}

type lighting2D struct {
	on        bool
	ambient   [3]float64
	sun       [3]float64 // colour * intensity; zero is no sun
	sunAngle  float64    // degrees the light travels toward
	sunLength float64    // shadow length in world units
	lights    map[int]*light2D
	next      int
	occ       map[int]*occluder2D
	nextOcc   int
	tileOcc   map[*tileMap]int // tilemap -> layer whose cells block light
	lightMap  *ebiten.Image
	scratch   *ebiten.Image
	falloff   *ebiten.Image
	tris      []ebiten.Vertex
	idx       []uint16
}

func (w *World) light2D() *lighting2D {
	if w.lit2 == nil {
		w.lit2 = &lighting2D{ambient: [3]float64{0.15, 0.15, 0.2}, sunLength: 400,
			lights: map[int]*light2D{}, occ: map[int]*occluder2D{}, tileOcc: map[*tileMap]int{}}
	}
	return w.lit2
}

func (w *World) lighting2DOn() bool { return w.lit2 != nil && w.lit2.on }

// falloffImage is a white disc that fades out smoothly: (1 - d^2)^2.
func (lg *lighting2D) falloffImage() *ebiten.Image {
	if lg.falloff != nil {
		return lg.falloff
	}
	const n = 256
	pix := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx, dy := (float64(x)+0.5)/n*2-1, (float64(y)+0.5)/n*2-1
			d2 := dx*dx + dy*dy
			v := 0.0
			if d2 < 1 {
				v = (1 - d2) * (1 - d2)
			}
			b := byte(math.Round(v * 255))
			i := (y*n + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = b, b, b, b
		}
	}
	lg.falloff = ebiten.NewImage(n, n)
	lg.falloff.WritePixels(pix)
	return lg.falloff
}

func ensureImage(img **ebiten.Image, w, h int) *ebiten.Image {
	if *img != nil {
		b := (*img).Bounds()
		if b.Dx() >= w && b.Dy() >= h {
			return *img
		}
		(*img).Deallocate()
	}
	*img = ebiten.NewImage(w, h)
	return *img
}

// lightPos is a light's world position, following its sprite if it has one.
func (w *World) lightPos(l *light2D) (float64, float64, float64) {
	if p := w.sprites[l.parent]; l.parent != 0 && p != nil {
		x, y, rot, _, _ := w.spriteWorld(p, 0)
		sin, cos := math.Sincos(rot * math.Pi / 180)
		return x + l.x*cos - l.y*sin, y + l.x*sin + l.y*cos, l.dir + rot
	}
	return l.x, l.y, l.dir
}

// edge2D is one side of an occluder, wound clockwise on screen, so its
// outward normal is (by-ay, ax-bx).
type edge2D struct{ ax, ay, bx, by float64 }

// occluderEdges appends the world-space edges that block light near
// (cx, cy) within reach: rectangles, occluding sprites, and tile cells.
// Tile edges between two solid cells are left out, so a wall of tiles
// casts one clean shadow.
func (w *World) occluderEdges(cx, cy, reach float64, out []edge2D) []edge2D {
	lg := w.lit2
	quad := func(q [4][2]float64) {
		area := 0.0
		for i := 0; i < 4; i++ {
			a, b := q[i], q[(i+1)%4]
			area += a[0]*b[1] - b[0]*a[1]
		}
		for i := 0; i < 4; i++ {
			a, b := q[i], q[(i+1)%4]
			if area < 0 { // mirrored by a flip: wind it back
				a, b = b, a
			}
			out = append(out, edge2D{a[0], a[1], b[0], b[1]})
		}
	}
	near := func(x, y, ww, hh float64) bool {
		return x <= cx+reach && x+ww >= cx-reach && y <= cy+reach && y+hh >= cy-reach
	}
	for _, o := range lg.occ {
		if near(o.x, o.y, o.w, o.h) {
			quad([4][2]float64{{o.x, o.y}, {o.x + o.w, o.y}, {o.x + o.w, o.y + o.h}, {o.x, o.y + o.h}})
		}
	}
	for _, s := range w.sprites {
		if s == nil || !s.occluder || !s.visible || s.img == nil {
			continue
		}
		_, bw, bh := s.frameImage()
		x, y, rot, sx, sy := w.spriteWorld(s, 0)
		r := float64(bw)*math.Abs(sx) + float64(bh)*math.Abs(sy)
		if !near(x-r, y-r, 2*r, 2*r) {
			continue
		}
		sin, cos := math.Sincos(rot * math.Pi / 180)
		var q [4][2]float64
		for i, c := range [4][2]float64{{-0.5, -0.5}, {0.5, -0.5}, {0.5, 0.5}, {-0.5, 0.5}} {
			lx := (c[0] - s.pvx) * float64(bw) * sx
			ly := (c[1] - s.pvy) * float64(bh) * sy
			q[i] = [2]float64{x + lx*cos - ly*sin, y + lx*sin + ly*cos}
		}
		quad(q)
	}
	for tm, li := range lg.tileOcc {
		l := tm.layer(li)
		if l == nil {
			continue
		}
		solid := func(x, y int) bool {
			return x >= 0 && y >= 0 && x < tm.cols && y < tm.rows && tileID(l.cells[y*tm.cols+x]) > 0
		}
		x0, y0 := max(0, floorCell(cx-reach, tm.tw)), max(0, floorCell(cy-reach, tm.th))
		x1, y1 := min(tm.cols-1, floorCell(cx+reach, tm.tw)), min(tm.rows-1, floorCell(cy+reach, tm.th))
		tw, th := float64(tm.tw), float64(tm.th)
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				if !solid(x, y) {
					continue
				}
				px, py := float64(x)*tw, float64(y)*th
				if !solid(x, y-1) {
					out = append(out, edge2D{px, py, px + tw, py})
				}
				if !solid(x+1, y) {
					out = append(out, edge2D{px + tw, py, px + tw, py + th})
				}
				if !solid(x, y+1) {
					out = append(out, edge2D{px + tw, py + th, px, py + th})
				}
				if !solid(x-1, y) {
					out = append(out, edge2D{px, py + th, px, py})
				}
			}
		}
	}
	return out
}

// cutShadows erases, from img, everything behind the occluders as seen
// from a point light at (lx, ly), or along direction (dx, dy) for the sun.
// Only edges facing away from the light throw shadow, so the lit faces of
// walls and crates stay lit. Points are world space; cm takes them to img.
func (w *World) cutShadows(img *ebiten.Image, cm ebiten.GeoM, edges []edge2D, lx, ly, dx, dy, far float64, point bool) {
	lg := w.lit2
	lg.tris, lg.idx = lg.tris[:0], lg.idx[:0]
	vert := func(x, y float64) {
		sx, sy := cm.Apply(x, y)
		lg.tris = append(lg.tris, ebiten.Vertex{DstX: float32(sx), DstY: float32(sy), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1})
	}
	flush := func() {
		if len(lg.idx) == 0 {
			return
		}
		img.DrawTriangles(lg.tris, lg.idx, w.whitePixel(), &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendDestinationOut})
		lg.tris, lg.idx = lg.tris[:0], lg.idx[:0]
	}
	for _, e := range edges {
		nx, ny := e.by-e.ay, e.ax-e.bx
		tx, ty := dx, dy
		if point {
			tx, ty = (e.ax+e.bx)/2-lx, (e.ay+e.by)/2-ly
		}
		if nx*tx+ny*ty <= 0 {
			continue
		}
		ax, ay, bx, by := dx, dy, dx, dy
		if point {
			ax, ay = e.ax-lx, e.ay-ly
			bx, by = e.bx-lx, e.by-ly
			if l := math.Hypot(ax, ay); l > 1e-6 {
				ax, ay = ax/l, ay/l
			}
			if l := math.Hypot(bx, by); l > 1e-6 {
				bx, by = bx/l, by/l
			}
		}
		base := uint16(len(lg.tris))
		vert(e.ax, e.ay)
		vert(e.bx, e.by)
		vert(e.bx+bx*far, e.by+by*far)
		vert(e.ax+ax*far, e.ay+ay*far)
		lg.idx = append(lg.idx, base, base+1, base+2, base, base+2, base+3)
		if len(lg.tris) > 60000 {
			flush()
		}
	}
	flush()
}

// renderLight2D builds the light map for one camera and multiplies it over
// the viewport.
func (w *World) renderLight2D(dst *ebiten.Image, c *camera2D) {
	lg := w.lit2
	vx, vy, vw, vh := c.viewport(w)
	sw, sh := w.screenSize2D()
	lm := ensureImage(&lg.lightMap, sw, sh).SubImage(rectXYWH(vx, vy, vw, vh)).(*ebiten.Image)
	sc := ensureImage(&lg.scratch, sw, sh).SubImage(rectXYWH(vx, vy, vw, vh)).(*ebiten.Image)
	lm.Fill(color.RGBA{u8f(lg.ambient[0]), u8f(lg.ambient[1]), u8f(lg.ambient[2]), 255})
	cm := c.geoM(w)
	zoom := c.zoom
	if zoom <= 0 {
		zoom = 1
	}
	wx0, wy0, wx1, wy1 := c.worldRect(w)
	// A sub-image used as a source draws from (0, 0), so put each one back
	// at its own corner.
	addAt := func(src *ebiten.Image) {
		op := &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter}
		op.GeoM.Translate(float64(src.Bounds().Min.X), float64(src.Bounds().Min.Y))
		lm.DrawImage(src, op)
	}

	if lg.sun != [3]float64{} {
		sc.Clear()
		sc.Fill(color.RGBA{u8f(lg.sun[0]), u8f(lg.sun[1]), u8f(lg.sun[2]), 255})
		sin, cos := math.Sincos(lg.sunAngle * math.Pi / 180)
		mx, my := (wx0+wx1)/2, (wy0+wy1)/2
		reach := math.Hypot(wx1-wx0, wy1-wy0)/2 + lg.sunLength
		w.cutShadows(sc, cm, w.occluderEdges(mx, my, reach, nil), 0, 0, cos, sin, lg.sunLength, false)
		addAt(sc)
	}

	for _, l := range lg.lights {
		if !l.visible || l.radius <= 0 || l.intensity <= 0 {
			continue
		}
		x, y, dir := w.lightPos(l)
		if x+l.radius < wx0 || x-l.radius > wx1 || y+l.radius < wy0 || y-l.radius > wy1 {
			continue
		}
		// Work only in the light's on-screen box.
		bx0, by0 := cm.Apply(x, y)
		rr := l.radius*zoom + 2
		box := rectXYWH(int(bx0-rr), int(by0-rr), int(2*rr)+1, int(2*rr)+1).Intersect(sc.Bounds())
		if box.Empty() {
			continue
		}
		tmp := sc.SubImage(box).(*ebiten.Image)
		tmp.Clear()
		fo := lg.falloffImage()
		opt := &ebiten.DrawImageOptions{}
		n := float64(fo.Bounds().Dx())
		opt.GeoM.Translate(-n/2, -n/2)
		opt.GeoM.Scale(2*l.radius/n, 2*l.radius/n)
		opt.GeoM.Translate(x, y)
		opt.GeoM.Concat(cm)
		k := float32(l.intensity)
		opt.ColorScale.Scale(float32(l.r)*k, float32(l.g)*k, float32(l.b)*k, 1)
		tmp.DrawImage(fo, opt)
		if l.cone > 0 && l.cone < 360 {
			w.cutCone(tmp, cm, x, y, dir, l.cone, l.radius)
		}
		if l.shadows {
			w.cutShadows(tmp, cm, w.occluderEdges(x, y, l.radius, nil), x, y, 0, 0, l.radius*2, true)
		}
		addAt(tmp)
	}

	mul := &ebiten.DrawImageOptions{Blend: blendMul}
	mul.GeoM.Translate(float64(vx), float64(vy))
	dst.DrawImage(lm, mul)
}

// cutCone erases the part of a light outside its cone.
func (w *World) cutCone(img *ebiten.Image, cm ebiten.GeoM, x, y, dir, cone, radius float64) {
	lg := w.lit2
	lg.tris, lg.idx = lg.tris[:0], lg.idx[:0]
	half := cone / 2 * math.Pi / 180
	d := dir * math.Pi / 180
	// A fan over the outside arc, from the cone's far edge round to its near edge.
	steps := 24
	r := radius * 1.5
	sx, sy := cm.Apply(x, y)
	lg.tris = append(lg.tris, ebiten.Vertex{DstX: float32(sx), DstY: float32(sy), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1})
	span := 2*math.Pi - 2*half
	for i := 0; i <= steps; i++ {
		a := d + half + span*float64(i)/float64(steps)
		px, py := cm.Apply(x+math.Cos(a)*r, y+math.Sin(a)*r)
		lg.tris = append(lg.tris, ebiten.Vertex{DstX: float32(px), DstY: float32(py), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1})
		if i > 0 {
			lg.idx = append(lg.idx, 0, uint16(i), uint16(i+1))
		}
	}
	img.DrawTriangles(lg.tris, lg.idx, w.whitePixel(), &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendDestinationOut})
}

func (w *World) lighting2DCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	lightOf := func(a []value.Value, where string) (*light2D, error) {
		l := w.light2D().lights[argI(a, 0, 0)]
		if l == nil {
			return nil, fmt.Errorf("%s: %d is not a 2D light", where, argI(a, 0, 0))
		}
		return l, nil
	}
	set := func(where string, fn func(l *light2D, a []value.Value)) cmd {
		return n(func(a []value.Value) (value.Value, error) {
			l, err := lightOf(a, where)
			if err != nil {
				return value.Value{}, err
			}
			fn(l, a)
			return z()
		})
	}
	return map[string]cmd{
		// EnableLighting2D [on]
		"enablelighting2d": n(func(a []value.Value) (value.Value, error) {
			w.light2D().on = argI(a, 0, 1) != 0
			return z()
		}),
		// SetAmbient2D r, g, b: the light where no lamp reaches (0-255).
		"setambient2d": n(func(a []value.Value) (value.Value, error) {
			lg := w.light2D()
			lg.ambient = [3]float64{float64(colorArg(argN(a, 0, 40))), float64(colorArg(argN(a, 1, 40))), float64(colorArg(argN(a, 2, 50)))}
			return z()
		}),
		// SetSunLight2D r, g, b [, intensity, angle, shadowLength]: light
		// everywhere, with shadows cast along angle (degrees, 90 is down).
		"setsunlight2d": n(func(a []value.Value) (value.Value, error) {
			lg := w.light2D()
			k := argN(a, 3, 1)
			lg.sun = [3]float64{float64(colorArg(argN(a, 0, 255))) * k, float64(colorArg(argN(a, 1, 255))) * k, float64(colorArg(argN(a, 2, 255))) * k}
			lg.sunAngle = argN(a, 4, 90)
			lg.sunLength = math.Max(0, argN(a, 5, 400))
			return z()
		}),
		// CreateLight2D(x, y, radius [, r, g, b, intensity])
		"createlight2d": n(func(a []value.Value) (value.Value, error) {
			lg := w.light2D()
			lg.next++
			lg.lights[lg.next] = &light2D{
				x: argN(a, 0, 0), y: argN(a, 1, 0), radius: math.Max(1, argN(a, 2, 200)),
				r: float64(colorArg(argN(a, 3, 255))), g: float64(colorArg(argN(a, 4, 240))), b: float64(colorArg(argN(a, 5, 220))),
				intensity: argN(a, 6, 1), shadows: true, visible: true,
			}
			return value.Num(float64(lg.next)), nil
		}),
		"freelight2d": n(func(a []value.Value) (value.Value, error) {
			delete(w.light2D().lights, argI(a, 0, 0))
			return z()
		}),
		"positionlight2d": set("PositionLight2D", func(l *light2D, a []value.Value) {
			l.x, l.y = argN(a, 1, 0), argN(a, 2, 0)
		}),
		"setlight2dcolor": set("SetLight2DColor", func(l *light2D, a []value.Value) {
			l.r, l.g, l.b = float64(colorArg(argN(a, 1, 255))), float64(colorArg(argN(a, 2, 255))), float64(colorArg(argN(a, 3, 255)))
		}),
		"setlight2dradius": set("SetLight2DRadius", func(l *light2D, a []value.Value) {
			l.radius = math.Max(1, argN(a, 1, 200))
		}),
		"setlight2dintensity": set("SetLight2DIntensity", func(l *light2D, a []value.Value) {
			l.intensity = math.Max(0, argN(a, 1, 1))
		}),
		"setlight2dshadows": set("SetLight2DShadows", func(l *light2D, a []value.Value) {
			l.shadows = argI(a, 1, 1) != 0
		}),
		// SetLight2DCone light, direction, angle: a spotlight; angle 0 or
		// 360 is all round.
		"setlight2dcone": set("SetLight2DCone", func(l *light2D, a []value.Value) {
			l.dir, l.cone = argN(a, 1, 0), math.Max(0, argN(a, 2, 0))
		}),
		"showlight2d": set("ShowLight2D", func(l *light2D, a []value.Value) {
			l.visible = argI(a, 1, 1) != 0
		}),
		// ParentLight2D light, sprite [, offsetX, offsetY]: the light rides
		// along; its cone turns with the sprite.
		"parentlight2d": set("ParentLight2D", func(l *light2D, a []value.Value) {
			l.parent = argI(a, 1, 0)
			l.x, l.y = argN(a, 2, 0), argN(a, 3, 0)
		}),
		// AddOccluder2D(x, y, w, h): a wall that blocks light.
		"addoccluder2d": n(func(a []value.Value) (value.Value, error) {
			lg := w.light2D()
			lg.nextOcc++
			lg.occ[lg.nextOcc] = &occluder2D{argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 16), argN(a, 3, 16)}
			return value.Num(float64(lg.nextOcc)), nil
		}),
		"freeoccluder2d": n(func(a []value.Value) (value.Value, error) {
			delete(w.light2D().occ, argI(a, 0, 0))
			return z()
		}),
		// SetSpriteOccluder sprite [, on]: the sprite's outline blocks light.
		"setspriteoccluder": n(func(a []value.Value) (value.Value, error) {
			s := w.sprites[argI(a, 0, 0)]
			if s == nil {
				return value.Value{}, fmt.Errorf("SetSpriteOccluder: %d is not a sprite", argI(a, 0, 0))
			}
			s.occluder = argI(a, 1, 1) != 0
			w.light2D()
			return z()
		}),
		// SetSpriteLit sprite, on: 0 draws it after lighting, at full
		// brightness (lamps, glowing eyes, pickups).
		"setspritelit": n(func(a []value.Value) (value.Value, error) {
			s := w.sprites[argI(a, 0, 0)]
			if s == nil {
				return value.Value{}, fmt.Errorf("SetSpriteLit: %d is not a sprite", argI(a, 0, 0))
			}
			s.unlit = argI(a, 1, 1) == 0
			return z()
		}),
		// SetTileOccluders map, layer [, on]: non-empty cells block light.
		"settileoccluders": n(func(a []value.Value) (value.Value, error) {
			tm := w.tiles[argI(a, 0, 0)]
			if tm == nil {
				return value.Value{}, fmt.Errorf("SetTileOccluders: %d is not a tilemap", argI(a, 0, 0))
			}
			if tm.layer(argI(a, 1, 0)) == nil {
				return value.Value{}, fmt.Errorf("SetTileOccluders: layer %d of %d", argI(a, 1, 0), len(tm.layers))
			}
			lg := w.light2D()
			if argI(a, 2, 1) == 0 {
				delete(lg.tileOcc, tm)
			} else {
				lg.tileOcc[tm] = argI(a, 1, 0)
			}
			return z()
		}),
	}
}
