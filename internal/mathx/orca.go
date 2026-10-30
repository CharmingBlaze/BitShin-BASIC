package mathx

import "math"

// Disk is one crowd agent on the XZ plane.
type Disk struct {
	X, Z   float64
	VX, VZ float64
	R      float64
}

// ORCA adjusts a preferred velocity so agent i reciprocally avoids the others.
// horizon is seconds. dt is the step used when two disks already overlap.
// The result is clamped to maxSpeed. This is not the DetourCrowd C API.
func ORCA(i int, agents []Disk, prefX, prefZ, maxSpeed, horizon, dt float64) (float64, float64) {
	if i < 0 || i >= len(agents) {
		return clampSpeed(prefX, prefZ, maxSpeed)
	}
	if horizon <= 0 {
		horizon = 1
	}
	if dt <= 0 {
		dt = 1.0 / 60
	}
	self := agents[i]
	vx, vz := prefX, prefZ
	for pass := 0; pass < 2; pass++ {
		for j := range agents {
			if j == i {
				continue
			}
			o := agents[j]
			if o.R < 0 {
				continue
			}
			relX := o.X - self.X
			relZ := o.Z - self.Z
			dist := math.Hypot(relX, relZ)
			if dist < 1e-8 {
				relX, relZ = 1, 0
				dist = 1
			}
			nx, nz := relX/dist, relZ/dist
			combined := self.R + o.R
			if combined <= 0 {
				combined = 0.3
			}
			sep := dist - combined
			rvx := vx - o.VX
			rvz := vz - o.VZ
			approach := rvx*nx + rvz*nz
			var minSep float64
			if sep >= 0 {
				if approach <= 0 {
					continue
				}
				if sep/approach > horizon {
					continue
				}
				minSep = sep / horizon
			} else {
				minSep = sep / dt
			}
			if approach <= minSep {
				continue
			}
			corr := (minSep - approach) * 0.5
			vx += nx * corr
			vz += nz * corr
		}
		vx, vz = clampSpeed(vx, vz, maxSpeed)
	}
	return vx, vz
}

func clampSpeed(vx, vz, maxSpeed float64) (float64, float64) {
	if maxSpeed <= 0 {
		return vx, vz
	}
	sp := math.Hypot(vx, vz)
	if sp > maxSpeed {
		s := maxSpeed / sp
		return vx * s, vz * s
	}
	return vx, vz
}
