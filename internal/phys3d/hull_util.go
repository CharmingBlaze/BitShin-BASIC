package phys3d

func hullHalfExtents(points [][3]float32) (hx, hy, hz float32) {
	if len(points) == 0 {
		return 0.5, 0.5, 0.5
	}
	minX, minY, minZ := points[0][0], points[0][1], points[0][2]
	maxX, maxY, maxZ := minX, minY, minZ
	for _, p := range points {
		if p[0] < minX {
			minX = p[0]
		}
		if p[1] < minY {
			minY = p[1]
		}
		if p[2] < minZ {
			minZ = p[2]
		}
		if p[0] > maxX {
			maxX = p[0]
		}
		if p[1] > maxY {
			maxY = p[1]
		}
		if p[2] > maxZ {
			maxZ = p[2]
		}
	}
	hx, hy, hz = (maxX-minX)*0.5, (maxY-minY)*0.5, (maxZ-minZ)*0.5
	if hx < 0.05 {
		hx = 0.05
	}
	if hy < 0.05 {
		hy = 0.05
	}
	if hz < 0.05 {
		hz = 0.05
	}
	return hx, hy, hz
}

func hullRadius(points [][3]float32) float32 {
	r := float32(0.1)
	for _, p := range points {
		for _, c := range p {
			if c < 0 {
				c = -c
			}
			if c > r {
				r = c
			}
		}
	}
	return r
}
