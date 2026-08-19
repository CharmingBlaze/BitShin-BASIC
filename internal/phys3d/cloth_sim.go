package phys3d

import "math"

type clothSim struct {
	nx, ny int
	pos    [][3]float32
	prev   [][3]float32
	inv    []float32
	rest   [][3]float32 // i0, i1 as float in [0], [1]; rest in [2]
	nRest  int
	damp   float32
}

func newClothSim(x, y, z, width, height float32, nx, ny, pinFlags int, damping float32) *clothSim {
	if nx < 2 {
		nx = 2
	}
	if ny < 2 {
		ny = 2
	}
	if nx > 24 {
		nx = 24
	}
	if ny > 24 {
		ny = 24
	}
	if pinFlags == 0 {
		pinFlags = 1
	}
	if damping <= 0 {
		damping = 0.5
	}
	sx := width / float32(nx-1)
	sy := height / float32(ny-1)
	n := nx * ny
	c := &clothSim{nx: nx, ny: ny, pos: make([][3]float32, n), prev: make([][3]float32, n), inv: make([]float32, n), rest: make([][3]float32, n*6), damp: damping}
	idx := func(i, j int) int { return i + j*nx }
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			k := idx(i, j)
			c.pos[k] = [3]float32{x + float32(i)*sx, y - float32(j)*sy, z}
			c.prev[k] = c.pos[k]
			c.inv[k] = 1 / 0.2
		}
	}
	if pinFlags&1 != 0 {
		for i := 0; i < nx; i++ {
			c.inv[idx(i, 0)] = 0
		}
	}
	if pinFlags&2 != 0 {
		for i := 0; i < nx; i++ {
			c.inv[idx(i, ny-1)] = 0
		}
	}
	if pinFlags&4 != 0 {
		for j := 0; j < ny; j++ {
			c.inv[idx(0, j)] = 0
		}
	}
	if pinFlags&8 != 0 {
		for j := 0; j < ny; j++ {
			c.inv[idx(nx-1, j)] = 0
		}
	}
	addE := func(a, b int) {
		dx := c.pos[a][0] - c.pos[b][0]
		dy := c.pos[a][1] - c.pos[b][1]
		dz := c.pos[a][2] - c.pos[b][2]
		c.rest[c.nRest] = [3]float32{float32(a), float32(b), sqrt32(dx*dx + dy*dy + dz*dz)}
		c.nRest++
	}
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			if i < nx-1 {
				addE(idx(i, j), idx(i+1, j))
			}
			if j < ny-1 {
				addE(idx(i, j), idx(i, j+1))
			}
			if i < nx-1 && j < ny-1 {
				addE(idx(i, j), idx(i+1, j+1))
				addE(idx(i+1, j), idx(i, j+1))
			}
		}
	}
	return c
}

func (w *fallback) AddCloth(id int, x, y, z, width, height float32, nx, ny, pinFlags int, thickness, damping, gravityFactor float32) {
	_ = thickness
	_ = gravityFactor
	c := newClothSim(x, y, z, width, height, nx, ny, pinFlags, damping)
	if w.cloths == nil {
		w.cloths = map[int]*clothSim{}
	}
	w.cloths[id] = c
	cx, cy, cz := c.com()
	w.AddBox(id, cx, cy, cz, width*0.5, 0.05, 0.05, true)
}

func (c *clothSim) com() (float32, float32, float32) {
	sx, sy, sz := float32(0), float32(0), float32(0)
	for i := 0; i < len(c.pos); i++ {
		sx += c.pos[i][0]
		sy += c.pos[i][1]
		sz += c.pos[i][2]
	}
	n := float32(len(c.pos))
	if n < 1 {
		return 0, 0, 0
	}
	return sx / n, sy / n, sz / n
}

func (c *clothSim) step(dt, g float32) {
	if c == nil {
		return
	}
	for i := 0; i < len(c.pos); i++ {
		if c.inv[i] == 0 {
			continue
		}
		px, py, pz := c.pos[i][0], c.pos[i][1], c.pos[i][2]
		vx := (px - c.prev[i][0]) * (1 - c.damp*dt)
		vy := (py - c.prev[i][1]) * (1 - c.damp*dt)
		vz := (pz - c.prev[i][2]) * (1 - c.damp*dt)
		c.prev[i] = c.pos[i]
		c.pos[i][0] = px + vx
		c.pos[i][1] = py + vy + g*dt*dt
		c.pos[i][2] = pz + vz
	}
	for k := 0; k < 8; k++ {
		for e := 0; e < c.nRest; e++ {
			a := int(c.rest[e][0])
			b := int(c.rest[e][1])
			rest := c.rest[e][2]
			dx := c.pos[b][0] - c.pos[a][0]
			dy := c.pos[b][1] - c.pos[a][1]
			dz := c.pos[b][2] - c.pos[a][2]
			d := sqrt32(dx*dx + dy*dy + dz*dz)
			if d < 1e-6 {
				continue
			}
			diff := (d - rest) / d
			ia, ib := c.inv[a], c.inv[b]
			wsum := ia + ib
			if wsum <= 0 {
				continue
			}
			c.pos[a][0] += dx * diff * 0.5 * (ia / wsum)
			c.pos[a][1] += dy * diff * 0.5 * (ia / wsum)
			c.pos[a][2] += dz * diff * 0.5 * (ia / wsum)
			c.pos[b][0] -= dx * diff * 0.5 * (ib / wsum)
			c.pos[b][1] -= dy * diff * 0.5 * (ib / wsum)
			c.pos[b][2] -= dz * diff * 0.5 * (ib / wsum)
		}
	}
}

func (w *fallback) stepCloths(dt float32) {
	g := w.gy
	if g == 0 {
		g = -9.81
	}
	for id, c := range w.cloths {
		if c == nil {
			continue
		}
		c.step(dt, g)
		cx, cy, cz := c.com()
		if b := w.bodies[id]; b != nil {
			b.x, b.y, b.z = cx, cy, cz
			b.vx, b.vy, b.vz = 0, 0, 0
		}
	}
}

func (w *fallback) ClothVertexCount(id int) int {
	c := w.cloths[id]
	if c == nil {
		return 0
	}
	return len(c.pos)
}

func (w *fallback) ClothVertices(id int, dst []float32) int {
	c := w.cloths[id]
	if c == nil {
		return 0
	}
	cx, cy, cz := c.com()
	n := len(c.pos)
	if n*3 > len(dst) {
		n = len(dst) / 3
	}
	for i := 0; i < n; i++ {
		dst[i*3+0] = c.pos[i][0] - cx
		dst[i*3+1] = c.pos[i][1] - cy
		dst[i*3+2] = c.pos[i][2] - cz
	}
	return n
}

func (w *fallback) ApplyClothWind(id int, vx, vy, vz float32, start, step uint32) {
	c := w.cloths[id]
	if c == nil {
		return
	}
	if step < 1 {
		step = 1
	}
	phase := float32(start) * 0.11
	for i := 0; i < len(c.pos); i++ {
		if c.inv[i] == 0 {
			continue
		}
		ripple := float32(math.Sin(float64(phase + float32(i)*0.37)))
		s := float32(0.016)
		if step > 1 && (i+int(start))%int(step) != 0 {
			s *= 0.35
		}
		c.pos[i][0] += vx * s
		c.pos[i][1] += vy * s
		c.pos[i][2] += vz*s + ripple*0.02
	}
}
