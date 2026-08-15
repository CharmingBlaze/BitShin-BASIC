package runtime

import (
	"fmt"

	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

func (w *World) xformCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	_ = z
	return map[string]cmd{
		"tformpoint": need(func(a []value.Value) (value.Value, error) {
			return w.tform(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), argI(a, 3, 0), argI(a, 4, 0), true)
		}),
		"tformvector": need(func(a []value.Value) (value.Value, error) {
			return w.tform(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), argI(a, 3, 0), argI(a, 4, 0), false)
		}),
		"tformedx": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.tformX)), nil }),
		"tformedy": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.tformY)), nil }),
		"tformedz": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.tformZ)), nil }),
		"projectedx": need(func(a []value.Value) (value.Value, error) {
			x, _, _, err := w.projectPoint(a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(x), nil
		}),
		"projectedy": need(func(a []value.Value) (value.Value, error) {
			_, y, _, err := w.projectPoint(a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(y), nil
		}),
		"projectedz": need(func(a []value.Value) (value.Value, error) {
			_, _, zndc, err := w.projectPoint(a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(zndc), nil
		}),
		"unprojectx": need(func(a []value.Value) (value.Value, error) {
			x, _, _, err := w.unprojectPoint(a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(x), nil
		}),
		"unprojecty": need(func(a []value.Value) (value.Value, error) {
			_, y, _, err := w.unprojectPoint(a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(y), nil
		}),
		"unprojectz": need(func(a []value.Value) (value.Value, error) {
			_, _, zz, err := w.unprojectPoint(a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(zz), nil
		}),
	}
}

func (w *World) tform(x, y, z float64, src, dest int, point bool) (value.Value, error) {
	gx, gy, gz := toG3N(float32(x), float32(y), float32(z))
	v := math32.Vector3{gx, gy, gz}
	if src != 0 {
		m, err := w.worldMat(src)
		if err != nil {
			return value.Value{}, err
		}
		if point {
			v.ApplyMatrix4(&m)
		} else {
			v = applyMatDir(m, v)
		}
	}
	if dest != 0 {
		m, err := w.worldMat(dest)
		if err != nil {
			return value.Value{}, err
		}
		var inv math32.Matrix4
		if err := inv.GetInverse(&m); err != nil {
			return value.Value{}, fmt.Errorf("TForm: cannot invert dest")
		}
		if point {
			v.ApplyMatrix4(&inv)
		} else {
			v = applyMatDir(inv, v)
		}
	}
	bx, by, bz := fromG3N(v.X, v.Y, v.Z)
	w.tformX, w.tformY, w.tformZ = bx, by, bz
	return value.Num(0), nil
}

func applyMatDir(m math32.Matrix4, v math32.Vector3) math32.Vector3 {
	o := math32.Vector3{}
	o.ApplyMatrix4(&m)
	v.ApplyMatrix4(&m)
	return math32.Vector3{v.X - o.X, v.Y - o.Y, v.Z - o.Z}
}

func (w *World) worldMat(id int) (math32.Matrix4, error) {
	e, err := w.ent(id)
	if err != nil {
		return math32.Matrix4{}, err
	}
	if w.scene != nil {
		w.scene.UpdateMatrixWorld()
	} else {
		e.node.GetNode().UpdateMatrixWorld()
	}
	return e.node.GetNode().MatrixWorld(), nil
}

func (w *World) projectPoint(a []value.Value) (float64, float64, float64, error) {
	cam := w.cam
	off := 0
	if len(a) >= 1 {
		if e, err := w.ent(argI(a, 0, 0)); err == nil && e.cam != nil {
			cam = e.cam
			off = 1
		}
	}
	if cam == nil {
		return 0, 0, 0, fmt.Errorf("Projected: Graphics3D camera required")
	}
	if w.scene != nil {
		w.scene.UpdateMatrixWorld()
	}
	gx, gy, gz := toG3N(float32(argN(a, off, 0)), float32(argN(a, off+1, 0)), float32(argN(a, off+2, 0)))
	v := math32.Vector3{gx, gy, gz}
	cam.Project(&v)
	ww, hh := float64(w.scrW), float64(w.scrH)
	if ww < 1 {
		ww = 800
	}
	if hh < 1 {
		hh = 600
	}
	return (float64(v.X) + 1) / 2 * ww, (1 - float64(v.Y)) / 2 * hh, float64(v.Z), nil
}

func (w *World) unprojectPoint(a []value.Value) (float64, float64, float64, error) {
	cam := w.cam
	off := 0
	if len(a) >= 1 {
		if e, err := w.ent(argI(a, 0, 0)); err == nil && e.cam != nil {
			cam = e.cam
			off = 1
		}
	}
	if cam == nil {
		return 0, 0, 0, fmt.Errorf("Unproject: Graphics3D camera required")
	}
	if w.scene != nil {
		w.scene.UpdateMatrixWorld()
	}
	ww, hh := float64(w.scrW), float64(w.scrH)
	if ww < 1 {
		ww = 800
	}
	if hh < 1 {
		hh = 600
	}
	sx, sy := argN(a, off, 0), argN(a, off+1, 0)
	depth := argN(a, off+2, 0)
	nx := float32(sx/ww)*2 - 1
	ny := 1 - float32(sy/hh)*2
	v := math32.Vector3{nx, ny, float32(depth)}
	cam.Unproject(&v)
	bx, by, bz := fromG3N(v.X, v.Y, v.Z)
	return float64(bx), float64(by), float64(bz), nil
}
