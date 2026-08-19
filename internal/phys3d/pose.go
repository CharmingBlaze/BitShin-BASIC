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
