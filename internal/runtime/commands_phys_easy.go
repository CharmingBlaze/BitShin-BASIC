package runtime

import (
	"fmt"
	"math"
	"strings"

	"bitshinbasic/internal/phys3d"
	"bitshinbasic/internal/value"
)

// physicsMaterials is friction, restitution, linear damping, angular damping.
var physicsMaterials = map[string][4]float32{
	"ice":     {0.02, 0.02, 0.05, 0.05},
	"glass":   {0.12, 0.45, 0.02, 0.02},
	"rubber":  {0.95, 0.80, 0.15, 0.20},
	"wood":    {0.55, 0.25, 0.40, 0.40},
	"metal":   {0.25, 0.10, 0.02, 0.02},
	"stone":   {0.75, 0.05, 0.20, 0.20},
	"plastic": {0.45, 0.35, 0.10, 0.10},
	"bouncy":  {0.40, 0.95, 0.05, 0.05},
	"default": {0.50, 0.35, 0.05, 0.05},
}

func (w *World) easyPhysCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"collide": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if _, err := w.ent(id); err != nil {
				return value.Value{}, err
			}
			motion, mass, explicit := parseCollideMotion(a)
			if err := w.collideEntity(id, motion, mass, explicit); err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(id)), nil
		}),
		"setphysicsmaterial": need(func(a []value.Value) (value.Value, error) {
			if err := w.applyPhysicsMaterial(argI(a, 0, 0), argS(a, 1)); err != nil {
				return value.Value{}, err
			}
			return z()
		}),
		"getphysicsmaterial": n(func(a []value.Value) (value.Value, error) {
			e := w.ents[argI(a, 0, 0)]
			if e == nil {
				return value.Str(""), nil
			}
			return value.Str(e.physMat), nil
		}),
		"raycasthit": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			h, ok := w.phys3.RaycastDetail(
				float32(argN(a, 0, 0)), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)),
				float32(argN(a, 3, 0)), float32(argN(a, 4, 0)), float32(argN(a, 5, 0)),
			)
			return rayHitValue(h, ok), nil
		}),
		"raycastall": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			max := argI(a, 6, 16)
			if max <= 0 {
				max = 16
			}
			hits := w.phys3.RaycastAll(
				float32(argN(a, 0, 0)), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)),
				float32(argN(a, 3, 0)), float32(argN(a, 4, 0)), float32(argN(a, 5, 0)),
				max,
			)
			elems := make([]value.Value, len(hits))
			for i, h := range hits {
				elems[i] = rayHitValue(h, true)
			}
			return value.Value{Kind: value.KindArray, Elems: elems}, nil
		}),
		"getraynormalx": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.pickNX)), nil
		}),
		"getraynormaly": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.pickNY)), nil
		}),
		"getraynormalz": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.pickNZ)), nil
		}),
		"getrayfraction": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.pickFrac)), nil
		}),
	}
}

func parseCollideMotion(a []value.Value) (motion int, mass float32, explicit bool) {
	motion = phys3d.MotionTypeDynamic
	if len(a) < 2 {
		return motion, 0, false
	}
	if a[1].Kind == value.KindStr {
		switch strings.ToLower(strings.TrimSpace(a[1].Str)) {
		case "static":
			motion = phys3d.MotionTypeStatic
		case "kinematic":
			motion = phys3d.MotionTypeKinematic
		default:
			motion = phys3d.MotionTypeDynamic
		}
	} else {
		switch a[1].Int() {
		case phys3d.MotionTypeStatic:
			motion = phys3d.MotionTypeStatic
		case phys3d.MotionTypeKinematic:
			motion = phys3d.MotionTypeKinematic
		case phys3d.MotionTypeDynamic:
			motion = phys3d.MotionTypeDynamic
		default:
			if a[1].Number() > 0 {
				motion = phys3d.MotionTypeDynamic
				mass = float32(a[1].Number())
				explicit = true
			}
		}
	}
	if motion == phys3d.MotionTypeDynamic && len(a) >= 3 && a[2].Number() > 0 {
		mass = float32(a[2].Number())
		explicit = true
	}
	return motion, mass, explicit
}

func (w *World) collideEntity(id, motion int, mass float32, explicit bool) error {
	e := w.ents[id]
	if e == nil {
		return fmt.Errorf("Collide: entity %d not found", id)
	}
	hx, hy, hz, sphere := meshColliderExtents(e)
	if !explicit || mass <= 0 {
		mass = volumeMass(sphere, hx, hy, hz)
	}
	w.ensurePhys3()
	px, py, pz := w.entityBlitzPos(id)
	if sphere {
		r := hx
		if hy > r {
			r = hy
		}
		if hz > r {
			r = hz
		}
		w.phys3.AddSphereEx(id, px, py, pz, r, motion)
		e.collKind = 2
		e.radius = r
		e.boxX, e.boxY, e.boxZ = r, r, r
	} else {
		w.phys3.AddBoxEx(id, px, py, pz, hx, hy, hz, motion)
		e.collKind = 1
		e.boxX, e.boxY, e.boxZ = hx, hy, hz
		if e.radius < hx {
			e.radius = hx
		}
		if e.radius < hy {
			e.radius = hy
		}
		if e.radius < hz {
			e.radius = hz
		}
		if e.node != nil {
			wq := worldQuat(e.node.GetNode())
			w.phys3.SetRotation(id, wq.X, wq.Y, -wq.Z, wq.W)
		}
	}
	e.bodyType = motionBodyType(motion)
	if motion == phys3d.MotionTypeDynamic && mass > 0 {
		w.phys3.SetMass(id, mass)
	}
	if e.physMat != "" {
		_ = w.applyPhysicsMaterial(id, e.physMat)
	}
	return nil
}

func (w *World) entityBlitzPos(id int) (float32, float32, float32) {
	e := w.ents[id]
	if e == nil || e.node == nil {
		return 0, 0, 0
	}
	w.refreshWorldMatrices()
	p := worldPos(e.node.GetNode())
	return fromG3N(p.X, p.Y, p.Z)
}

func meshColliderExtents(e *Entity) (hx, hy, hz float32, sphere bool) {
	sphere = e.kind == "sphere"
	sx, sy, sz := float32(1), float32(1), float32(1)
	if e.node != nil {
		s := e.node.GetNode().Scale()
		sx, sy, sz = absf(s.X), absf(s.Y), absf(s.Z)
	}
	if e.mesh != nil && e.mesh.GetGeometry() != nil {
		bb := e.mesh.GetGeometry().BoundingBox()
		hx = (bb.Max.X - bb.Min.X) * 0.5 * sx
		hy = (bb.Max.Y - bb.Min.Y) * 0.5 * sy
		hz = (bb.Max.Z - bb.Min.Z) * 0.5 * sz
	} else {
		r := e.radius
		if r <= 0 {
			r = 1
		}
		hx, hy, hz = r*sx, r*sy, r*sz
	}
	if hx < 0.02 {
		hx = 0.02
	}
	if hy < 0.02 {
		hy = 0.02
	}
	if hz < 0.02 {
		hz = 0.02
	}
	return hx, hy, hz, sphere
}

func volumeMass(sphere bool, hx, hy, hz float32) float32 {
	var vol float64
	if sphere {
		r := float64(hx)
		if float64(hy) > r {
			r = float64(hy)
		}
		if float64(hz) > r {
			r = float64(hz)
		}
		vol = (4.0 / 3.0) * math.Pi * r * r * r
	} else {
		vol = float64(8 * hx * hy * hz)
	}
	if vol < 0.05 {
		vol = 0.05
	}
	if vol > 10000 {
		vol = 10000
	}
	return float32(vol)
}

func (w *World) applyPhysicsMaterial(id int, name string) error {
	e, err := w.ent(id)
	if err != nil {
		return err
	}
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "ball" || key == "rubberball" {
		key = "bouncy"
	}
	if key == "concrete" {
		key = "stone"
	}
	preset, ok := physicsMaterials[key]
	if !ok {
		return fmt.Errorf("SetPhysicsMaterial: unknown material %q", name)
	}
	e.physMat = key
	if e.bodyType == 0 {
		return nil
	}
	w.ensurePhys3()
	w.phys3.SetFriction(id, preset[0])
	w.phys3.SetRestitution(id, preset[1])
	w.phys3.SetLinearDamping(id, preset[2])
	w.phys3.SetAngularDamping(id, preset[3])
	return nil
}

func rayHitValue(h phys3d.RayHit, hit bool) value.Value {
	v := value.StructOf("RayHit", []string{"entity", "x", "y", "z", "nx", "ny", "nz", "fraction", "hit"})
	v.Fields["entity"] = value.Num(float64(h.ID))
	v.Fields["x"] = value.Num(float64(h.X))
	v.Fields["y"] = value.Num(float64(h.Y))
	v.Fields["z"] = value.Num(float64(h.Z))
	v.Fields["nx"] = value.Num(float64(h.NX))
	v.Fields["ny"] = value.Num(float64(h.NY))
	v.Fields["nz"] = value.Num(float64(h.NZ))
	v.Fields["fraction"] = value.Num(float64(h.Fraction))
	if hit {
		v.Fields["hit"] = value.Num(1)
	}
	return v
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
