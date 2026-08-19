package runtime

import (
	"fmt"
	"math"

	"github.com/g3n/engine/window"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/hajimehoshi/ebiten/v2"

	"bitshinbasic/internal/value"
)

func (w *World) classic3DCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"entityparent": need(func(a []value.Value) (value.Value, error) {
			child, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			pid := argI(a, 1, 0)
			cn := child.node.GetNode()
			if p := cn.Parent(); p != nil {
				p.GetNode().Remove(child.node)
			}
			w.parentNode(pid).GetNode().Add(child.node)
			child.parent = pid
			return z()
		}),
		"getparent": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.parent)), nil
		}),
		"nameentity": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			e.name = argS(a, 1)
			if e.node != nil {
				e.node.GetNode().SetName(e.name)
			}
			return z()
		}),
		"entityname": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Str(e.name), nil
		}),
		"lightcolor": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.lgt != nil {
				e.lgt.SetColor(rgb(argN(a, 1, 255), argN(a, 2, 255), argN(a, 3, 255)))
			}
			return z()
		}),
		"lightrange": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			rng := float32(argN(a, 1, 10))
			if rng < 0.1 {
				rng = 0.1
			}
			if p, ok := e.node.(interface{ SetLinearDecay(float32) }); ok {
				p.SetLinearDecay(1 / rng)
			}
			return z()
		}),
		"camerarange": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.cam != nil {
				e.cam.SetNear(float32(argN(a, 1, 0.1)))
				e.cam.SetFar(float32(argN(a, 2, 4000)))
			}
			return z()
		}),
		"camerazoom": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.cam != nil {
				z := float32(argN(a, 1, 1))
				if z <= 0 {
					z = 1
				}
				e.cam.SetFov(65 / z)
			}
			return z()
		}),
		"cameraviewport": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.cam == nil {
				return value.Value{}, fmt.Errorf("CameraViewport: not a camera")
			}
			e.vpX, e.vpY = argI(a, 1, 0), argI(a, 2, 0)
			e.vpW, e.vpH = argI(a, 3, w.scrW), argI(a, 4, w.scrH)
			return z()
		}),
		"camerafogmode": need(func(a []value.Value) (value.Value, error) {
			w.setCameraFogMode(a)
			return z()
		}),
		"camerafogcolor": need(func(a []value.Value) (value.Value, error) {
			w.setCameraFogColor(a)
			return z()
		}),
		"camerafogrange": need(func(a []value.Value) (value.Value, error) {
			w.setCameraFogRange(a)
			return z()
		}),
		"camerafogdensity": need(func(a []value.Value) (value.Value, error) {
			w.setCameraFogDensity(a)
			return z()
		}),
		"enablefog": need(func(a []value.Value) (value.Value, error) {
			if argI(a, 0, 1) == 0 {
				w.fogMode = 0
			} else if w.fogMode == 0 {
				w.fogMode = 1
			}
			w.wx.fogOwned = false
			w.applyLitShaders()
			return z()
		}),
		"entityvisible": need(func(a []value.Value) (value.Value, error) {
			n, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if n.Visible() {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"entityshininess": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.mat != nil {
				e.mat.SetShininess(float32(argN(a, 1, 0)) * 128)
			}
			return z()
		}),
		"entityspecular": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.mat != nil {
				e.mat.SetSpecularColor(rgb(argN(a, 1, 28), argN(a, 2, 28), argN(a, 3, 28)))
			}
			return z()
		}),
		"scaletexture": need(func(a []value.Value) (value.Value, error) {
			if t := w.texs[argI(a, 0, 0)]; t != nil && t.tex != nil {
				t.tex.SetRepeat(float32(argN(a, 1, 1)), float32(argN(a, 2, 1)))
			}
			return z()
		}),
		"entityscalex": need(func(a []value.Value) (value.Value, error) {
			n, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(n.Scale().X)), nil
		}),
		"entityscaley": need(func(a []value.Value) (value.Value, error) {
			n, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(n.Scale().Y)), nil
		}),
		"entityscalez": need(func(a []value.Value) (value.Value, error) {
			n, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(n.Scale().Z)), nil
		}),
		"getentitytype": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.etype)), nil
		}),
		"resetentity": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			e.collided = e.collided[:0]
			e.hitID = 0
			return z()
		}),
		"collisionentity": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.hitID)), nil
		}),
		"collisionx": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.hitX)), nil
		}),
		"collisiony": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.hitY)), nil
		}),
		"collisionz": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.hitZ)), nil
		}),
		"mousexspeed": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.mxs)), nil }),
		"mouseyspeed": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.mys)), nil }),
		"movemouse": n(func(a []value.Value) (value.Value, error) {
			x, y := argI(a, 0, 0), argI(a, 1, 0)
			if w.app != nil {
				if gw, ok := w.app.IWindow.(*window.GlfwWindow); ok {
					gw.SetCursorPos(float64(x), float64(y))
				}
			}
			w.mx, w.my = float32(x), float32(y)
			w.prevMX, w.prevMY = w.mx, w.my
			return z()
		}),
		"hidepointer": n(func(a []value.Value) (value.Value, error) {
			if w.mode2D {
				ebiten.SetCursorMode(ebiten.CursorModeHidden)
			} else if w.app != nil {
				if gw, ok := w.app.IWindow.(*window.GlfwWindow); ok {
					gw.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
				}
			}
			return z()
		}),
		"showpointer": n(func(a []value.Value) (value.Value, error) {
			if w.mode2D {
				ebiten.SetCursorMode(ebiten.CursorModeVisible)
			} else if w.app != nil {
				if gw, ok := w.app.IWindow.(*window.GlfwWindow); ok {
					gw.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
				}
			}
			return z()
		}),
		"deltayaw": need(func(a []value.Value) (value.Value, error) {
			return w.deltaAngle(argI(a, 0, 0), argI(a, 1, 0), true)
		}),
		"deltapitch": need(func(a []value.Value) (value.Value, error) {
			return w.deltaAngle(argI(a, 0, 0), argI(a, 1, 0), false)
		}),
		"vectoryaw": n(func(a []value.Value) (value.Value, error) {
			x, z := argN(a, 0, 0), argN(a, 2, 0)
			return value.Num(math.Atan2(x, z) * 180 / math.Pi), nil
		}),
		"vectorpitch": n(func(a []value.Value) (value.Value, error) {
			x, y, z := argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0)
			return value.Num(math.Atan2(y, math.Sqrt(x*x+z*z)) * 180 / math.Pi), nil
		}),
	}
}

func (w *World) deltaAngle(src, dest int, yaw bool) (value.Value, error) {
	a, err := w.ent(src)
	if err != nil {
		return value.Value{}, err
	}
	b, err := w.ent(dest)
	if err != nil {
		return value.Value{}, err
	}
	pa, pb := worldPos(a.node.GetNode()), worldPos(b.node.GetNode())
	dx, dy, dz := float64(pb.X-pa.X), float64(pb.Y-pa.Y), float64(-(pb.Z - pa.Z))
	if yaw {
		want := math.Atan2(dx, dz) * 180 / math.Pi
		return value.Num(wrap180(want - float64(a.yaw))), nil
	}
	want := math.Atan2(dy, math.Sqrt(dx*dx+dz*dz)) * 180 / math.Pi
	return value.Num(wrap180(want - float64(a.pitch))), nil
}

func wrap180(d float64) float64 {
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}
