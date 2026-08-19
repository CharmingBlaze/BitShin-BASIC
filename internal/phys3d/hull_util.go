package phys3d

import "math"

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

func compoundHullPoints(parts []CompoundPart) [][3]float32 {
	out := make([][3]float32, 0, len(parts)*8)
	for _, p := range parts {
		hx, hy, hz := p.A, p.B, p.C
		switch p.Kind {
		case CompoundSphere:
			hx, hy, hz = p.A, p.A, p.A
		case CompoundCapsule, CompoundCylinder:
			hx, hy, hz = p.B, p.A+p.B, p.B
		}
		if hx < 0.05 {
			hx = 0.05
		}
		if hy < 0.05 {
			hy = 0.05
		}
		if hz < 0.05 {
			hz = 0.05
		}
		sx := [2]float32{-hx, hx}
		sy := [2]float32{-hy, hy}
		sz := [2]float32{-hz, hz}
		for i := 0; i < 2; i++ {
			for j := 0; j < 2; j++ {
				for k := 0; k < 2; k++ {
					out = append(out, [3]float32{p.Ox + sx[i], p.Oy + sy[j], p.Oz + sz[k]})
				}
			}
		}
	}
	return out
}

func cylinderHullPoints(halfH, r float32, sides int) [][3]float32 {
	if sides < 6 {
		sides = 12
	}
	if r < 0.05 {
		r = 0.05
	}
	if halfH < 0.05 {
		halfH = 0.05
	}
	out := make([][3]float32, 0, sides*2)
	for ring := 0; ring < 2; ring++ {
		y := -halfH
		if ring == 1 {
			y = halfH
		}
		for i := 0; i < sides; i++ {
			a := float64(i) * 2 * math.Pi / float64(sides)
			out = append(out, [3]float32{r * float32(math.Cos(a)), y, r * float32(math.Sin(a))})
		}
	}
	return out
}

func heightFieldTris(samples []float32, n int, ox, oy, oz, sx, sy, sz float32) ([][3]float32, []int32) {
	if n < 2 || len(samples) < n*n {
		return nil, nil
	}
	verts := make([][3]float32, n*n)
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			h := samples[j*n+i]
			verts[j*n+i] = [3]float32{ox + float32(i)*sx, oy + h*sy, oz + float32(j)*sz}
		}
	}
	idx := make([]int32, 0, (n-1)*(n-1)*6)
	for j := 0; j < n-1; j++ {
		for i := 0; i < n-1; i++ {
			a := int32(i + j*n)
			b := a + 1
			c := int32(i + (j+1)*n)
			d := c + 1
			idx = append(idx, a, c, b, b, c, d)
		}
	}
	return verts, idx
}
