//go:build !nojolt

package phys3d

import "math"

func (w *joltWorld) CreatePlaneController(id int) int {
	if w.planes == nil {
		w.planes = map[int]planeAero{}
	}
	hx, hz := float32(4), float32(3)
	mass := float32(900)
	w.SetMass(id, mass)
	w.planes[id] = planeAero{
		thrust: 18000,
		cl:     0.9,
		cd:     0.08,
		stall:  12,
		rho:    1.2,
		area:   hx * hz * 2,
		mass:   mass,
	}
	return id
}

func (w *joltWorld) UpdatePlane(id int, throttle, pitch, roll, yaw float32) {
	a, ok := w.planes[id]
	if !ok {
		return
	}
	qx, qy, qz, qw, _ := w.GetRotation(id)
	fx, fy, fz := quatRotateVec(qx, qy, qz, qw, 0, 0, 1)
	ux, uy, uz := quatRotateVec(qx, qy, qz, qw, 0, 1, 0)
	rx, ry, rz := quatRotateVec(qx, qy, qz, qw, 1, 0, 0)
	vx, vy, vz, _ := w.GetVelocity(id)
	speed := float32(math.Sqrt(float64(vx*vx + vy*vy + vz*vz)))
	qdyn := 0.5 * a.rho * speed * speed
	cl := a.cl
	if a.stall > 0 && speed < a.stall {
		cl *= speed / a.stall
	}
	lift := qdyn * a.area * cl
	drag := qdyn * a.area * a.cd
	px, py, pz, _ := w.GetPosition(id)
	wing := a.area * 0.15
	if wing < 0.5 {
		wing = 0.5
	}
	halfLift := lift * 0.5
	w.ApplyForceAtPosition(id, ux*halfLift, uy*halfLift, uz*halfLift, px+rx*wing, py+ry*wing, pz+rz*wing)
	w.ApplyForceAtPosition(id, ux*halfLift, uy*halfLift, uz*halfLift, px-rx*wing, py-ry*wing, pz-rz*wing)
	w.ApplyLocalImpulse(id, 0, 0, throttle*a.thrust*0.016)
	if speed > 0.5 {
		inv := drag / speed
		w.ApplyForce(id, -vx*inv, -vy*inv, -vz*inv)
	}
	w.ApplyTorque(id,
		rx*pitch*8000+ux*yaw*6000+fx*roll*7000,
		ry*pitch*8000+uy*yaw*6000+fy*roll*7000,
		rz*pitch*8000+uz*yaw*6000+fz*roll*7000)
}
