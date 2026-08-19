package phys3d

import "math"

func quatNormalize(x, y, z, w float32) (float32, float32, float32, float32) {
	n := float32(math.Sqrt(float64(x*x + y*y + z*z + w*w)))
	if n < 1e-8 {
		return 0, 0, 0, 1
	}
	return x / n, y / n, z / n, w / n
}

func quatIntegrate(q [4]float32, omega [3]float32, dt float32) [4]float32 {
	qx, qy, qz, qw := quatNormalize(q[0], q[1], q[2], q[3])
	hx, hy, hz := omega[0]*dt*0.5, omega[1]*dt*0.5, omega[2]*dt*0.5
	nx := hx*qw + hy*qz - hz*qy
	ny := hy*qw + hz*qx - hx*qz
	nz := hz*qw + hx*qy - hy*qx
	nw := -hx*qx - hy*qy - hz*qz
	x, y, z, w := quatNormalize(qx+nx, qy+ny, qz+nz, qw+nw)
	return [4]float32{x, y, z, w}
}

func pairKey(a, b int) [2]int {
	if a > b {
		return [2]int{b, a}
	}
	return [2]int{a, b}
}

func clampLayer(n int) int {
	if n < 0 {
		return 0
	}
	if n > 31 {
		return 31
	}
	return n
}

// softConstraintDelta is the position error to remove this step.
// kind 2 = slider (free along axis), kind 3 = spring toward rest length, else point/fixed.
func softConstraintDelta(kind int, ax, ay, az, dx, dy, dz, rest float32) (float32, float32, float32) {
	switch kind {
	case 2:
		n2 := ax*ax + ay*ay + az*az
		if n2 < 1e-10 {
			return dx, dy, dz
		}
		n := float32(math.Sqrt(float64(n2)))
		ax, ay, az = ax/n, ay/n, az/n
		d := dx*ax + dy*ay + dz*az
		return dx - ax*d, dy - ay*d, dz - az*d
	case 3:
		n2 := dx*dx + dy*dy + dz*dz
		if n2 < 1e-10 {
			return 0, 0, 0
		}
		n := float32(math.Sqrt(float64(n2)))
		target := rest
		if target < 0 {
			target = 0
		}
		s := (n - target) / n
		return dx * s, dy * s, dz * s
	default:
		return dx, dy, dz
	}
}

func unit3(x, y, z float32) (float32, float32, float32, float32) {
	n := float32(math.Sqrt(float64(x*x + y*y + z*z)))
	if n < 1e-8 {
		return 0, 1, 0, 0
	}
	return x / n, y / n, z / n, n
}

func ortho3(ax, ay, az float32) (float32, float32, float32) {
	if ax*ax < 0.5 {
		return 0, -az, ay
	}
	return -ay, ax, 0
}

func rotateAxis(ax, ay, az, ang, vx, vy, vz float32) (float32, float32, float32) {
	ax, ay, az, _ = unit3(ax, ay, az)
	c := float32(math.Cos(float64(ang)))
	s := float32(math.Sin(float64(ang)))
	dot := ax*vx + ay*vy + az*vz
	cx := ay*vz - az*vy
	cy := az*vx - ax*vz
	cz := ax*vy - ay*vx
	t := 1 - c
	return vx*c + cx*s + ax*dot*t, vy*c + cy*s + ay*dot*t, vz*c + cz*s + az*dot*t
}

// coneClampDir keeps v inside a cone of halfRad around axis, same length.
func coneClampDir(ax, ay, az, halfRad, vx, vy, vz float32) (float32, float32, float32) {
	if halfRad <= 0 {
		return vx, vy, vz
	}
	if halfRad > 3.1 {
		return vx, vy, vz
	}
	ax, ay, az, an := unit3(ax, ay, az)
	if an == 0 {
		return vx, vy, vz
	}
	nx, ny, nz, vn := unit3(vx, vy, vz)
	if vn == 0 {
		return vx, vy, vz
	}
	c := ax*nx + ay*ny + az*nz
	minc := float32(math.Cos(float64(halfRad)))
	if c >= minc {
		return vx, vy, vz
	}
	px, py, pz := nx-ax*c, ny-ay*c, nz-az*c
	px, py, pz, pn := unit3(px, py, pz)
	if pn == 0 {
		px, py, pz = ortho3(ax, ay, az)
		px, py, pz, _ = unit3(px, py, pz)
	}
	s := float32(math.Sin(float64(halfRad)))
	return (ax*minc + px*s) * vn, (ay*minc + py*s) * vn, (az*minc + pz*s) * vn
}

func hingeAngle(ax, ay, az, rx, ry, rz, vx, vy, vz float32) float32 {
	ax, ay, az, _ = unit3(ax, ay, az)
	prx, pry, prz := rx-ax*(rx*ax+ry*ay+rz*az), ry-ay*(rx*ax+ry*ay+rz*az), rz-az*(rx*ax+ry*ay+rz*az)
	pvx, pvy, pvz := vx-ax*(vx*ax+vy*ay+vz*az), vy-ay*(vx*ax+vy*ay+vz*az), vz-az*(vx*ax+vy*ay+vz*az)
	prx, pry, prz, rn := unit3(prx, pry, prz)
	if rn == 0 {
		return 0
	}
	s := ax*(pry*pvz-prz*pvy) + ay*(prz*pvx-prx*pvz) + az*(prx*pvy-pry*pvx)
	c := prx*pvx + pry*pvy + prz*pvz
	return float32(math.Atan2(float64(s), float64(c)))
}

func hingeClampVec(ax, ay, az, rx, ry, rz, hmin, hmax, vx, vy, vz float32) (float32, float32, float32) {
	ang := hingeAngle(ax, ay, az, rx, ry, rz, vx, vy, vz)
	clamped := ang
	if ang < hmin {
		clamped = hmin
	}
	if ang > hmax {
		clamped = hmax
	}
	if clamped == ang {
		return vx, vy, vz
	}
	return rotateAxis(ax, ay, az, clamped-ang, vx, vy, vz)
}

func (j *softJoint) captureRef(ax, ay, az, bx, by, bz float32) {
	j.refx, j.refy, j.refz = bx-ax, by-ay, bz-az
	px, py, pz, pn := unit3(
		j.refx-j.ax*(j.refx*j.ax+j.refy*j.ay+j.refz*j.az),
		j.refy-j.ay*(j.refx*j.ax+j.refy*j.ay+j.refz*j.az),
		j.refz-j.az*(j.refx*j.ax+j.refy*j.ay+j.refz*j.az),
	)
	if pn < 0.05 {
		px, py, pz = ortho3(j.ax, j.ay, j.az)
		px, py, pz, _ = unit3(px, py, pz)
		j.refx, j.refy, j.refz = px, py, pz
	}
}

func applySoftLimits(j *softJoint, ax, ay, az, bx, by, bz float32) (float32, float32, float32, float32, float32, float32) {
	vx, vy, vz := bx-ax, by-ay, bz-az
	if j.cone > 0 {
		vx, vy, vz = coneClampDir(j.ax, j.ay, j.az, j.cone, vx, vy, vz)
	}
	if j.hlim {
		vx, vy, vz = hingeClampVec(j.ax, j.ay, j.az, j.refx, j.refy, j.refz, j.hmin, j.hmax, vx, vy, vz)
	}
	tx, ty, tz := float32(0), float32(0), float32(0)
	if j.mtorque > 0 {
		ang := hingeAngle(j.ax, j.ay, j.az, j.refx, j.refy, j.refz, vx, vy, vz)
		err := j.motor - ang
		step := err
		maxStep := j.mtorque * 0.002
		if maxStep < 0.01 {
			maxStep = 0.01
		}
		if step > maxStep {
			step = maxStep
		}
		if step < -maxStep {
			step = -maxStep
		}
		vx, vy, vz = rotateAxis(j.ax, j.ay, j.az, step, vx, vy, vz)
		t := err * 25
		if t > j.mtorque {
			t = j.mtorque
		}
		if t < -j.mtorque {
			t = -j.mtorque
		}
		nx, ny, nz, _ := unit3(j.ax, j.ay, j.az)
		tx, ty, tz = nx*t, ny*t, nz*t
	}
	return ax + vx, ay + vy, az + vz, tx, ty, tz
}

func quatRotateVec(qx, qy, qz, qw, vx, vy, vz float32) (float32, float32, float32) {
	ix := qw*vx + qy*vz - qz*vy
	iy := qw*vy + qz*vx - qx*vz
	iz := qw*vz + qx*vy - qy*vx
	iw := -qx*vx - qy*vy - qz*vz
	return ix*qw + iw*-qx + iy*-qz - iz*-qy,
		iy*qw + iw*-qy + iz*-qx - ix*-qz,
		iz*qw + iw*-qz + ix*-qy - iy*-qx
}

func bodyQuat(qx, qy, qz, qw float32) (float32, float32, float32, float32) {
	if qx == 0 && qy == 0 && qz == 0 && qw == 0 {
		return 0, 0, 0, 1
	}
	return quatNormalize(qx, qy, qz, qw)
}

func (j *softJoint) applySpin(vx, vy, vz, ox, oy, oz, dt float32) (float32, float32, float32, float32, float32, float32) {
	nx, ny, nz, _ := unit3(j.ax, j.ay, j.az)
	if j.hfric > 0 && dt > 0 {
		dot := vx*nx + vy*ny + vz*nz
		px, py, pz := vx-nx*dot, vy-ny*dot, vz-nz*dot
		s := 1 - j.hfric*dt
		if s < 0.05 {
			s = 0.05
		}
		vx, vy, vz = nx*dot+px*s, ny*dot+py*s, nz*dot+pz*s
	}
	if j.twist > 0 && dt > 0 {
		spin := ox*nx + oy*ny + oz*nz
		j.twang += spin * dt
		if j.twang > j.twist || j.twang < -j.twist {
			if j.twang > j.twist {
				j.twang = j.twist
			} else {
				j.twang = -j.twist
			}
			ox -= nx * spin
			oy -= ny * spin
			oz -= nz * spin
		}
	}
	return vx, vy, vz, ox, oy, oz
}
