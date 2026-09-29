package runtime

import (
	"math"

	"bitshinbasic/internal/value"
)

type vehCtrl struct {
	kind       string
	id         int
	hx, hy, hz float32
	native     int
	mass       float32
	area       float32
	rho        float32
	thrustMax  float32
	liftCl     float32
	dragCd     float32
	stall      float32
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
	if e == nil || e.mesh == nil || e.mesh.GetGeometry() == nil {
		// Pivots have no mesh. Keep the controller's hull instead of
		// treating a scale of 1 as a two-metre cube.
		return dx, dy, dz
	}
	s := e.node.GetNode().Scale()
	bb := e.mesh.GetGeometry().BoundingBox()
	hx := (bb.Max.X - bb.Min.X) * 0.5 * absf(s.X)
	hy := (bb.Max.Y - bb.Min.Y) * 0.5 * absf(s.Y)
	hz := (bb.Max.Z - bb.Min.Z) * 0.5 * absf(s.Z)
	if hx > 0.05 {
		dx = hx
	}
	if hy > 0.05 {
		dy = hy
	}
	if hz > 0.05 {
		dz = hz
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
	if len(a) >= 5 {
		if requestedMass := float32(argN(a, 4, float64(mass))); requestedMass > 0 {
			mass = requestedMass
		}
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
	if kind == "car" || kind == "moto" || kind == "tank" || kind == "boat" || kind == "jetski" || kind == "ski" || kind == "waterski" {
		w.phys3.OffsetCenterOfMass(id, 0, -boxHy*0.45, 0)
	}
	// OffsetCenterOfMass replaces the shape and Jolt rebuilds mass from
	// volume. Put the gameplay mass back or thrust sized for a few hundred
	// kilograms cannot move a hull that weighs as much as its displaced water.
	w.phys3.SetMass(id, mass)
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
	ax, ay, az, _ := w.phys3.GetAngularVelocity(c.id)
	speed := float32(math.Sqrt(float64(vx*vx + vy*vy + vz*vz)))
	q := 0.5 * c.rho * speed * speed
	cl := c.liftCl
	if c.stall > 0 && speed < c.stall {
		cl *= speed / c.stall
	}
	lift := q * c.area * cl * liftScale
	// Wing lift is nothing at a standstill, so a little throttle also holds
	// the nose up. Otherwise the demo planes lawn-dart before they reach
	// flying speed.
	if (c.kind == "plane" || c.kind == "jet" || c.kind == "glider") && throttle > 0 {
		lift += throttle * c.mass * 9.81 * 1.05
	}
	maxLift := c.mass * 9.81 * 1.8
	if lift > maxLift {
		lift = maxLift
	}
	drag := q * c.area * c.dragCd
	thrust := throttle * c.thrustMax * thrustScale

	w.phys3.SetLinearDamping(c.id, 0.15)
	w.phys3.SetAngularDamping(c.id, 1.2)

	// Forward thrust
	w.phys3.ApplyForce(c.id, fx*thrust, fy*thrust, fz*thrust)

	// Aerodynamic lift along wing up vector
	w.phys3.ApplyForce(c.id, ux*lift, uy*lift, uz*lift)

	// Aerodynamic drag opposing velocity
	if speed > 0.2 {
		inv := drag / speed
		w.phys3.ApplyForce(c.id, -vx*inv, -vy*inv, -vz*inv)
	}

	// Control surface torques scaled by dynamic pressure / speed
	spdFactor := float32(math.Min(float64(speed/20.0), 1.5))
	if spdFactor < 0.2 {
		spdFactor = 0.2
	}
	tPitch := pitch * c.mass * 8.0 * spdFactor
	tRoll := roll * c.mass * 7.0 * spdFactor
	tYaw := yaw * c.mass * 6.0 * spdFactor

	// Angular rate damping
	dampX := -ax * c.mass * 0.8
	dampY := -ay * c.mass * 0.8
	dampZ := -az * c.mass * 0.8

	w.phys3.ApplyTorque(c.id,
		rx*tPitch+fx*tRoll+ux*tYaw+dampX,
		ry*tPitch+fy*tRoll+uy*tYaw+dampY,
		rz*tPitch+fz*tRoll+uz*tYaw+dampZ)
}

func (w *World) applyBoatForces(c *vehCtrl, throttle, steer float32) {
	x, y, z, ok := w.phys3.GetPosition(c.id)
	if !ok {
		return
	}
	fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(c.id)
	damp := float32(1.15)
	if throttle > 0.15 || throttle < -0.15 {
		damp = 0.48
	}
	w.phys3.SetLinearDamping(c.id, damp)
	w.phys3.SetAngularDamping(c.id, 2.5)

	vx, vy, vz, _ := w.phys3.GetVelocity(c.id)
	angVx, angVy, angVz, _ := w.phys3.GetAngularVelocity(c.id)

	wh := w.waterHeight(x, z)
	// Push up only while the hull is in the water. Equilibrium is about
	// 40% submerged, so the deck stays above the surface and a wave does
	// not launch the boat.
	bottom := y - c.hy
	draft := wh - bottom
	if draft > 0 {
		frac := draft / (c.hy * 2)
		if frac > 1 {
			frac = 1
		}
		lift := c.mass*9.81*frac*2.4 - vy*c.mass*5.5*frac
		if lift < -c.mass*8 {
			lift = -c.mass * 8
		}
		w.phys3.ApplyForce(c.id, 0, lift, 0)
	}
	depth := wh - (y - c.hy)
	subRatio := depth / (c.hy * 2)
	if subRatio < 0 {
		subRatio = 0
	}
	if subRatio > 1 {
		subRatio = 1
	}

	if subRatio > 0.1 {
		// Lateral keel resistance (prevents sliding sideways in water)
		vLat := vx*rx + vy*ry + vz*rz
		w.phys3.ApplyForce(c.id, -rx*vLat*c.mass*3.5*subRatio, 0, -rz*vLat*c.mass*3.5*subRatio)

		// Water rotational damping
		w.phys3.ApplyTorque(c.id,
			-angVx*c.mass*0.8*subRatio,
			-angVy*c.mass*0.4*subRatio,
			-angVz*c.mass*0.8*subRatio)

		// Metacentric self-righting torque (keeps hull upright on water)
		rightX := -uz * c.mass * 35.0 * subRatio
		rightZ := ux * c.mass * 35.0 * subRatio
		w.phys3.ApplyTorque(c.id, rightX, 0, rightZ)
	}

	// Propeller thrust along hull forward direction
	thrust := throttle * c.thrustMax
	w.phys3.ApplyForce(c.id, fx*thrust, fy*thrust*0.1, fz*thrust)

	// Steering torque along local Up axis
	steerTorque := steer * c.mass * 6.5
	w.phys3.ApplyTorque(c.id, ux*steerTorque, uy*steerTorque, uz*steerTorque)
}

func (w *World) applyJetSkiForces(c *vehCtrl, throttle, steer float32) {
	x, y, z, ok := w.phys3.GetPosition(c.id)
	if !ok {
		return
	}
	fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(c.id)
	vx, vy, vz, _ := w.phys3.GetVelocity(c.id)
	angX, angY, angZ, _ := w.phys3.GetAngularVelocity(c.id)
	fwdSpeed := vx*fx + vy*fy + vz*fz
	if fwdSpeed < 0 {
		fwdSpeed = 0
	}
	planing := fwdSpeed / 15.0
	if planing > 1 {
		planing = 1
	}

	// Low global damping lets the ski coast. Hydrodynamic damping below is
	// applied only while the hull is touching the water.
	w.phys3.SetFriction(c.id, 0.03)
	w.phys3.SetLinearDamping(c.id, 0.28)
	w.phys3.SetAngularDamping(c.id, 3.0)

	// A narrow craft is unstable if its entire displacement is represented by
	// one force at the centre. These contacts let the bow, stern, and chines
	// follow the wave independently and turn a hard impact into several small
	// impulses.
	type hullPt struct {
		lx, ly, lz float32
		lift       float32
	}
	points := []hullPt{
		{-c.hx * 0.78, -c.hy * 0.78, c.hz * 0.72, 0.85},
		{c.hx * 0.78, -c.hy * 0.78, c.hz * 0.72, 0.85},
		{-c.hx * 0.82, -c.hy * 0.72, -c.hz * 0.72, 1.15},
		{c.hx * 0.82, -c.hy * 0.72, -c.hz * 0.72, 1.15},
		{0, -c.hy * 0.82, 0, 1.0},
	}
	nPts := float32(len(points))
	wet := float32(0)
	w.applyWaterBuoyancy(c.id, 3.4)

	for _, pt := range points {
		px := x + rx*pt.lx + ux*pt.ly + fx*pt.lz
		py := y + ry*pt.lx + uy*pt.ly + fy*pt.lz
		pz := z + rz*pt.lx + uz*pt.ly + fz*pt.lz
		depth := w.waterHeight(px, pz) - py
		if depth < -0.10 {
			continue
		}
		wet++
	}

	wetRatio := wet / nPts
	if wetRatio > 0 {
		planeLift := fwdSpeed * fwdSpeed * c.mass * 0.0065 * wetRatio
		maxPlane := c.mass * 9.81 * 0.38
		if planeLift > maxPlane {
			planeLift = maxPlane
		}
		w.phys3.ApplyForce(c.id, 0, planeLift, 0)
	}
	if wetRatio <= 0.05 {
		return
	}

	// The keel removes sideways skating without heavily damping forward speed.
	lateral := vx*rx + vy*ry + vz*rz
	keel := (2.8 + planing*4.5) * wetRatio
	w.phys3.ApplyForce(c.id, -rx*lateral*c.mass*keel, 0, -rz*lateral*c.mass*keel)

	// Water drag is strongest off-plane and light once the hull is skimming.
	forwardDrag := (2.4 - planing*0.8) * wetRatio
	w.phys3.ApplyForce(c.id, -fx*fwdSpeed*c.mass*forwardDrag, 0, -fz*fwdSpeed*c.mass*forwardDrag)

	// A jet nozzle needs water flow. Steering authority builds with speed but a
	// little remains at low speed so the demo never feels stuck.
	thrust := throttle * c.thrustMax * (0.35 + wetRatio*0.65)
	if throttle < 0 {
		thrust *= 0.55
	}
	w.phys3.ApplyForce(c.id, fx*thrust, fy*thrust*0.06, fz*thrust)

	steerAuthority := 0.28 + planing*0.92
	yawTorque := steer * c.mass * 8.5 * steerAuthority * wetRatio
	bankTorque := -steer * c.mass * (5.0 + planing*9.0) * wetRatio

	// Align the local up axis with world up. The damping terms suppress the
	// violent snap-back that previously showed up after landing on a crest.
	rightX := -uz * c.mass * (34.0 + planing*8.0) * wetRatio
	rightZ := ux * c.mass * (34.0 + planing*8.0) * wetRatio
	w.phys3.ApplyTorque(c.id,
		rightX+fx*bankTorque+ux*yawTorque-angX*c.mass*1.65,
		uy*yawTorque-angY*c.mass*(0.72+wetRatio*0.45),
		rightZ+fz*bankTorque+uz*yawTorque-angZ*c.mass*1.65)
}

func (w *World) applyWaterSkiForces(c *vehCtrl, throttle, edge float32, towID int) {
	x, y, z, ok := w.phys3.GetPosition(c.id)
	if !ok {
		return
	}
	fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(c.id)
	vx, vy, vz, _ := w.phys3.GetVelocity(c.id)
	angVx, angVy, angVz, _ := w.phys3.GetAngularVelocity(c.id)
	fwd := vx*fx + vy*fy + vz*fz

	w.phys3.SetFriction(c.id, 0.04)
	w.phys3.SetLinearDamping(c.id, 0.35)
	w.phys3.SetAngularDamping(c.id, 3.2)

	type skiPt struct{ lx, ly, lz float32 }
	pts := []skiPt{
		{0, -c.hy * 0.45, c.hz * 0.92},
		{0, -c.hy * 0.45, 0},
		{0, -c.hy * 0.45, -c.hz * 0.88},
		{c.hx * 0.9, -c.hy * 0.4, c.hz * 0.2},
		{-c.hx * 0.9, -c.hy * 0.4, c.hz * 0.2},
	}
	nPts := float32(len(pts))
	wet := float32(0)
	tipY, tailY := float32(0), float32(0)
	w.applyWaterBuoyancy(c.id, 3.0)
	plane := fwd / 11.0
	if plane < 0 {
		plane = 0
	}
	if plane > 1 {
		plane = 1
	}

	for i, pt := range pts {
		px := x + rx*pt.lx + ux*pt.ly + fx*pt.lz
		py := y + ry*pt.lx + uy*pt.ly + fy*pt.lz
		pz := z + rz*pt.lx + uz*pt.ly + fz*pt.lz
		wh := w.waterHeight(px, pz)
		if i == 0 {
			tipY = wh
		}
		if i == 2 {
			tailY = wh
		}
		depth := wh - py
		if depth <= -0.08 {
			continue
		}
		wet++
	}

	onWater := wet / nPts
	if onWater > 0 {
		planing := fwd * fwd * c.mass * 0.018 * (0.55 + plane) * onWater
		w.phys3.ApplyForce(c.id, 0, planing, 0)
	}
	if onWater > 0.08 {
		edgeAbs := float32(math.Abs(float64(edge)))
		grip := (1.8 + edgeAbs*5.5) * (0.35 + plane) * onWater
		vLat := vx*rx + vy*ry + vz*rz
		w.phys3.ApplyForce(c.id, -rx*vLat*c.mass*grip, 0, -rz*vLat*c.mass*grip)

		// Carve: edged skis turn instead of sliding.
		carve := edge * fwd * c.mass * 0.42 * onWater
		w.phys3.ApplyForce(c.id, rx*carve, 0, rz*carve)
		yawT := edge * (4.0 + plane*6.0) * c.mass * 0.12
		w.phys3.ApplyTorque(c.id, ux*yawT, uy*yawT-angVy*c.mass*0.7, uz*yawT)

		bank := -edge * c.mass * (6.0 + plane*8.0)
		rightX := -uz * c.mass * (18.0 + plane*10.0) * onWater
		rightZ := ux * c.mass * (18.0 + plane*10.0) * onWater
		w.phys3.ApplyTorque(c.id, rightX+fx*bank-angVx*c.mass*0.9, 0, rightZ+fz*bank-angVz*c.mass*0.9)

		pitchFollow := (tipY - tailY) * c.mass * 14.0 * onWater
		w.phys3.ApplyTorque(c.id, rx*pitchFollow, ry*pitchFollow, rz*pitchFollow)

		sinkDrag := (1.15 - plane*0.85) * onWater
		w.phys3.ApplyForce(c.id, -vx*c.mass*0.22*sinkDrag, 0, -vz*c.mass*0.22*sinkDrag)
	}

	if towID != 0 && towID != c.id {
		// A real rope constraint transfers the boat's force itself. Do not layer
		// the legacy spring approximation on top of it.
		if w.ropeBetween(c.id, towID) != nil {
			return
		}
		tx, ty, tz, tok := w.phys3.GetPosition(towID)
		if !tok {
			if e := w.ents[towID]; e != nil && e.node != nil {
				p := e.node.GetNode().Position()
				tx, ty, tz = fromG3N(p.X, p.Y, p.Z)
				tok = true
			}
		}
		if tok {
			dx := tx - x
			dy := ty - y
			dz := tz - z
			dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
			slack := float32(13.5)
			if dist > 0.2 {
				nx, ny, nz := dx/dist, dy/dist, dz/dist
				pull := throttle
				if pull < 0.25 {
					pull = 0.25
				}
				if dist > slack {
					stretch := dist - slack
					towVX, towVY, towVZ, _ := w.phys3.GetVelocity(towID)
					stretchRate := (towVX-vx)*nx + (towVY-vy)*ny + (towVZ-vz)*nz
					spring := (stretch*c.mass*12.0 + stretchRate*c.mass*3.5) * pull
					if spring < 0 {
						spring = 0
					}
					maxPull := c.mass * 32.0
					if spring > maxPull {
						spring = maxPull
					}
					w.phys3.ApplyForce(c.id, nx*spring, ny*spring*0.35, nz*spring)
					if _, _, _, bok := w.phys3.GetPosition(towID); bok {
						w.phys3.ApplyForce(towID, -nx*spring*0.08, 0, -nz*spring*0.08)
					}
				}
				// Face the boat a little so the rope stays in front.
				align := (nx*rx + nz*rz) * c.mass * 2.4
				w.phys3.ApplyTorque(c.id, 0, -align, 0)
			}
			return
		}
	}

	// No tow boat: throttle is the pull, like a ski being yanked up to plane.
	thrust := throttle * c.thrustMax * (0.45 + plane*0.7)
	w.phys3.ApplyForce(c.id, fx*thrust, fy*thrust*0.05, fz*thrust)
}

func (w *World) vehicleCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	_ = need
	return map[string]cmd{
		"applytorque": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.ApplyTorque(argI(a, 0, 0), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			return z()
		}),
		"applyforceatposition": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.ApplyForceAtPosition(argI(a, 0, 0),
				float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)),
				float32(argN(a, 4, 0)), float32(argN(a, 5, 0)), float32(argN(a, 6, 0)))
			return z()
		}),
		"applylocalimpulse": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.ApplyLocalImpulse(argI(a, 0, 0), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			return z()
		}),
		"setgravityscale": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetGravityScale(argI(a, 0, 0), float32(argN(a, 1, 1)))
			return z()
		}),
		"getgravityscale": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Num(float64(w.phys3.GetGravityScale(argI(a, 0, 0)))), nil
		}),
		"setrestitution": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetRestitution(argI(a, 0, 0), float32(argN(a, 1, 0)))
			return z()
		}),
		"getrestitution": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Num(float64(w.phys3.GetRestitution(argI(a, 0, 0)))), nil
		}),
		"setlineardamping": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetLinearDamping(argI(a, 0, 0), float32(argN(a, 1, 0)))
			return z()
		}),
		"getlineardamping": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Num(float64(w.phys3.GetLinearDamping(argI(a, 0, 0)))), nil
		}),
		"setangulardamping": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetAngularDamping(argI(a, 0, 0), float32(argN(a, 1, 0)))
			return z()
		}),
		"getangulardamping": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Num(float64(w.phys3.GetAngularDamping(argI(a, 0, 0)))), nil
		}),
		"setfriction": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetFriction(argI(a, 0, 0), float32(argN(a, 1, 0.5)))
			return z()
		}),
		"getfriction": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Num(float64(w.phys3.GetFriction(argI(a, 0, 0)))), nil
		}),
		"getccd": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Num(float64(w.phys3.GetCCD(argI(a, 0, 0)))), nil
		}),
		"getmass": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Num(float64(w.phys3.GetMass(argI(a, 0, 0)))), nil
		}),
		"setmass": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetMass(argI(a, 0, 0), float32(argN(a, 1, 1)))
			return z()
		}),
		"applybuoyancy": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			scale := float32(argN(a, 2, 1))
			if len(a) >= 2 {
				x, _, pz, ok := w.phys3.GetPosition(id)
				if !ok {
					return z()
				}
				waterY := float32(argN(a, 1, 0))
				dt := float32(w.delta)
				if dt <= 0 {
					dt = 1.0 / 60
				}
				w.phys3.ApplyBuoyancyImpulse(id, x, waterY, pz, 0, 1, 0, scale, 0.5, 0.35, 0, 0, 0, dt)
				return z()
			}
			w.applyWaterBuoyancy(id, scale)
			return z()
		}),
		"extendedupdate": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return z()
		}),
		"characterextendedupdate": n(func(a []value.Value) (value.Value, error) {
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
		"createvehicle": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("car", a, 0.9, 0.35, 1.8, 1200)
			c.native = w.phys3.CreateWheeledVehicle(c.id, c.hx, c.hy, c.hz)
			return value.Num(float64(c.id)), nil
		}),
		"createcarcontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("car", a, 0.9, 0.35, 1.8, 1200)
			c.native = w.phys3.CreateWheeledVehicle(c.id, c.hx, c.hy, c.hz)
			return value.Num(float64(c.id)), nil
		}),
		"setvehicleinput": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			steer := float32(argN(a, 1, 0))
			throttle := float32(argN(a, 2, 0))
			brake := float32(argN(a, 3, 0))
			if c := w.vehCtrls[id]; c != nil && (c.kind == "car" || c.kind == "moto" || c.kind == "tank") {
				if c.native != 0 {
					w.phys3.SetVehicleInput(id, steer, throttle, brake)
				} else {
					w.updateWheeled(c, steer, throttle, brake)
				}
			} else {
				w.phys3.SetVehicleInput(id, steer, throttle, brake)
			}
			return z()
		}),
		"updatecar": n(func(a []value.Value) (value.Value, error) {
			return w.updateNamed("car", a)
		}),
		"updatevehicle": n(func(a []value.Value) (value.Value, error) {
			return w.updateNamed("car", a)
		}),
		"createplanecontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("plane", a, 4, 0.4, 3, 900)
			c.thrustMax, c.liftCl, c.dragCd, c.stall = 18000, 1.7, 0.14, 9
			w.phys3.CreatePlaneController(c.id)
			return value.Num(float64(c.id)), nil
		}),
		"updateplane": n(func(a []value.Value) (value.Value, error) {
			return w.updateAero("plane", a, 1, 1)
		}),
		"createjetcontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("jet", a, 3.2, 0.35, 4.5, 1100)
			c.thrustMax, c.liftCl, c.dragCd, c.stall = 42000, 1.15, 0.07, 16
			return value.Num(float64(c.id)), nil
		}),
		"updatejet": n(func(a []value.Value) (value.Value, error) {
			return w.updateAero("jet", a, 1.6, 0.7)
		}),
		"createspaceshipcontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("ship", a, 1.4, 0.6, 2.2, 400)
			w.phys3.SetGravityScale(c.id, 0)
			w.phys3.SetLinearDamping(c.id, 0.4)
			c.thrustMax = 8000
			return value.Num(float64(c.id)), nil
		}),
		"updatespaceship": n(func(a []value.Value) (value.Value, error) {
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
			w.phys3.SetLinearDamping(id, 0.25)
			w.phys3.SetAngularDamping(id, 1.8)

			fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(id)
			ax, ay, az, _ := w.phys3.GetAngularVelocity(id)

			// RCS Main Thruster
			thrust := throttle * c.thrustMax
			w.phys3.ApplyForce(id, fx*thrust, fy*thrust, fz*thrust)

			// RCS Rotational Thrusters
			tPitch := pitch * c.mass * 9.0
			tRoll := roll * c.mass * 8.0
			tYaw := yaw * c.mass * 8.0

			w.phys3.ApplyTorque(id,
				rx*tPitch+fx*tRoll+ux*tYaw-ax*c.mass*0.8,
				ry*tPitch+fy*tRoll+uy*tYaw-ay*c.mass*0.8,
				rz*tPitch+fz*tRoll+uz*tYaw-az*c.mass*0.8)
			return z()
		}),
		"createboatcontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("boat", a, 1.6, 0.35, 3.2, 800)
			c.thrustMax = 9000
			w.phys3.SetLinearDamping(c.id, 1.2)
			w.phys3.SetAngularDamping(c.id, 2.5)
			return value.Num(float64(c.id)), nil
		}),
		"updateboat": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			w.applyBoatForces(c, float32(argN(a, 1, 0)), float32(argN(a, 2, 0)))
			return z()
		}),
		"createmotorcyclecontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("moto", a, 0.25, 0.45, 1.1, 220)
			c.native = w.phys3.CreateMotorcycleVehicle(c.id, c.hx, c.hy, c.hz)
			return value.Num(float64(c.id)), nil
		}),
		"updatemotorcycle": n(func(a []value.Value) (value.Value, error) {
			return w.updateNamed("moto", a)
		}),
		"createhelicoptercontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("heli", a, 1.4, 0.5, 2.4, 700)
			c.thrustMax = 14000
			w.phys3.SetLinearDamping(c.id, 0.4)
			w.phys3.SetAngularDamping(c.id, 2.0)
			return value.Num(float64(c.id)), nil
		}),
		"updatehelicopter": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			col := float32(argN(a, 1, 0.0)) // collective delta (-1 to +1, 0 is hover)
			cp := float32(argN(a, 2, 0))    // cyclic pitch (-1 forward, +1 back)
			cr := float32(argN(a, 3, 0))    // cyclic roll
			yaw := float32(argN(a, 4, 0))   // tail rotor yaw

			fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(id)
			vx, vy, vz, _ := w.phys3.GetVelocity(id)
			ax, ay, az, _ := w.phys3.GetAngularVelocity(id)

			w.phys3.SetLinearDamping(id, 0.4)
			w.phys3.SetAngularDamping(id, 2.0)

			_, gy, _ := w.phys3.GetGravity()
			hover := c.mass * float32(math.Abs(float64(gy)))
			lift := hover + col*(c.mass*12.0)
			if lift < 0 {
				lift = 0
			}

			// Main rotor thrust vector tilted by cyclic input
			w.phys3.ApplyForce(id,
				ux*lift+fx*cp*c.mass*6.0+rx*cr*c.mass*6.0,
				uy*lift+fy*cp*c.mass*6.0+ry*cr*c.mass*6.0,
				uz*lift+fz*cp*c.mass*6.0+rz*cr*c.mass*6.0)

			// Air drag
			w.phys3.ApplyForce(id, -vx*c.mass*0.15, -vy*c.mass*0.1, -vz*c.mass*0.15)

			// Control torques
			tPitch := cp * c.mass * 6.0
			tRoll := cr * c.mass * 6.0
			tYaw := yaw * c.mass * 5.0

			// Gyro horizon auto-leveling when cyclic inputs are neutral
			levelX := (uy*0 - uz*1) * c.mass * 8.0 * (1.0 - float32(math.Abs(float64(cp))))
			levelZ := (ux*1 - uy*0) * c.mass * 8.0 * (1.0 - float32(math.Abs(float64(cr))))

			w.phys3.ApplyTorque(id,
				rx*tPitch+fx*tRoll+ux*tYaw+levelX-ax*c.mass*1.2,
				ry*tPitch+fy*tRoll+uy*tYaw-ay*c.mass*1.0,
				rz*tPitch+fz*tRoll+uz*tYaw+levelZ-az*c.mass*1.2)
			return z()
		}),
		"createhovercraftcontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("hover", a, 1.8, 0.25, 2.4, 500)
			c.thrustMax = 7000
			w.phys3.SetFriction(c.id, 0.05)
			w.phys3.SetLinearDamping(c.id, 0.6)
			w.phys3.SetAngularDamping(c.id, 2.5)
			return value.Num(float64(c.id)), nil
		}),
		"updatehovercraft": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			th := float32(argN(a, 1, 0))
			steer := float32(argN(a, 2, 0))

			px, py, pz, _ := w.phys3.GetPosition(id)
			fx, _, fz, ux, uy, uz, rx, ry, rz := w.physAxes(id)
			vx, vy, vz, _ := w.phys3.GetVelocity(id)
			ax, ay, az, _ := w.phys3.GetAngularVelocity(id)

			w.phys3.SetLinearDamping(id, 0.6)
			w.phys3.SetAngularDamping(id, 2.5)

			wh := w.waterHeight(px, pz)
			surfY := wh
			if len(w.terrains) > 0 {
				if th := w.terrainHeight(px, pz); th > surfY {
					surfY = th
				}
			}

			dist := py - surfY
			targetHover := c.hy * 1.6
			if targetHover < 0.6 {
				targetHover = 0.6
			}

			if dist < targetHover*2.0 {
				cushionRatio := 1.0 - (dist / (targetHover * 2.0))
				if cushionRatio < 0 {
					cushionRatio = 0
				}
				cushionForce := c.mass*9.81*(1.0+cushionRatio*2.5) - vy*c.mass*3.5
				if cushionForce > 0 {
					w.phys3.ApplyForce(id, 0, cushionForce, 0)
				}

				// Skirt lateral drift resistance
				vLat := vx*rx + vy*ry + vz*rz
				w.phys3.ApplyForce(id, -rx*vLat*c.mass*1.2, 0, -rz*vLat*c.mass*1.2)

				// Self-righting on cushion
				rightX := (uy*0 - uz*1) * c.mass * 20.0
				rightZ := (ux*1 - uy*0) * c.mass * 20.0
				w.phys3.ApplyTorque(id, rightX, 0, rightZ)
			}

			// Forward / reverse thruster
			thrust := th * c.thrustMax
			w.phys3.ApplyForce(id, fx*thrust, 0, fz*thrust)

			// Rudder yaw torque
			steerTorque := steer * c.mass * 5.5
			w.phys3.ApplyTorque(id, ux*steerTorque-ax*c.mass*0.5, uy*steerTorque-ay*c.mass*0.5, uz*steerTorque-az*c.mass*0.5)
			return z()
		}),
		"createsubmarinecontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("sub", a, 1.2, 0.7, 4, 1400)
			c.thrustMax = 11000
			w.phys3.SetLinearDamping(c.id, 1.8)
			w.phys3.SetAngularDamping(c.id, 3.0)
			return value.Num(float64(c.id)), nil
		}),
		"updatesubmarine": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			th := float32(argN(a, 1, 0))
			steer := float32(argN(a, 2, 0))
			dive := float32(argN(a, 3, 0))

			px, py, pz, _ := w.phys3.GetPosition(id)
			fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(id)
			vx, vy, vz, _ := w.phys3.GetVelocity(id)
			ax, ay, az, _ := w.phys3.GetAngularVelocity(id)

			w.phys3.SetLinearDamping(id, 1.8)
			w.phys3.SetAngularDamping(id, 3.0)

			wh := w.waterHeight(px, pz)
			depth := wh - py
			if depth > 0 {
				// Submerged: neutral buoyancy + ballast tank control
				buoy := c.mass*9.81 + (-dive * c.mass * 4.5) - vy*c.mass*2.0
				w.phys3.ApplyForce(id, 0, buoy, 0)

				// Water drag on all velocities
				w.phys3.ApplyForce(id, -vx*c.mass*0.5, 0, -vz*c.mass*0.5)

				// Hydroplanes pitch control when moving forward
				fwdSpeed := vx*fx + vy*fy + vz*fz
				tPitch := dive * fwdSpeed * c.mass * 0.8
				tYaw := steer * c.mass * 5.0

				// Self-righting roll stabilization
				rollLevel := (ux*1 - uy*0) * c.mass * 15.0

				w.phys3.ApplyTorque(id,
					rx*tPitch+ux*tYaw-ax*c.mass*1.2,
					ry*tPitch+uy*tYaw-ay*c.mass*1.2,
					rz*tPitch+uz*tYaw+rollLevel-az*c.mass*1.2)
			}

			// Propeller thrust
			thrust := th * c.thrustMax
			w.phys3.ApplyForce(id, fx*thrust, fy*thrust, fz*thrust)
			return z()
		}),
		"createtrackedcontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("tank", a, 1.3, 0.45, 2.4, 1800)
			c.native = w.phys3.CreateTrackedVehicle(c.id, c.hx, c.hy, c.hz)
			w.phys3.SetLinearDamping(c.id, 2.0)
			return value.Num(float64(c.id)), nil
		}),
		"createtankcontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("tank", a, 1.3, 0.45, 2.4, 1800)
			c.native = w.phys3.CreateTrackedVehicle(c.id, c.hx, c.hy, c.hz)
			w.phys3.SetLinearDamping(c.id, 2.0)
			return value.Num(float64(c.id)), nil
		}),
		"updatetank": n(func(a []value.Value) (value.Value, error) {
			return w.updateNamed("tank", a)
		}),
		"updatetracked": n(func(a []value.Value) (value.Value, error) {
			return w.updateNamed("tank", a)
		}),
		"createdronecontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("drone", a, 0.45, 0.12, 0.45, 8)
			c.thrustMax = 220
			w.phys3.SetLinearDamping(c.id, 0.8)
			w.phys3.SetAngularDamping(c.id, 2.5)
			return value.Num(float64(c.id)), nil
		}),
		"updatedrone": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			th := float32(argN(a, 1, 0.0)) // throttle delta (-1 to +1, 0 is hover)
			cp := float32(argN(a, 2, 0))   // pitch
			cr := float32(argN(a, 3, 0))   // roll
			yaw := float32(argN(a, 4, 0))  // yaw

			fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(id)
			vx, vy, vz, _ := w.phys3.GetVelocity(id)
			ax, ay, az, _ := w.phys3.GetAngularVelocity(id)

			w.phys3.SetLinearDamping(id, 0.8)
			w.phys3.SetAngularDamping(id, 2.5)

			_, gy, _ := w.phys3.GetGravity()
			hover := c.mass * float32(math.Abs(float64(gy)))
			lift := hover + th*(c.mass*14.0)
			if lift < 0 {
				lift = 0
			}

			// Quadcopter 4-rotor thrust vector
			w.phys3.ApplyForce(id,
				ux*lift+fx*cp*c.mass*7.0+rx*cr*c.mass*7.0,
				uy*lift+fy*cp*c.mass*7.0+ry*cr*c.mass*7.0,
				uz*lift+fz*cp*c.mass*7.0+rz*cr*c.mass*7.0)

			// Air drag
			w.phys3.ApplyForce(id, -vx*c.mass*0.3, -vy*c.mass*0.2, -vz*c.mass*0.3)

			// Quadcopter flight controller auto-leveling PID
			tPitch := cp * c.mass * 7.0
			tRoll := cr * c.mass * 7.0
			tYaw := yaw * c.mass * 6.0

			levelX := (uy*0 - uz*1) * c.mass * 12.0 * (1.0 - float32(math.Abs(float64(cp))))
			levelZ := (ux*1 - uy*0) * c.mass * 12.0 * (1.0 - float32(math.Abs(float64(cr))))

			w.phys3.ApplyTorque(id,
				rx*tPitch+fx*tRoll+ux*tYaw+levelX-ax*c.mass*1.8,
				ry*tPitch+fy*tRoll+uy*tYaw-ay*c.mass*1.5,
				rz*tPitch+fz*tRoll+uz*tYaw+levelZ-az*c.mass*1.8)
			return z()
		}),
		"createglidercontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("glider", a, 5, 0.25, 2.2, 180)
			c.thrustMax, c.liftCl, c.dragCd, c.stall = 0, 1.15, 0.06, 8
			return value.Num(float64(c.id)), nil
		}),
		"updateglider": n(func(a []value.Value) (value.Value, error) {
			return w.updateAero("glider", a, 0, 1.2)
		}),
		"createskicontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("ski", a, 0.35, 0.2, 1.4, 90)
			c.thrustMax = 2500
			w.phys3.SetFriction(c.id, 0.08)
			w.phys3.SetLinearDamping(c.id, 0.6)
			return value.Num(float64(c.id)), nil
		}),
		"updateski": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			th := float32(argN(a, 1, 0))
			steer := float32(argN(a, 2, 0))
			if w.currentWater() != nil {
				w.applyWaterSkiForces(c, th, steer, argI(a, 3, 0))
				return z()
			}
			fx, _, fz, _, _, _, _, _, _ := w.physAxes(id)
			w.phys3.ApplyForce(id, fx*th*c.thrustMax, 0, fz*th*c.thrustMax)
			w.phys3.ApplyTorque(id, 0, steer*1200, 0)
			return z()
		}),
		"createwaterskicontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("waterski", a, 0.22, 0.12, 1.05, 85)
			c.thrustMax = 4200
			w.phys3.SetFriction(c.id, 0.04)
			w.phys3.SetLinearDamping(c.id, 0.4)
			w.phys3.SetAngularDamping(c.id, 3.0)
			return value.Num(float64(c.id)), nil
		}),
		"updatewaterski": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			w.applyWaterSkiForces(c, float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), argI(a, 3, 0))
			return z()
		}),
		"createmechcontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("mech", a, 1.2, 1.6, 1.2, 4500)
			c.thrustMax = 65000
			w.phys3.SetFriction(c.id, 0.85)
			w.phys3.SetLinearDamping(c.id, 1.5)
			w.phys3.SetAngularDamping(c.id, 4.0)
			return value.Num(float64(c.id)), nil
		}),
		"updatemech": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			throttle := float32(argN(a, 1, 0))
			turn := float32(argN(a, 2, 0))
			strafe := float32(argN(a, 3, 0))

			fx, _, fz, ux, uy, uz, rx, _, rz := w.physAxes(id)
			ax, ay, az, _ := w.phys3.GetAngularVelocity(id)

			// Walk force
			fwdF := throttle * c.mass * 16.0
			sideF := strafe * c.mass * 12.0
			w.phys3.ApplyForce(id, fx*fwdF+rx*sideF, 0, fz*fwdF+rz*sideF)

			// Turn yaw torque
			tYaw := turn * c.mass * 12.0
			w.phys3.ApplyTorque(id, ux*tYaw, uy*tYaw-ay*c.mass*2.0, uz*tYaw)

			// Gyro self-righting to keep heavy mech standing
			rightX := (uy*0 - uz*1) * c.mass * 40.0
			rightZ := (ux*1 - uy*0) * c.mass * 40.0
			w.phys3.ApplyTorque(id, rightX-ax*c.mass*2.0, 0, rightZ-az*c.mass*2.0)
			return z()
		}),
		"createlandercontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("lander", a, 1.4, 1.2, 1.4, 1200)
			c.thrustMax = 32000
			w.phys3.SetLinearDamping(c.id, 0.1)
			w.phys3.SetAngularDamping(c.id, 2.0)
			return value.Num(float64(c.id)), nil
		}),
		"updatelander": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			thrust := float32(argN(a, 1, 0))
			pitch := float32(argN(a, 2, 0))
			roll := float32(argN(a, 3, 0))
			yaw := float32(argN(a, 4, 0))

			fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(id)
			ax, ay, az, _ := w.phys3.GetAngularVelocity(id)

			// Main gimbal engine thrust along local +Up axis
			mainF := thrust * c.thrustMax
			w.phys3.ApplyForce(id, ux*mainF, uy*mainF, uz*mainF)

			// 3-axis RCS attitude control
			tPitch := pitch * c.mass * 8.0
			tRoll := roll * c.mass * 8.0
			tYaw := yaw * c.mass * 7.0
			w.phys3.ApplyTorque(id,
				rx*tPitch+fx*tRoll+ux*tYaw-ax*c.mass*0.9,
				ry*tPitch+fy*tRoll+uy*tYaw-ay*c.mass*0.9,
				rz*tPitch+fz*tRoll+uz*tYaw-az*c.mass*0.9)
			return z()
		}),
		"createjetskicontroller": n(func(a []value.Value) (value.Value, error) {
			c := w.bindCtrl("jetski", a, 0.6, 0.35, 1.5, 380)
			c.thrustMax = 15500
			w.phys3.SetFriction(c.id, 0.03)
			w.phys3.SetLinearDamping(c.id, 0.28)
			w.phys3.SetAngularDamping(c.id, 3.0)
			return value.Num(float64(c.id)), nil
		}),
		"updatejetski": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.vehCtrls[id]
			if c == nil {
				return z()
			}
			w.applyJetSkiForces(c, float32(argN(a, 1, 0)), float32(argN(a, 2, 0)))
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
	} else {
		w.updateWheeled(c, steer, throttle, brake)
	}
	_ = kind
	return value.Num(0), nil
}

func (w *World) updateWheeled(c *vehCtrl, steer, throttle, brake float32) {
	fx, _, fz, _, _, _, rx, _, rz := w.physAxes(c.id)
	vx, vy, vz, _ := w.phys3.GetVelocity(c.id)
	ax, _, az, _ := w.phys3.GetAngularVelocity(c.id)

	w.phys3.SetAngularDamping(c.id, 2.0)

	force := throttle * c.mass * 9.0
	if c.kind == "tank" {
		force = throttle * c.mass * 12.0
		if throttle != 0 || steer != 0 {
			w.phys3.SetLinearDamping(c.id, 0.85)
		} else {
			w.phys3.SetLinearDamping(c.id, 2.5)
		}
	} else {
		w.phys3.SetLinearDamping(c.id, 0.35)
	}

	// Drive traction
	w.phys3.ApplyForce(c.id, fx*force, 0, fz*force)

	// Steering yaw torque
	steerTorque := steer * c.mass * 4.5
	w.phys3.ApplyTorque(c.id, 0, steerTorque, 0)

	// Anti-rollover stabilization: counteract extreme body roll
	rightX := -rz * c.mass * 12.0
	rightZ := rx * c.mass * 12.0
	w.phys3.ApplyTorque(c.id, rightX-ax*c.mass*0.5, 0, rightZ-az*c.mass*0.5)

	if brake > 0 {
		w.phys3.ApplyForce(c.id, -vx*brake*c.mass*1.5, -vy*brake*c.mass*0.1, -vz*brake*c.mass*1.5)
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
