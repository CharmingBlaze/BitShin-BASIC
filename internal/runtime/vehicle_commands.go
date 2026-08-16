package runtime

import (
	"math"

	"bitshinbasic/internal/value"
)

type vehCtrl struct {
	kind      string
	id        int
	hx, hy, hz float32
	native    int
	mass      float32
	area      float32
	rho       float32
	thrustMax float32
	liftCl    float32
	dragCd    float32
	stall     float32
}

func quatRotate(qx, qy, qz, qw, vx, vy, vz float32) (float32, float32, float32) {
	ix := qw*vx + qy*vz - qz*vy
	iy := qw*vy + qz*vx - qx*vz
	iz := qw*vz + qx*vy - qy*vx
	iw := -qx*vx - qy*vy - qz*vz
	return ix*qw + iw*-qx + iy*-qz - iz*-qy,
		iy*qw + iw*-qy + iz*-qx - ix*-qz,
		iz*qw + iw*-qz + ix*-qy - iy*-qx
}

func (w *World) physAxes(id int) (fx, fy, fz, ux, uy, uz, rx, ry, rz float32) {
	qx, qy, qz, qw, ok := w.phys3.GetRotation(id)
	if !ok {
		return 0, 0, 1, 0, 1, 0, 1, 0, 0
	}
	fx, fy, fz = quatRotate(qx, qy, qz, qw, 0, 0, 1)
	ux, uy, uz = quatRotate(qx, qy, qz, qw, 0, 1, 0)
	rx, ry, rz = quatRotate(qx, qy, qz, qw, 1, 0, 0)
	return fx, fy, fz, ux, uy, uz, rx, ry, rz
}

func (w *World) hullOf(id int, dx, dy, dz float32) (float32, float32, float32) {
	e := w.ents[id]
	if e == nil || e.node == nil {
		return dx, dy, dz
	}
	s := e.node.GetNode().Scale()
	if s.X > 0.05 {
		dx = s.X
	}
	if s.Y > 0.05 {
		dy = s.Y
	}
	if s.Z > 0.05 {
		dz = s.Z
	}
	return dx, dy, dz
}

func (w *World) ensureCtrlBody(id int, hx, hy, hz, mass float32) {
	w.ensurePhys3()
	if _, _, _, ok := w.phys3.GetPosition(id); ok {
		w.phys3.SetMass(id, mass)
		return
	}
	px, py, pz := float32(0), float32(0), float32(0)
	if e := w.ents[id]; e != nil && e.node != nil {
		p := e.node.GetNode().Position()
		px, py, pz = fromG3N(p.X, p.Y, p.Z)
	}
	w.phys3.AddBox(id, px, py, pz, hx, hy, hz, true)
	w.phys3.SetMass(id, mass)
	if e := w.ents[id]; e != nil {
		e.bodyType = 1
	}
}

func (w *World) putCtrl(c *vehCtrl) {
	if w.vehCtrls == nil {
		w.vehCtrls = map[int]*vehCtrl{}
	}
	w.vehCtrls[c.id] = c
}

func (w *World) bindCtrl(kind string, a []value.Value, hx, hy, hz, mass float32) *vehCtrl {
	id := argI(a, 0, 0)
	hx, hy, hz = w.hullOf(id, hx, hy, hz)
	if len(a) >= 4 {
		hx = float32(argN(a, 1, float64(hx)))
		hy = float32(argN(a, 2, float64(hy)))
		hz = float32(argN(a, 3, float64(hz)))
	}
	boxHy := hy
	if kind == "car" || kind == "moto" || kind == "tank" {
		// Keep the collider above the tires so the hull does not rest on
		// the ground and unload the Jolt wheels.
		if boxHy > 0.18 {
			boxHy = 0.18
		}
	}
	w.ensureCtrlBody(id, hx, boxHy, hz, mass)
	if kind == "car" || kind == "moto" || kind == "tank" {
		w.phys3.SetFriction(id, 0.35)
	}
	c := &vehCtrl{kind: kind, id: id, hx: hx, hy: hy, hz: hz, mass: mass, rho: 1.2, area: hx * hz * 2}
	w.putCtrl(c)
	return c
}

func (w *World) applyAero(c *vehCtrl, throttle, pitch, roll, yaw, thrustScale, liftScale float32) {
	fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(c.id)
	vx, vy, vz, _ := w.phys3.GetVelocity(c.id)
	speed := float32(math.Sqrt(float64(vx*vx + vy*vy + vz*vz)))
	q := 0.5 * c.rho * speed * speed
	cl := c.liftCl
	if c.stall > 0 && speed < c.stall {
		cl *= speed / c.stall
	}
	lift := q * c.area * cl * liftScale
	drag := q * c.area * c.dragCd
	thrust := throttle * c.thrustMax * thrustScale
	px, py, pz, _ := w.phys3.GetPosition(c.id)
	wing := c.hx
	if wing < 0.4 {
		wing = 0.4
	}
	w.phys3.ApplyForceAtPosition(c.id, ux*lift*0.5, uy*lift*0.5, uz*lift*0.5, px+rx*wing, py, pz+rz*wing)
	w.phys3.ApplyForceAtPosition(c.id, ux*lift*0.5, uy*lift*0.5, uz*lift*0.5, px-rx*wing, py, pz-rz*wing)
	w.phys3.ApplyLocalImpulse(c.id, 0, 0, thrust*0.016)
	if speed > 0.5 {
		inv := drag / speed
		w.phys3.ApplyForce(c.id, -vx*inv, -vy*inv, -vz*inv)
	}
	w.phys3.ApplyTorque(c.id, rx*pitch*8000+ux*yaw*6000+fx*roll*7000, ry*pitch*8000+uy*yaw*6000+fy*roll*7000, rz*pitch*8000+uz*yaw*6000+fz*roll*7000)
}

func (w *World) applyBoatForces(c *vehCtrl, throttle, steer float32) {
	x, y, z, ok := w.phys3.GetPosition(c.id)
	if !ok {
		return
	}
	fx, _, fz, _, uy, _, rx, _, rz := w.physAxes(c.id)
	w.phys3.SetLinearDamping(c.id, 1.8)
	corners := [4][2]float32{{c.hx, c.hz}, {-c.hx, c.hz}, {c.hx, -c.hz}, {-c.hx, -c.hz}}
	for i := 0; i < 4; i++ {
		px := x + rx*corners[i][0] + fx*corners[i][1]
		pz := z + rz*corners[i][0] + fz*corners[i][1]
		wh := w.waterHeight(px, pz)
		depth := wh - (y - c.hy*0.4)
		if depth > 0 {
			if depth > c.hy*2 {
				depth = c.hy * 2
			}
			buoy := depth * c.mass * 6
			w.phys3.ApplyForceAtPosition(c.id, 0, buoy, 0, px, y-c.hy*0.3, pz)
		}
	}
	w.phys3.ApplyForce(c.id, fx*throttle*c.thrustMax, uy*throttle*c.thrustMax*0.02, fz*throttle*c.thrustMax)
	w.phys3.ApplyTorque(c.id, 0, steer*4500, 0)
}

func (w *World) vehicleCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"applytorque": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.ApplyTorque(argI(a, 0, 0), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			return z()
		}),
		"applyforceatposition": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.ApplyForceAtPosition(argI(a, 0, 0),
				float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)),
				float32(argN(a, 4, 0)), float32(argN(a, 5, 0)), float32(argN(a, 6, 0)))
			return z()
		}),
		"applylocalimpulse": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.ApplyLocalImpulse(argI(a, 0, 0), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			return z()
		}),
		"setgravityscale": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetGravityScale(argI(a, 0, 0), float32(argN(a, 1, 1)))
			return z()
		}),
		"setrestitution": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetRestitution(argI(a, 0, 0), float32(argN(a, 1, 0)))
			return z()
		}),
		"setlineardamping": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetLinearDamping(argI(a, 0, 0), float32(argN(a, 1, 0)))
			return z()
		}),
		"setangulardamping": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetAngularDamping(argI(a, 0, 0), float32(argN(a, 1, 0)))
			return z()
		}),
		"setfriction": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetFriction(argI(a, 0, 0), float32(argN(a, 1, 0.5)))
			return z()
		}),
		"setmass": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetMass(argI(a, 0, 0), float32(argN(a, 1, 1)))
			return z()
		}),
		"applybuoyancy": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			x, y, pz, ok := w.phys3.GetPosition(id)
			if !ok {
				return z()
			}
			waterY := float32(argN(a, 1, float64(w.waterHeight(x, pz))))
			scale := float32(argN(a, 2, 1))
			depth := waterY - y
			if depth > 0 {
				w.phys3.ApplyForce(id, 0, depth*scale*400, 0)
			}
			return z()
		}),
		"extendedupdate": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return z()
		}),
		"characterextendedupdate": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return z()
		}),
		"charactergroundnormalx": n(func(a []value.Value) (value.Value, error) {
			if w.phys3 == nil {
				return value.Num(0), nil
			}
			nx, _, _ := w.phys3.CharacterGroundNormal(argI(a, 0, 0))
			return value.Num(float64(nx)), nil
		}),
		"charactergroundnormaly": n(func(a []value.Value) (value.Value, error) {
			if w.phys3 == nil {
				return value.Num(0), nil
			}
			_, ny, _ := w.phys3.CharacterGroundNormal(argI(a, 0, 0))
			return value.Num(float64(ny)), nil
		}),
		"charactergroundnormalz": n(func(a []value.Value) (value.Value, error) {
			if w.phys3 == nil {
				return value.Num(0), nil
			}
			_, _, nz := w.phys3.CharacterGroundNormal(argI(a, 0, 0))
			return value.Num(float64(nz)), nil
		}),
		"createvehicle": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("car", a, 0.9, 0.35, 1.8, 1200)
			c.native = w.phys3.CreateWheeledVehicle(c.id, c.hx, c.hy, c.hz)
			return value.Num(float64(c.id)), nil
		}),
		"createcarcontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("car", a, 0.9, 0.35, 1.8, 1200)
			c.native = w.phys3.CreateWheeledVehicle(c.id, c.hx, c.hy, c.hz)
			return value.Num(float64(c.id)), nil
		}),
		"setvehicleinput": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			steer := float32(argN(a, 1, 0))
			throttle := float32(argN(a, 2, 0))
			brake := float32(argN(a, 3, 0))
			w.phys3.SetVehicleInput(id, steer, throttle, brake)
			if c := w.vehCtrls[id]; c != nil && (c.kind == "car" || c.kind == "moto" || c.kind == "tank") {
				w.updateWheeled(c, steer, throttle, brake)
			}
			return z()
		}),
		"updatecar": need(func(a []value.Value) (value.Value, error) {
			return w.updateNamed("car", a)
		}),
		"updatevehicle": need(func(a []value.Value) (value.Value, error) {
			return w.updateNamed("car", a)
		}),
		"createplanecontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("plane", a, 4, 0.4, 3, 900)
			c.thrustMax, c.liftCl, c.dragCd, c.stall = 18000, 0.9, 0.08, 12
			w.phys3.CreatePlaneController(c.id)
			return value.Num(float64(c.id)), nil
		}),
		"updateplane": need(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			w.phys3.UpdatePlane(id, float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)), float32(argN(a, 4, 0)))
			return z()
		}),
		"createjetcontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("jet", a, 3.2, 0.35, 4.5, 1100)
			c.thrustMax, c.liftCl, c.dragCd, c.stall = 42000, 0.55, 0.05, 28
			return value.Num(float64(c.id)), nil
		}),
		"updatejet": need(func(a []value.Value) (value.Value, error) {
			return w.updateAero("jet", a, 1.6, 0.7)
		}),
		"createspaceshipcontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("ship", a, 1.4, 0.6, 2.2, 400)
			w.phys3.SetGravityScale(c.id, 0)
			w.phys3.SetLinearDamping(c.id, 0.4)
			c.thrustMax = 8000
			return value.Num(float64(c.id)), nil
		}),
		"updatespaceship": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			throttle := float32(argN(a, 1, 0))
			pitch := float32(argN(a, 2, 0))
			roll := float32(argN(a, 3, 0))
			yaw := float32(argN(a, 4, 0))
			w.phys3.SetGravityScale(id, 0)
			w.phys3.ApplyLocalImpulse(id, 0, 0, throttle*c.thrustMax*0.016)
			fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(id)
			w.phys3.ApplyTorque(id,
				rx*pitch*3500+ux*yaw*3500+fx*roll*3500,
				ry*pitch*3500+uy*yaw*3500+fy*roll*3500,
				rz*pitch*3500+uz*yaw*3500+fz*roll*3500)
			return z()
		}),
		"createboatcontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("boat", a, 1.6, 0.35, 3.2, 800)
			c.thrustMax = 9000
			w.phys3.SetLinearDamping(c.id, 1.6)
			return value.Num(float64(c.id)), nil
		}),
		"updateboat": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			w.applyBoatForces(c, float32(argN(a, 1, 0)), float32(argN(a, 2, 0)))
			return z()
		}),
		"createmotorcyclecontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("moto", a, 0.25, 0.45, 1.1, 220)
			c.native = w.phys3.CreateMotorcycleVehicle(c.id, c.hx, c.hy, c.hz)
			return value.Num(float64(c.id)), nil
		}),
		"updatemotorcycle": need(func(a []value.Value) (value.Value, error) {
			return w.updateNamed("moto", a)
		}),
		"createhelicoptercontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("heli", a, 1.4, 0.5, 2.4, 700)
			c.thrustMax = 14000
			w.phys3.SetLinearDamping(c.id, 1.2)
			return value.Num(float64(c.id)), nil
		}),
		"updatehelicopter": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			col := float32(argN(a, 1, 0.8))
			cp := float32(argN(a, 2, 0))
			cr := float32(argN(a, 3, 0))
			yaw := float32(argN(a, 4, 0))
			fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(id)
			_, gy, _ := w.phys3.GetGravity()
			hover := c.mass * float32(math.Abs(float64(gy)))
			lift := hover + col*c.thrustMax
			w.phys3.ApplyForce(id,
				ux*lift+fx*cp*c.thrustMax*0.35+rx*cr*c.thrustMax*0.35,
				uy*lift+fy*cp*c.thrustMax*0.35+ry*cr*c.thrustMax*0.35,
				uz*lift+fz*cp*c.thrustMax*0.35+rz*cr*c.thrustMax*0.35)
			w.phys3.ApplyTorque(id, rx*cp*5000+ux*yaw*4000+fx*cr*5000, ry*cp*5000+uy*yaw*4000+fy*cr*5000, rz*cp*5000+uz*yaw*4000+fz*cr*5000)
			return z()
		}),
		"createhovercraftcontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("hover", a, 1.8, 0.25, 2.4, 500)
			c.thrustMax = 7000
			w.phys3.SetFriction(c.id, 0.05)
			w.phys3.SetLinearDamping(c.id, 1.4)
			return value.Num(float64(c.id)), nil
		}),
		"updatehovercraft": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			th := float32(argN(a, 1, 0))
			steer := float32(argN(a, 2, 0))
			fx, _, fz, _, _, _, _, _, _ := w.physAxes(id)
			w.phys3.ApplyForce(id, 0, c.mass*12, 0)
			w.phys3.ApplyForce(id, fx*th*c.thrustMax, 0, fz*th*c.thrustMax)
			w.phys3.ApplyTorque(id, 0, steer*3800, 0)
			return z()
		}),
		"createsubmarinecontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("sub", a, 1.2, 0.7, 4, 1400)
			c.thrustMax = 11000
			w.phys3.SetLinearDamping(c.id, 2.4)
			return value.Num(float64(c.id)), nil
		}),
		"updatesubmarine": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			th := float32(argN(a, 1, 0))
			steer := float32(argN(a, 2, 0))
			dive := float32(argN(a, 3, 0))
			x, y, pz, _ := w.phys3.GetPosition(id)
			wh := w.waterHeight(x, pz)
			depth := wh - y
			if depth > 0 {
				w.phys3.ApplyForce(id, 0, depth*c.mass*2.2, 0)
			}
			fx, _, fz, _, _, _, _, _, _ := w.physAxes(id)
			w.phys3.ApplyForce(id, fx*th*c.thrustMax, -dive*c.thrustMax*0.6, fz*th*c.thrustMax)
			w.phys3.ApplyTorque(id, 0, steer*5000, 0)
			return z()
		}),
		"createtrackedcontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("tank", a, 1.3, 0.45, 2.4, 1800)
			c.native = w.phys3.CreateTrackedVehicle(c.id, c.hx, c.hy, c.hz)
			w.phys3.SetLinearDamping(c.id, 2.0)
			return value.Num(float64(c.id)), nil
		}),
		"createtankcontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("tank", a, 1.3, 0.45, 2.4, 1800)
			c.native = w.phys3.CreateTrackedVehicle(c.id, c.hx, c.hy, c.hz)
			w.phys3.SetLinearDamping(c.id, 2.0)
			return value.Num(float64(c.id)), nil
		}),
		"updatetank": need(func(a []value.Value) (value.Value, error) {
			return w.updateNamed("tank", a)
		}),
		"updatetracked": need(func(a []value.Value) (value.Value, error) {
			return w.updateNamed("tank", a)
		}),
		"createdronecontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("drone", a, 0.45, 0.12, 0.45, 8)
			c.thrustMax = 220
			w.phys3.SetLinearDamping(c.id, 2.8)
			return value.Num(float64(c.id)), nil
		}),
		"updatedrone": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			th := float32(argN(a, 1, 0.55))
			cp := float32(argN(a, 2, 0))
			cr := float32(argN(a, 3, 0))
			yaw := float32(argN(a, 4, 0))
			fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(id)
			_, gy, _ := w.phys3.GetGravity()
			hover := c.mass * float32(math.Abs(float64(gy)))
			lift := hover + th*c.thrustMax
			ax, ay, az, _ := w.phys3.GetAngularVelocity(id)
			w.phys3.ApplyForce(id, ux*lift, uy*lift, uz*lift)
			w.phys3.ApplyTorque(id,
				rx*cp*40+ux*yaw*35+fx*cr*40-ax*12,
				ry*cp*40+uy*yaw*35+fy*cr*40-ay*12,
				rz*cp*40+uz*yaw*35+fz*cr*40-az*12)
			return z()
		}),
		"createglidercontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("glider", a, 5, 0.25, 2.2, 180)
			c.thrustMax, c.liftCl, c.dragCd, c.stall = 0, 1.15, 0.06, 8
			return value.Num(float64(c.id)), nil
		}),
		"updateglider": need(func(a []value.Value) (value.Value, error) {
			return w.updateAero("glider", a, 0, 1.2)
		}),
		"createskicontroller": need(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("ski", a, 0.35, 0.2, 1.4, 90)
			c.thrustMax = 2500
			w.phys3.SetFriction(c.id, 0.08)
			w.phys3.SetLinearDamping(c.id, 0.6)
			return value.Num(float64(c.id)), nil
		}),
		"updateski": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			th := float32(argN(a, 1, 0))
			steer := float32(argN(a, 2, 0))
			fx, _, fz, _, _, _, _, _, _ := w.physAxes(id)
			w.phys3.ApplyForce(id, fx*th*c.thrustMax, 0, fz*th*c.thrustMax)
			w.phys3.ApplyTorque(id, 0, steer*1200, 0)
			return z()
		}),
	}
}

func (w *World) updateNamed(kind string, a []value.Value) (value.Value, error) {
	id := argI(a, 0, 0)
	c := w.vehCtrls[id]
	if c == nil {
		return value.Num(0), nil
	}
	steer := float32(argN(a, 1, 0))
	throttle := float32(argN(a, 2, 0))
	brake := float32(argN(a, 3, 0))
	if c.native != 0 {
		w.phys3.SetVehicleInput(id, steer, throttle, brake)
	}
	w.updateWheeled(c, steer, throttle, brake)
	_ = kind
	return value.Num(0), nil
}

func (w *World) updateWheeled(c *vehCtrl, steer, throttle, brake float32) {
	fx, _, fz, _, _, _, _, _, _ := w.physAxes(c.id)
	force := throttle * 11000
	if c.kind == "tank" {
		force = throttle * 16000
		if throttle != 0 || steer != 0 {
			w.phys3.SetLinearDamping(c.id, 0.85)
		} else {
			w.phys3.SetLinearDamping(c.id, 2.2)
		}
	}
	w.phys3.ApplyForce(c.id, fx*force, 0, fz*force)
	w.phys3.ApplyTorque(c.id, 0, steer*5000, 0)
	if brake > 0 {
		vx, vy, vz, _ := w.phys3.GetVelocity(c.id)
		w.phys3.ApplyForce(c.id, -vx*brake*800, -vy*brake*80, -vz*brake*800)
	}
}

func (w *World) updateAero(kind string, a []value.Value, thrustScale, liftScale float32) (value.Value, error) {
	id := argI(a, 0, 0)
	c := w.vehCtrls[id]
	if c == nil {
		return value.Num(0), nil
	}
	_ = kind
	w.applyAero(c, float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)), float32(argN(a, 4, 0)), thrustScale, liftScale)
	return value.Num(0), nil
}
