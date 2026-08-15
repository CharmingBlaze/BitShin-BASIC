package phys3d

// BoxOverlapMTV is the minimum translation that separates two oriented boxes.
// Half-extents are Jolt/CreateBodyBox sizes. hit is false when the boxes are apart.
func BoxOverlapMTV(
	ax, ay, az, ahx, ahy, ahz, aqx, aqy, aqz, aqw float32,
	bx, by, bz, bhx, bhy, bhz, bqx, bqy, bqz, bqw float32,
) (nx, ny, nz, depth float32, hit bool) {
	if ahx <= 0 || ahy <= 0 || ahz <= 0 || bhx <= 0 || bhy <= 0 || bhz <= 0 {
		return 0, 0, 0, 0, false
	}
	a0x, a0y, a0z := quatMulVec(aqx, aqy, aqz, aqw, 1, 0, 0)
	a1x, a1y, a1z := quatMulVec(aqx, aqy, aqz, aqw, 0, 1, 0)
	a2x, a2y, a2z := quatMulVec(aqx, aqy, aqz, aqw, 0, 0, 1)
	b0x, b0y, b0z := quatMulVec(bqx, bqy, bqz, bqw, 1, 0, 0)
	b1x, b1y, b1z := quatMulVec(bqx, bqy, bqz, bqw, 0, 1, 0)
	b2x, b2y, b2z := quatMulVec(bqx, bqy, bqz, bqw, 0, 0, 1)
	dx, dy, dz := bx-ax, by-ay, bz-az
	axes := [15][3]float32{
		{a0x, a0y, a0z}, {a1x, a1y, a1z}, {a2x, a2y, a2z},
		{b0x, b0y, b0z}, {b1x, b1y, b1z}, {b2x, b2y, b2z},
		{a0y*b0z - a0z*b0y, a0z*b0x - a0x*b0z, a0x*b0y - a0y*b0x},
		{a0y*b1z - a0z*b1y, a0z*b1x - a0x*b1z, a0x*b1y - a0y*b1x},
		{a0y*b2z - a0z*b2y, a0z*b2x - a0x*b2z, a0x*b2y - a0y*b2x},
		{a1y*b0z - a1z*b0y, a1z*b0x - a1x*b0z, a1x*b0y - a1y*b0x},
		{a1y*b1z - a1z*b1y, a1z*b1x - a1x*b1z, a1x*b1y - a1y*b1x},
		{a1y*b2z - a1z*b2y, a1z*b2x - a1x*b2z, a1x*b2y - a1y*b2x},
		{a2y*b0z - a2z*b0y, a2z*b0x - a2x*b0z, a2x*b0y - a2y*b0x},
		{a2y*b1z - a2z*b1y, a2z*b1x - a2x*b1z, a2x*b1y - a2y*b1x},
		{a2y*b2z - a2z*b2y, a2z*b2x - a2x*b2z, a2x*b2y - a2y*b2x},
	}
	best := float32(1e9)
	bnx, bny, bnz := float32(0), float32(0), float32(0)
	for i := 0; i < 15; i++ {
		ux, uy, uz := axes[i][0], axes[i][1], axes[i][2]
		ulen := sqrt32(ux*ux + uy*uy + uz*uz)
		if ulen < 1e-5 {
			continue
		}
		ux, uy, uz = ux/ulen, uy/ulen, uz/ulen
		ar := ahx*abs32(dot3(a0x, a0y, a0z, ux, uy, uz)) +
			ahy*abs32(dot3(a1x, a1y, a1z, ux, uy, uz)) +
			ahz*abs32(dot3(a2x, a2y, a2z, ux, uy, uz))
		br := bhx*abs32(dot3(b0x, b0y, b0z, ux, uy, uz)) +
			bhy*abs32(dot3(b1x, b1y, b1z, ux, uy, uz)) +
			bhz*abs32(dot3(b2x, b2y, b2z, ux, uy, uz))
		sep := abs32(dot3(dx, dy, dz, ux, uy, uz))
		if sep > ar+br {
			return 0, 0, 0, 0, false
		}
		ov := ar + br - sep
		if ov < best {
			best = ov
			bnx, bny, bnz = ux, uy, uz
			if dot3(dx, dy, dz, ux, uy, uz) < 0 {
				bnx, bny, bnz = -ux, -uy, -uz
			}
		}
	}
	if best >= 1e8 {
		return 0, 0, 0, 0, false
	}
	return bnx, bny, bnz, best, true
}

func quatMulVec(qx, qy, qz, qw, vx, vy, vz float32) (float32, float32, float32) {
	tx := 2 * (qy*vz - qz*vy)
	ty := 2 * (qz*vx - qx*vz)
	tz := 2 * (qx*vy - qy*vx)
	return vx + qw*tx + (qy*tz - qz*ty),
		vy + qw*ty + (qz*tx - qx*tz),
		vz + qw*tz + (qx*ty - qy*tx)
}

func dot3(ax, ay, az, bx, by, bz float32) float32 {
	return ax*bx + ay*by + az*bz
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
