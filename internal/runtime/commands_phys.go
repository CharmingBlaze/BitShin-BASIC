package runtime

import (
	"math"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/phys2d"
	"bitshinbasic/internal/phys3d"
	"bitshinbasic/internal/value"
)

func (w *World) physCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	blitzPos := func(id int) (float32, float32, float32) {
		e := w.ents[id]
		if e == nil {
			return 0, 0, 0
		}
		w.refreshWorldMatrices()
		p := worldPos(e.node.GetNode())
		return fromG3N(p.X, p.Y, p.Z)
	}
	return map[string]cmd{
		"physicsbackend": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Str(w.phys3.Backend()), nil
		}),
		"setgravity": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetGravity(float32(argN(a, 0, 0)), float32(argN(a, 1, -9.81)), float32(argN(a, 2, 0)))
			return z()
		}),
		"createbody": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			shape := argI(a, 1, 1)
			dyn := argI(a, 2, 1) != 0
			e, err := w.ent(id)
			if err != nil {
				return value.Value{}, err
			}
			r := e.radius
			if r <= 0 {
				r = 1
			}
			px, py, pz := blitzPos(id)
			if shape == 2 {
				w.phys3.AddBox(id, px, py, pz, r, r, r, dyn)
			} else {
				w.phys3.AddSphere(id, px, py, pz, r, dyn)
			}
			if dyn {
				e.bodyType = 1
			} else {
				e.bodyType = 2
			}
			return value.Num(float64(id)), nil
		}),
		"bodytype": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if len(a) >= 2 {
				e.bodyType = argI(a, 1, 0)
			}
			return value.Num(float64(e.bodyType)), nil
		}),
		"createbodysphere": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			r := float32(argN(a, 1, 1))
			motion, mass := bodyMotionMass(argN(a, 2, 1), argN(a, 3, 0))
			px, py, pz := blitzPos(id)
			w.phys3.AddSphereEx(id, px, py, pz, r, motion)
			if e := w.ents[id]; e != nil {
				e.bodyType = motionBodyType(motion)
			}
			if motion == phys3d.MotionTypeDynamic && mass > 0 {
				w.phys3.SetMass(id, mass)
			}
			return value.Num(float64(id)), nil
		}),
		"createbodybox": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			hx, hy, hz := float32(argN(a, 1, 1)), float32(argN(a, 2, 1)), float32(argN(a, 3, 1))
			motion, mass := bodyMotionMass(argN(a, 4, 1), argN(a, 5, 0))
			px, py, pz := blitzPos(id)
			w.phys3.AddBoxEx(id, px, py, pz, hx, hy, hz, motion)
			if e := w.ents[id]; e != nil {
				e.bodyType = motionBodyType(motion)
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
				wq := worldQuat(e.node.GetNode())
				w.phys3.SetRotation(id, wq.X, wq.Y, -wq.Z, wq.W)
			}
			if motion == phys3d.MotionTypeDynamic && mass > 0 {
				w.phys3.SetMass(id, mass)
			}
			return value.Num(float64(id)), nil
		}),
		"setmass": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetMass(argI(a, 0, 0), float32(argN(a, 1, 1)))
			return z()
		}),
		"setmotiontype": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			motion := argI(a, 1, phys3d.MotionTypeDynamic)
			if e := w.ents[id]; e != nil {
				e.bodyType = motionBodyType(motion)
			}
			return value.Num(float64(motion)), nil
		}),
		"setbodyvelocity": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetVelocity(argI(a, 0, 0), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			return z()
		}),
		"setlinearvelocity": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetVelocity(argI(a, 0, 0), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			return z()
		}),
		"bodyvelocityx": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			x, _, _, _ := w.phys3.GetVelocity(argI(a, 0, 0))
			return value.Num(float64(x)), nil
		}),
		"bodyvelocityy": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			_, y, _, _ := w.phys3.GetVelocity(argI(a, 0, 0))
			return value.Num(float64(y)), nil
		}),
		"bodyvelocityz": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			_, _, z, _ := w.phys3.GetVelocity(argI(a, 0, 0))
			return value.Num(float64(z)), nil
		}),
		"applyimpulse": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.ApplyImpulse(argI(a, 0, 0), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			return z()
		}),
		"physics2d": n(func(a []value.Value) (value.Value, error) {
			if w.phys2 == nil {
				w.phys2 = phys2d.New()
			}
			return z()
		}),
		"gravity2d": n(func(a []value.Value) (value.Value, error) {
			if w.phys2 == nil {
				w.phys2 = phys2d.New()
			}
			w.phys2.SetGravity(argN(a, 0, 0), argN(a, 1, -200))
			return z()
		}),
		"createcircle2d": n(func(a []value.Value) (value.Value, error) {
			if w.phys2 == nil {
				w.phys2 = phys2d.New()
			}
			id := argI(a, 0, 0)
			r := argN(a, 1, 1)
			mass := argN(a, 2, 1)
			dyn := argI(a, 3, 1) != 0
			x, y := w.pos2D(id)
			w.phys2.AddCircle(id, x, y, r, mass, dyn)
			return z()
		}),
		"createbox2d": n(func(a []value.Value) (value.Value, error) {
			if w.phys2 == nil {
				w.phys2 = phys2d.New()
			}
			id := argI(a, 0, 0)
			ww, hh := argN(a, 1, 2), argN(a, 2, 2)
			mass := argN(a, 3, 1)
			dyn := argI(a, 4, 1) != 0
			x, y := w.pos2D(id)
			w.phys2.AddBox(id, x, y, ww, hh, mass, dyn)
			return z()
		}),
		"setvelocity2d": need(func(a []value.Value) (value.Value, error) {
			if w.phys2 != nil {
				w.phys2.SetVelocity(argI(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0))
			}
			return z()
		}),
		"applyimpulse2d": need(func(a []value.Value) (value.Value, error) {
			if w.phys2 != nil {
				w.phys2.ApplyImpulse(argI(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0))
			}
			return z()
		}),
		"entityangle2d": need(func(a []value.Value) (value.Value, error) {
			if w.phys2 == nil {
				return value.Num(0), nil
			}
			return value.Num(w.phys2.Angle(argI(a, 0, 0))), nil
		}),
		"applyforce": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.ApplyForce(argI(a, 0, 0), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			return z()
		}),
		"setbodyangularvelocity": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetAngularVelocity(argI(a, 0, 0), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			return z()
		}),
		"bodyvelocity": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			x, y, z, _ := w.phys3.GetVelocity(argI(a, 0, 0))
			return value.Num(float64(x*x + y*y + z*z)), nil
		}),
		"setbodymass": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetMass(argI(a, 0, 0), float32(argN(a, 1, 1)))
			return z()
		}),
		"createbodycapsule": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			px, py, pz := blitzPos(id)
			massOrDyn := argN(a, 3, 1)
			w.phys3.AddCapsule(id, px, py, pz, float32(argN(a, 1, 1)), float32(argN(a, 2, 0.4)), massOrDyn != 0)
			if massOrDyn > 0 {
				w.phys3.SetMass(id, float32(massOrDyn))
			}
			return value.Num(float64(id)), nil
		}),
		"bodycapsule": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			px, py, pz := blitzPos(id)
			massOrDyn := argN(a, 3, 1)
			w.phys3.AddCapsule(id, px, py, pz, float32(argN(a, 1, 1)), float32(argN(a, 2, 0.4)), massOrDyn != 0)
			if massOrDyn > 0 {
				w.phys3.SetMass(id, float32(massOrDyn))
			}
			return value.Num(float64(id)), nil
		}),
		"bodybox": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			px, py, pz := blitzPos(id)
			w.phys3.AddBox(id, px, py, pz, float32(argN(a, 1, 1)), float32(argN(a, 2, 1)), float32(argN(a, 3, 1)), argI(a, 4, 1) != 0)
			return z()
		}),
		"bodysphere": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			px, py, pz := blitzPos(id)
			w.phys3.AddSphere(id, px, py, pz, float32(argN(a, 1, 1)), argI(a, 2, 1) != 0)
			return z()
		}),
		"groundplane": need(func(a []value.Value) (value.Value, error) {
			y := argN(a, 0, 0)
			ground := w.meshEnt(nilGeomPlane(), 0)
			if e := w.ents[ground]; e != nil {
				e.node.GetNode().SetPosition(0, float32(y), 0)
			}
			w.ensurePhys3()
			w.phys3.AddGround(ground, float32(y))
			if e := w.ents[ground]; e != nil {
				e.bodyType = 2
			}
			return value.Num(float64(ground)), nil
		}),
		"linepick": need(func(a []value.Value) (value.Value, error) {
			return w.linePick(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0), argN(a, 4, 0), argN(a, 5, 0), argN(a, 6, 0))
		}),
		"raypick": need(func(a []value.Value) (value.Value, error) {
			return w.linePick(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0), argN(a, 4, 0), argN(a, 5, 0), argN(a, 6, 0))
		}),
		"camerapick": need(func(a []value.Value) (value.Value, error) {
			return w.cameraPick(a)
		}),
		"pickedentity": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.pickID)), nil }),
		"pickedx":      n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.pickX)), nil }),
		"pickedy":      n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.pickY)), nil }),
		"pickedz":      n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.pickZ)), nil }),
		"entitybox": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			e.boxX = float32(argN(a, 1, 1))
			e.boxY = float32(argN(a, 2, 1))
			e.boxZ = float32(argN(a, 3, 1))
			if e.boxX > e.radius {
				e.radius = e.boxX
			}
			return z()
		}),
		"setgravity2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			w.phys2.SetGravity(argN(a, 0, 0), argN(a, 1, 900))
			return z()
		}),
		"setstatic2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			w.phys2.SetStatic(argI(a, 0, 0), argI(a, 1, 1) != 0)
			return z()
		}),
		"setmass2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			w.phys2.SetMass(argI(a, 0, 0), argN(a, 1, 1))
			return z()
		}),
		"velocity2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			if len(a) >= 3 {
				w.phys2.SetVelocity(argI(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0))
				return z()
			}
			x, y, _ := w.phys2.GetVelocity(argI(a, 0, 0))
			return value.Num(x*x + y*y), nil
		}),
		"position2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			if len(a) >= 3 {
				w.phys2.SetPosition(argI(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0))
				if s := w.sprites[argI(a, 0, 0)]; s != nil {
					s.x, s.y = argN(a, 1, 0), argN(a, 2, 0)
				}
				return z()
			}
			x, _, _ := w.phys2.GetPosition(argI(a, 0, 0))
			return value.Num(x), nil
		}),
		"collides2d": n(func(a []value.Value) (value.Value, error) {
			if w.phys2 == nil {
				return value.Num(0), nil
			}
			if w.phys2.Collides(argI(a, 0, 0), argI(a, 1, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"countcollisions2d": n(func(a []value.Value) (value.Value, error) {
			if w.phys2 == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(w.phys2.CountHits(argI(a, 0, 0)))), nil
		}),
		"raycast2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			id, x, y, ok := w.phys2.Raycast(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0))
			if !ok {
				return value.Num(0), nil
			}
			w.pickID, w.pickX, w.pickY = id, float32(x), float32(y)
			return value.Num(float64(id)), nil
		}),
		"updateworld2d": n(func(a []value.Value) (value.Value, error) {
			w.updateWorld()
			return z()
		}),
		"body2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			id := argI(a, 0, 0)
			kind := argI(a, 1, 1)
			x, y := w.pos2D(id)
			if kind == 2 {
				w.phys2.AddBox(id, x, y, argN(a, 2, 32), argN(a, 3, 32), argN(a, 4, 1), argI(a, 5, 1) != 0)
			} else {
				w.phys2.AddCircle(id, x, y, argN(a, 2, 16), argN(a, 3, 1), argI(a, 4, 1) != 0)
			}
			return z()
		}),
		"applyforce2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			w.phys2.ApplyForce(argI(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0))
			return z()
		}),
		"velocity2dx": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			x, _, _ := w.phys2.GetVelocity(argI(a, 0, 0))
			return value.Num(x), nil
		}),
		"velocity2dy": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			_, y, _ := w.phys2.GetVelocity(argI(a, 0, 0))
			return value.Num(y), nil
		}),
		"position2dy": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			_, y, _ := w.phys2.GetPosition(argI(a, 0, 0))
			return value.Num(y), nil
		}),
		"createpin2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			id := w.phys2.AddPin(argI(a, 0, 0), argI(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0), argN(a, 4, 0), argN(a, 5, 0))
			return value.Num(float64(id)), nil
		}),
		"createspring2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			id := w.phys2.AddSpring(argI(a, 0, 0), argI(a, 1, 0), argN(a, 2, 0), argN(a, 3, 80), argN(a, 4, 8), argN(a, 5, 0), argN(a, 6, 0), argN(a, 7, 0), argN(a, 8, 0))
			return value.Num(float64(id)), nil
		}),
		"createslide2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			id := w.phys2.AddSlide(argI(a, 0, 0), argI(a, 1, 0), argN(a, 2, 0), argN(a, 3, 8), argN(a, 4, 0), argN(a, 5, 0), argN(a, 6, 0), argN(a, 7, 0))
			return value.Num(float64(id)), nil
		}),
		"createjoint2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			kind := argI(a, 0, 1)
			aa, bb := argI(a, 1, 0), argI(a, 2, 0)
			id := 0
			switch kind {
			case 2:
				id = w.phys2.AddSpring(aa, bb, argN(a, 3, 0), argN(a, 4, 80), argN(a, 5, 8), 0, 0, 0, 0)
			case 3:
				id = w.phys2.AddSlide(aa, bb, argN(a, 3, 0), argN(a, 4, 8), 0, 0, 0, 0)
			default:
				id = w.phys2.AddPin(aa, bb, argN(a, 3, 0), argN(a, 4, 0), argN(a, 5, 0), argN(a, 6, 0))
			}
			return value.Num(float64(id)), nil
		}),
		"freejoint2d": n(func(a []value.Value) (value.Value, error) {
			if w.phys2 != nil {
				w.phys2.RemoveJoint(argI(a, 0, 0))
			}
			return z()
		}),
		"deletejoint2d": n(func(a []value.Value) (value.Value, error) {
			if w.phys2 != nil {
				w.phys2.RemoveJoint(argI(a, 0, 0))
			}
			return z()
		}),
		"raycast": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id, x, y, z, ok := w.phys3.Raycast(
				float32(argN(a, 0, 0)), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)),
				float32(argN(a, 3, 0)), float32(argN(a, 4, 0)), float32(argN(a, 5, 0)),
			)
			if !ok {
				w.pickID = 0
				return value.Num(0), nil
			}
			w.pickID, w.pickX, w.pickY, w.pickZ = id, x, y, z
			return value.Num(float64(id)), nil
		}),
		"getgravity": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			x, y, z := w.phys3.GetGravity()
			if len(a) > 0 {
				switch argI(a, 0, 1) {
				case 2:
					return value.Num(float64(y)), nil
				case 3:
					return value.Num(float64(z)), nil
				}
			}
			return value.Num(float64(x)), nil
		}),
		"getgravityx": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			x, _, _ := w.phys3.GetGravity()
			return value.Num(float64(x)), nil
		}),
		"getgravityy": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			_, y, _ := w.phys3.GetGravity()
			return value.Num(float64(y)), nil
		}),
		"getgravityz": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			_, _, z := w.phys3.GetGravity()
			return value.Num(float64(z)), nil
		}),
		"createcharacter": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			px, py, pz := blitzPos(id)
			w.phys3.AddCharacter(id, px, py, pz, float32(argN(a, 1, 0.9)), float32(argN(a, 2, 0.4)))
			if e := w.ents[id]; e != nil {
				e.bodyType = 3
			}
			return z()
		}),
		"setcharactervelocity": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetVelocity(argI(a, 0, 0), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			return z()
		}),
		"movecharacter": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			if len(a) >= 4 {
				w.phys3.SetVelocity(id, float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
				return z()
			}
			_, vy, _, _ := w.phys3.GetVelocity(id)
			w.phys3.SetVelocity(id, float32(argN(a, 1, 0)), vy, float32(argN(a, 2, 0)))
			return z()
		}),
		"characteronground": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Num(float64(w.phys3.CharacterGround(argI(a, 0, 0)))), nil
		}),
		"physicscharacter": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Str(w.phys3.CharacterBackend()), nil
		}),
		"getphysicscharacter": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Str(w.phys3.CharacterBackend()), nil
		}),
		"createhinge": n(func(a []value.Value) (value.Value, error) {
			if len(a) < 2 {
				return value.Num(0), nil
			}
			w.ensurePhys3()
			id := w.phys3.CreateHingeJoint(argI(a, 0, 0), argI(a, 1, 0),
				float32(argN(a, 2, 0)), float32(argN(a, 3, 0)), float32(argN(a, 4, 0)),
				float32(argN(a, 5, 0)), float32(argN(a, 6, 1)), float32(argN(a, 7, 0)))
			return value.Num(float64(id)), nil
		}),
		"createhingejoint": n(func(a []value.Value) (value.Value, error) {
			if len(a) < 2 {
				return value.Num(0), nil
			}
			w.ensurePhys3()
			id := w.phys3.CreateHingeJoint(argI(a, 0, 0), argI(a, 1, 0),
				float32(argN(a, 2, 0)), float32(argN(a, 3, 0)), float32(argN(a, 4, 0)),
				float32(argN(a, 5, 0)), float32(argN(a, 6, 1)), float32(argN(a, 7, 0)))
			return value.Num(float64(id)), nil
		}),
		"createpointjoint": n(func(a []value.Value) (value.Value, error) {
			if len(a) < 2 {
				return value.Num(0), nil
			}
			w.ensurePhys3()
			id := w.phys3.CreatePointJoint(argI(a, 0, 0), argI(a, 1, 0),
				float32(argN(a, 2, 0)), float32(argN(a, 3, 0)), float32(argN(a, 4, 0)))
			return value.Num(float64(id)), nil
		}),
		"createballsocketjoint": n(func(a []value.Value) (value.Value, error) {
			if len(a) < 2 {
				return value.Num(0), nil
			}
			w.ensurePhys3()
			id := w.phys3.CreatePointJoint(argI(a, 0, 0), argI(a, 1, 0),
				float32(argN(a, 2, 0)), float32(argN(a, 3, 0)), float32(argN(a, 4, 0)))
			return value.Num(float64(id)), nil
		}),
		"createsliderjoint": n(func(a []value.Value) (value.Value, error) {
			if len(a) < 2 {
				return value.Num(0), nil
			}
			w.ensurePhys3()
			id := w.phys3.CreateSliderJoint(argI(a, 0, 0), argI(a, 1, 0),
				float32(argN(a, 2, 0)), float32(argN(a, 3, 0)), float32(argN(a, 4, 0)),
				float32(argN(a, 5, 1)), float32(argN(a, 6, 0)), float32(argN(a, 7, 0)))
			return value.Num(float64(id)), nil
		}),
		"createspringjoint": n(func(a []value.Value) (value.Value, error) {
			if len(a) < 2 {
				return value.Num(0), nil
			}
			w.ensurePhys3()
			id := w.phys3.CreateSpringJoint(argI(a, 0, 0), argI(a, 1, 0),
				float32(argN(a, 2, 0)), float32(argN(a, 3, 0)), float32(argN(a, 4, 0)),
				float32(argN(a, 5, 1)), float32(argN(a, 6, 8)), float32(argN(a, 7, 1)))
			return value.Num(float64(id)), nil
		}),
		"createjoint": n(func(a []value.Value) (value.Value, error) {
			if len(a) < 3 {
				return value.Num(0), nil
			}
			w.ensurePhys3()
			kind := argI(a, 0, 1)
			aa, bb := argI(a, 1, 0), argI(a, 2, 0)
			px, py, pz := float32(argN(a, 3, 0)), float32(argN(a, 4, 0)), float32(argN(a, 5, 0))
			id := 0
			switch kind {
			case phys3d.JointPoint:
				id = w.phys3.CreatePointJoint(aa, bb, px, py, pz)
			case phys3d.JointSlider:
				id = w.phys3.CreateSliderJoint(aa, bb, px, py, pz, float32(argN(a, 6, 1)), float32(argN(a, 7, 0)), float32(argN(a, 8, 0)))
			case phys3d.JointSpring:
				id = w.phys3.CreateSpringJoint(aa, bb, px, py, pz, float32(argN(a, 6, 1)), float32(argN(a, 7, 8)), float32(argN(a, 8, 1)))
			default:
				id = w.phys3.CreateHingeJoint(aa, bb, px, py, pz, float32(argN(a, 6, 0)), float32(argN(a, 7, 1)), float32(argN(a, 8, 0)))
			}
			return value.Num(float64(id)), nil
		}),
		"createhinge3d": n(func(a []value.Value) (value.Value, error) {
			if len(a) < 2 {
				return value.Num(0), nil
			}
			w.ensurePhys3()
			id := w.phys3.CreateHingeJoint(argI(a, 0, 0), argI(a, 1, 0),
				float32(argN(a, 2, 0)), float32(argN(a, 3, 0)), float32(argN(a, 4, 0)),
				float32(argN(a, 5, 0)), float32(argN(a, 6, 1)), float32(argN(a, 7, 0)))
			return value.Num(float64(id)), nil
		}),
		"freejoint": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.RemoveJoint(argI(a, 0, 0))
			return z()
		}),
		"sethingelimits": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetHingeLimits(argI(a, 0, 0), float32(argN(a, 1, -10)), float32(argN(a, 2, 95)))
			return z()
		}),
		"sethingefriction": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetHingeFriction(argI(a, 0, 0), float32(argN(a, 1, 15)))
			return z()
		}),
		"sethingemotor": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetHingeMotor(argI(a, 0, 0), float32(argN(a, 1, 0)), float32(argN(a, 2, 20)))
			return z()
		}),
		"disablebodycollision": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.DisableBodyCollision(argI(a, 0, 0), argI(a, 1, 0))
			return z()
		}),
		"activatebody": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.Wake(argI(a, 0, 0))
			return z()
		}),
		"setbodyrotation": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			pitch, yaw, roll := float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0))
			if e := w.ents[id]; e != nil {
				e.pitch, e.yaw, e.roll = pitch, yaw, roll
				w.applyRot(e)
			}
			qx, qy, qz, qw := eulerToQuat(pitch, yaw, roll)
			w.phys3.SetRotation(id, qx, qy, qz, qw)
			return z()
		}),
		"getbodypitch": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.pitch)), nil
		}),
		"getbodyyaw": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.yaw)), nil
		}),
		"getbodyroll": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.roll)), nil
		}),
		"joint_hinge":      n(func(a []value.Value) (value.Value, error) { return value.Num(float64(phys3d.JointHinge)), nil }),
		"joint_point":      n(func(a []value.Value) (value.Value, error) { return value.Num(float64(phys3d.JointPoint)), nil }),
		"joint_slider":     n(func(a []value.Value) (value.Value, error) { return value.Num(float64(phys3d.JointSlider)), nil }),
		"joint_spring":     n(func(a []value.Value) (value.Value, error) { return value.Num(float64(phys3d.JointSpring)), nil }),
		"motion_static":    n(func(a []value.Value) (value.Value, error) { return value.Num(float64(phys3d.MotionTypeStatic)), nil }),
		"motion_kinematic": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(phys3d.MotionTypeKinematic)), nil }),
		"motion_dynamic":   n(func(a []value.Value) (value.Value, error) { return value.Num(float64(phys3d.MotionTypeDynamic)), nil }),
		"createpoly2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			id := argI(a, 0, 0)
			x, y := w.pos2D(id)
			verts := [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}}
			if len(a) >= 9 {
				verts = [][2]float64{
					{argN(a, 1, -1), argN(a, 2, -1)},
					{argN(a, 3, 1), argN(a, 4, -1)},
					{argN(a, 5, 1), argN(a, 6, 1)},
					{argN(a, 7, -1), argN(a, 8, 1)},
				}
			}
			w.phys2.AddPoly(id, x, y, verts, argN(a, 9, 1), argI(a, 10, 1) != 0)
			return value.Num(float64(id)), nil
		}),
		"setccd2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			w.phys2.SetCCD(argI(a, 0, 1) != 0)
			return z()
		}),
		"bodyangularvelocityx": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			x, _, _, _ := w.phys3.GetAngularVelocity(argI(a, 0, 0))
			return value.Num(float64(x)), nil
		}),
		"bodyangularvelocityy": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			_, y, _, _ := w.phys3.GetAngularVelocity(argI(a, 0, 0))
			return value.Num(float64(y)), nil
		}),
		"bodyangularvelocityz": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			_, _, z, _ := w.phys3.GetAngularVelocity(argI(a, 0, 0))
			return value.Num(float64(z)), nil
		}),
	}
}

// bodyMotionMass maps CreateBody* trailing args: 0 static, 1 kinematic, else dynamic.
// massHint is used when the motion slot is a type (0/1/2) and a separate mass follows.
func bodyMotionMass(motionOrMass, massHint float64) (motion int, mass float32) {
	v := int(motionOrMass)
	switch v {
	case phys3d.MotionTypeStatic:
		return phys3d.MotionTypeStatic, 0
	case phys3d.MotionTypeKinematic:
		return phys3d.MotionTypeKinematic, 0
	default:
		mass = float32(motionOrMass)
		if massHint > 0 {
			mass = float32(massHint)
		}
		if mass <= 0 {
			mass = 1
		}
		return phys3d.MotionTypeDynamic, mass
	}
}

func motionBodyType(motion int) int {
	switch motion {
	case phys3d.MotionTypeStatic:
		return 2
	case phys3d.MotionTypeKinematic:
		return 3
	default:
		return 1
	}
}

func eulerToQuat(pitch, yaw, roll float32) (float32, float32, float32, float32) {
	px := pitch * math32.Pi / 180
	py := -yaw * math32.Pi / 180
	pz := -roll * math32.Pi / 180
	cx, sx := float32(math.Cos(float64(px)*0.5)), float32(math.Sin(float64(px)*0.5))
	cy, sy := float32(math.Cos(float64(py)*0.5)), float32(math.Sin(float64(py)*0.5))
	cz, sz := float32(math.Cos(float64(pz)*0.5)), float32(math.Sin(float64(pz)*0.5))
	// XYZ intrinsic matching G3N SetRotationX/Y/Z
	x := sx*cy*cz + cx*sy*sz
	y := cx*sy*cz - sx*cy*sz
	z := cx*cy*sz + sx*sy*cz
	w := cx*cy*cz - sx*sy*sz
	return x, y, z, w
}

func nilGeomPlane() *geometry.Geometry {
	return geometry.NewPlane(40, 40)
}

func (w *World) cameraPick(a []value.Value) (value.Value, error) {
	cam := w.cam
	sx, sy := float64(w.mx), float64(w.my)
	switch {
	case len(a) >= 3:
		if e, err := w.ent(argI(a, 0, 0)); err == nil && e.cam != nil {
			cam = e.cam
		}
		sx, sy = argN(a, 1, sx), argN(a, 2, sy)
	case len(a) == 2:
		sx, sy = argN(a, 0, sx), argN(a, 1, sy)
	case len(a) == 1:
		if e, err := w.ent(argI(a, 0, 0)); err == nil && e.cam != nil {
			cam = e.cam
		}
	}
	if cam == nil {
		return value.Num(0), nil
	}
	ww, hh := float64(w.scrW), float64(w.scrH)
	if ww < 1 {
		ww = 800
	}
	if hh < 1 {
		hh = 600
	}
	nx := float32(sx/ww)*2 - 1
	ny := 1 - float32(sy/hh)*2
	p0 := math32.Vector3{nx, ny, -1}
	p1 := math32.Vector3{nx, ny, 1}
	cam.Unproject(&p0)
	cam.Unproject(&p1)
	bx, by, bz := fromG3N(p0.X, p0.Y, p0.Z)
	ex, ey, ez := fromG3N(p1.X, p1.Y, p1.Z)
	return w.linePick(float64(bx), float64(by), float64(bz), float64(ex-bx), float64(ey-by), float64(ez-bz), 0)
}

func (w *World) linePick(x, y, z, dx, dy, dz, rng float64) (value.Value, error) {
	w.pickID = 0
	best := 1e9
	llen := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if llen < 1e-8 {
		return value.Num(0), nil
	}
	ndx, ndy, ndz := dx/llen, dy/llen, dz/llen
	maxT := llen
	if rng > 0 {
		maxT = rng
	}
	if w.phys3 != nil {
		if id, px, py, pz, ok := w.phys3.Raycast(float32(x), float32(y), float32(z), float32(ndx*maxT), float32(ndy*maxT), float32(ndz*maxT)); ok {
			w.pickID, w.pickX, w.pickY, w.pickZ = id, px, py, pz
			if id != 0 {
				return value.Num(float64(id)), nil
			}
		}
	}
	for id, e := range w.ents {
		if e.cam != nil || e.node == nil {
			continue
		}
		p := worldPos(e.node.GetNode())
		ex, ey, ez := fromG3N(p.X, p.Y, p.Z)
		ox, oy, oz := float32(x), float32(y), float32(z)
		t := float64((ex-ox)*float32(ndx) + (ey-oy)*float32(ndy) + (ez-oz)*float32(ndz))
		if t < 0 || t > maxT {
			continue
		}
		px := ox + float32(ndx*t)
		py := oy + float32(ndy*t)
		pz := oz + float32(ndz*t)
		dist := (px-ex)*(px-ex) + (py-ey)*(py-ey) + (pz-ez)*(pz-ez)
		r := e.radius
		if r <= 0 {
			r = 1
		}
		if dist <= r*r && t < best {
			best = t
			w.pickID, w.pickX, w.pickY, w.pickZ = id, px, py, pz
		}
	}
	return value.Num(float64(w.pickID)), nil
}
