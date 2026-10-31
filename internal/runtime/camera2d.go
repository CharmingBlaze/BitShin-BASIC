package runtime

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"moonbasic/internal/value"
)

// 2D cameras. Camera 0 always exists and fills the window; SetCamera2D,
// MoveCamera2D, and CameraFollow2D drive it, as they always have. More
// cameras come from CreateCamera2D, each with its own viewport for split
// screen. A camera looks at a world point (its centre) with a zoom and a
// rotation, and can follow a sprite with smoothing, a dead zone, bounds,
// and shake.
type camera2D struct {
	id         int
	cx, cy     float64 // world point at the viewport centre
	zoom       float64
	rot        float64 // degrees
	vx, vy     int     // viewport; vw or vh 0 is the whole window
	vw, vh     int
	order      int
	hidden     bool
	target     int // sprite to follow, 0 none
	offX, offY float64
	smooth     float64 // seconds to close most of the gap; 0 snaps
	deadW      float64 // dead zone size in world units
	deadH      float64
	bounded    bool
	bx0, by0   float64
	bx1, by1   float64
	shakeAmp   float64
	shakeTime  float64
	shakeLeft  float64
	shakeX     float64
	shakeY     float64
}

func (w *World) cam2D(id int) *camera2D {
	if w.cams2 == nil {
		w.cams2 = map[int]*camera2D{}
	}
	if c := w.cams2[id]; c != nil {
		return c
	}
	if id != 0 {
		return nil
	}
	c := &camera2D{zoom: 1}
	vw, vh := c.viewSize(w)
	c.cx, c.cy = float64(vw)/2, float64(vh)/2
	w.cams2[0] = c
	return c
}

func (w *World) screenSize2D() (int, int) {
	sw, sh := w.scrW, w.scrH
	if sw < 1 {
		sw = 800
	}
	if sh < 1 {
		sh = 600
	}
	return sw, sh
}

// viewport is the camera's rectangle in window pixels.
func (c *camera2D) viewport(w *World) (x, y, vw, vh int) {
	if c.vw > 0 && c.vh > 0 {
		return c.vx, c.vy, c.vw, c.vh
	}
	sw, sh := w.screenSize2D()
	return 0, 0, sw, sh
}

func (c *camera2D) viewSize(w *World) (int, int) {
	_, _, vw, vh := c.viewport(w)
	return vw, vh
}

// geoM maps world coordinates to window pixels.
func (c *camera2D) geoM(w *World) ebiten.GeoM {
	x, y, vw, vh := c.viewport(w)
	var m ebiten.GeoM
	m.Translate(-(c.cx + c.shakeX), -(c.cy + c.shakeY))
	m.Rotate(-c.rot * math.Pi / 180)
	z := c.zoom
	if z <= 0 {
		z = 1
	}
	m.Scale(z, z)
	m.Translate(float64(x)+float64(vw)/2, float64(y)+float64(vh)/2)
	return m
}

// toWorld maps a window pixel to the world under this camera.
func (c *camera2D) toWorld(w *World, sx, sy float64) (float64, float64) {
	m := c.geoM(w)
	m.Invert()
	return m.Apply(sx, sy)
}

// worldRect is the axis-aligned world box the viewport can see, for culling.
func (c *camera2D) worldRect(w *World) (x0, y0, x1, y1 float64) {
	x, y, vw, vh := c.viewport(w)
	x0, y0 = math.Inf(1), math.Inf(1)
	x1, y1 = math.Inf(-1), math.Inf(-1)
	for _, p := range [4][2]float64{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		wx, wy := c.toWorld(w, float64(x)+p[0]*float64(vw), float64(y)+p[1]*float64(vh))
		x0, y0 = math.Min(x0, wx), math.Min(y0, wy)
		x1, y1 = math.Max(x1, wx), math.Max(y1, wy)
	}
	return
}

// halfView is half the viewport size in world units, ignoring rotation.
func (c *camera2D) halfView(w *World) (float64, float64) {
	vw, vh := c.viewSize(w)
	z := c.zoom
	if z <= 0 {
		z = 1
	}
	return float64(vw) / 2 / z, float64(vh) / 2 / z
}

func (c *camera2D) clampBounds(w *World) {
	if !c.bounded {
		return
	}
	hw, hh := c.halfView(w)
	clampAxis := func(v, lo, hi, half float64) float64 {
		if hi-lo <= 2*half {
			return (lo + hi) / 2 // the level is smaller than the view: centre it
		}
		return math.Max(lo+half, math.Min(hi-half, v))
	}
	c.cx = clampAxis(c.cx, c.bx0, c.bx1, hw)
	c.cy = clampAxis(c.cy, c.by0, c.by1, hh)
}

// followStep moves the camera centre toward the target point. Inside the
// dead zone it does not move; outside, it closes the gap to the zone edge.
func (c *camera2D) followStep(tx, ty, dt float64) {
	gx := deadZoneGap(c.cx, tx, c.deadW/2)
	gy := deadZoneGap(c.cy, ty, c.deadH/2)
	k := 1.0
	if c.smooth > 0 && dt > 0 {
		// Closes 95% of the gap in `smooth` seconds at any frame rate.
		k = 1 - math.Pow(0.05, dt/c.smooth)
	}
	c.cx += gx * k
	c.cy += gy * k
}

func deadZoneGap(cur, target, half float64) float64 {
	d := target - cur
	switch {
	case d > half:
		return d - half
	case d < -half:
		return d + half
	}
	return 0
}

// tickCameras2D runs once per frame after the script: follow, bounds, shake.
func (w *World) tickCameras2D() {
	dt := w.delta
	for _, c := range w.cams2 {
		if c.target != 0 {
			if s := w.sprites[c.target]; s != nil {
				x, y, _, _, _ := w.spriteWorld(s, 0)
				c.followStep(x+c.offX, y+c.offY, dt)
			}
		}
		c.clampBounds(w)
		c.shakeX, c.shakeY = 0, 0
		if c.shakeLeft > 0 && dt > 0 {
			c.shakeLeft -= dt
			fade := math.Max(c.shakeLeft, 0) / math.Max(c.shakeTime, 1e-6)
			amp := c.shakeAmp * fade * fade
			c.shakeX = (rand.Float64()*2 - 1) * amp
			c.shakeY = (rand.Float64()*2 - 1) * amp
		}
	}
}

// views2D lists the cameras to draw, camera 0 first unless it is hidden.
func (w *World) views2D() []*camera2D {
	w.cam2D(0)
	list := make([]*camera2D, 0, len(w.cams2))
	for _, c := range w.cams2 {
		if !c.hidden {
			list = append(list, c)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].order != list[j].order {
			return list[i].order < list[j].order
		}
		return list[i].id < list[j].id
	})
	return list
}

func (w *World) camera2DCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	get := func(a []value.Value, where string) (*camera2D, error) {
		c := w.cam2D(argI(a, 0, 0))
		if c == nil {
			return nil, fmt.Errorf("%s: %d is not a 2D camera", where, argI(a, 0, 0))
		}
		return c, nil
	}
	set := func(where string, fn func(c *camera2D, a []value.Value)) cmd {
		return n(func(a []value.Value) (value.Value, error) {
			c, err := get(a, where)
			if err != nil {
				return value.Value{}, err
			}
			fn(c, a)
			return z()
		})
	}
	read := func(where string, fn func(c *camera2D, a []value.Value) float64) cmd {
		return n(func(a []value.Value) (value.Value, error) {
			c, err := get(a, where)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(fn(c, a)), nil
		})
	}
	return map[string]cmd{
		// CreateCamera2D([x, y, w, h]): a camera with its own viewport.
		"createcamera2d": n(func(a []value.Value) (value.Value, error) {
			w.cam2D(0)
			w.nextCam2++
			c := &camera2D{id: w.nextCam2, zoom: 1, order: w.nextCam2}
			if len(a) >= 4 {
				c.vx, c.vy, c.vw, c.vh = argI(a, 0, 0), argI(a, 1, 0), argI(a, 2, 0), argI(a, 3, 0)
			}
			d := w.cam2D(0)
			c.cx, c.cy = d.cx, d.cy
			w.cams2[c.id] = c
			return value.Num(float64(c.id)), nil
		}),
		"freecamera2d": n(func(a []value.Value) (value.Value, error) {
			if id := argI(a, 0, 0); id != 0 {
				delete(w.cams2, id)
			}
			return z()
		}),
		// PositionCamera2D cam, x, y: centre the view on a world point.
		"positioncamera2d": set("PositionCamera2D", func(c *camera2D, a []value.Value) {
			c.cx, c.cy = argN(a, 1, 0), argN(a, 2, 0)
		}),
		"translatecamera2d": set("TranslateCamera2D", func(c *camera2D, a []value.Value) {
			c.cx += argN(a, 1, 0)
			c.cy += argN(a, 2, 0)
		}),
		"camerazoom2d": set("CameraZoom2D", func(c *camera2D, a []value.Value) {
			c.zoom = math.Max(0.01, argN(a, 1, 1))
		}),
		"camerarotation2d": set("CameraRotation2D", func(c *camera2D, a []value.Value) {
			c.rot = argN(a, 1, 0)
		}),
		"turncamera2d": set("TurnCamera2D", func(c *camera2D, a []value.Value) {
			c.rot += argN(a, 1, 0)
		}),
		// CameraViewport2D cam, x, y, w, h; w or h 0 fills the window.
		"cameraviewport2d": set("CameraViewport2D", func(c *camera2D, a []value.Value) {
			c.vx, c.vy, c.vw, c.vh = argI(a, 1, 0), argI(a, 2, 0), argI(a, 3, 0), argI(a, 4, 0)
		}),
		"cameraorder2d": set("CameraOrder2D", func(c *camera2D, a []value.Value) {
			c.order = argI(a, 1, 0)
		}),
		"showcamera2d": set("ShowCamera2D", func(c *camera2D, a []value.Value) {
			c.hidden = argI(a, 1, 1) == 0
		}),
		// CameraTarget2D cam, sprite [, offsetX, offsetY]; sprite 0 stops.
		"cameratarget2d": set("CameraTarget2D", func(c *camera2D, a []value.Value) {
			c.target = argI(a, 1, 0)
			c.offX, c.offY = argN(a, 2, 0), argN(a, 3, 0)
		}),
		"camerasmooth2d": set("CameraSmooth2D", func(c *camera2D, a []value.Value) {
			c.smooth = math.Max(0, argN(a, 1, 0.25))
		}),
		"cameradeadzone2d": set("CameraDeadZone2D", func(c *camera2D, a []value.Value) {
			c.deadW, c.deadH = math.Max(0, argN(a, 1, 0)), math.Max(0, argN(a, 2, 0))
		}),
		// CameraBounds2D cam, x0, y0, x1, y1 keeps the view inside a world
		// box; CameraBounds2D cam alone removes it.
		"camerabounds2d": set("CameraBounds2D", func(c *camera2D, a []value.Value) {
			if len(a) < 5 {
				c.bounded = false
				return
			}
			c.bounded = true
			c.bx0, c.by0 = math.Min(argN(a, 1, 0), argN(a, 3, 0)), math.Min(argN(a, 2, 0), argN(a, 4, 0))
			c.bx1, c.by1 = math.Max(argN(a, 1, 0), argN(a, 3, 0)), math.Max(argN(a, 2, 0), argN(a, 4, 0))
			c.clampBounds(w)
		}),
		// CameraShake2D cam, pixels [, seconds]
		"camerashake2d": set("CameraShake2D", func(c *camera2D, a []value.Value) {
			c.shakeAmp = math.Max(0, argN(a, 1, 8))
			c.shakeTime = math.Max(0.01, argN(a, 2, 0.4))
			c.shakeLeft = c.shakeTime
		}),
		"getcamerax2d":        read("GetCameraX2D", func(c *camera2D, _ []value.Value) float64 { return c.cx }),
		"getcameray2d":        read("GetCameraY2D", func(c *camera2D, _ []value.Value) float64 { return c.cy }),
		"getcamerazoom2d":     read("GetCameraZoom2D", func(c *camera2D, _ []value.Value) float64 { return c.zoom }),
		"getcamerarotation2d": read("GetCameraRotation2D", func(c *camera2D, _ []value.Value) float64 { return c.rot }),
		// ScreenToWorldX2D(cam, sx, sy) / ScreenToWorldY2D: what is under a pixel.
		"screentoworldx2d": read("ScreenToWorldX2D", func(c *camera2D, a []value.Value) float64 {
			x, _ := c.toWorld(w, argN(a, 1, 0), argN(a, 2, 0))
			return x
		}),
		"screentoworldy2d": read("ScreenToWorldY2D", func(c *camera2D, a []value.Value) float64 {
			_, y := c.toWorld(w, argN(a, 1, 0), argN(a, 2, 0))
			return y
		}),
		"worldtoscreenx2d": read("WorldToScreenX2D", func(c *camera2D, a []value.Value) float64 {
			m := c.geoM(w)
			x, _ := m.Apply(argN(a, 1, 0), argN(a, 2, 0))
			return x
		}),
		"worldtoscreeny2d": read("WorldToScreenY2D", func(c *camera2D, a []value.Value) float64 {
			m := c.geoM(w)
			_, y := m.Apply(argN(a, 1, 0), argN(a, 2, 0))
			return y
		}),
		// MouseWorldX2D([cam]) / MouseWorldY2D: the mouse in world space.
		"mouseworldx2d": read("MouseWorldX2D", func(c *camera2D, _ []value.Value) float64 {
			x, _ := c.toWorld(w, float64(w.mx), float64(w.my))
			return x
		}),
		"mouseworldy2d": read("MouseWorldY2D", func(c *camera2D, _ []value.Value) float64 {
			_, y := c.toWorld(w, float64(w.mx), float64(w.my))
			return y
		}),
	}
}
