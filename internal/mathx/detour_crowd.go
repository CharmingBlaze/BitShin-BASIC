package mathx

import "math"

// SampleDetour picks a velocity near the preferred one that delays collisions
// with neighbor disks. Rings of samples around the preferred velocity follow
// the DetourCrowd adaptive sampler. It does not call the Recast C API.
func SampleDetour(i int, disks []Disk, prefX, prefZ, maxSpeed, horizon float64) (float64, float64) {
	if i < 0 || i >= len(disks) || disks[i].R < 0 {
		return prefX, prefZ
	}
	if maxSpeed < 0 {
		maxSpeed = 0
	}
	if horizon <= 0 {
		horizon = 2.5
	}
	self := disks[i]
	bestX, bestZ := clampDetourSpeed(prefX, prefZ, maxSpeed)
	best := detourScore(i, self, bestX, bestZ, prefX, prefZ, disks, horizon)
	for ring := 1; ring <= 3; ring++ {
		rad := maxSpeed * float64(ring) / 3
		n := 8 * ring
		for s := 0; s < n; s++ {
			ang := 2 * math.Pi * float64(s) / float64(n)
			vx, vz := clampDetourSpeed(prefX+math.Cos(ang)*rad, prefZ+math.Sin(ang)*rad, maxSpeed)
			sc := detourScore(i, self, vx, vz, prefX, prefZ, disks, horizon)
			if sc > best {
				best = sc
				bestX, bestZ = vx, vz
			}
		}
	}
	return bestX, bestZ
}

func detourScore(selfI int, self Disk, vx, vz, prefX, prefZ float64, disks []Disk, horizon float64) float64 {
	align := 0.0
	ps := math.Hypot(prefX, prefZ)
	vs := math.Hypot(vx, vz)
	if ps > 1e-6 && vs > 1e-6 {
		align = (prefX*vx + prefZ*vz) / (ps * vs)
	}
	toi := horizon
	for j, other := range disks {
		if j == selfI || other.R < 0 {
			continue
		}
		t := diskTOI(self, vx, vz, other)
		if t < toi {
			toi = t
		}
	}
	return 2*align + toi/horizon
}

func diskTOI(self Disk, vx, vz float64, other Disk) float64 {
	rx := other.X - self.X
	rz := other.Z - self.Z
	rvx := other.VX - vx
	rvz := other.VZ - vz
	rad := self.R + other.R
	dist := math.Hypot(rx, rz)
	if dist < rad {
		// Already overlapping. A velocity that increases separation scores later.
		if rx*rvx+rz*rvz > 0 {
			return 0.35
		}
		return 0
	}
	a := rvx*rvx + rvz*rvz
	if a < 1e-8 {
		return 1e9
	}
	b := 2 * (rx*rvx + rz*rvz)
	c := dist*dist - rad*rad
	disc := b*b - 4*a*c
	if disc < 0 || b >= 0 {
		return 1e9
	}
	t := (-b - math.Sqrt(disc)) / (2 * a)
	if t < 0 {
		return 0
	}
	return t
}

func clampDetourSpeed(x, z, max float64) (float64, float64) {
	s := math.Hypot(x, z)
	if max <= 0 || s <= max || s < 1e-8 {
		return x, z
	}
	k := max / s
	return x * k, z * k
}
