package runtime

import "math"

// value / Perlin-style 2D noise + FBM. No extra module.
type noise2 struct {
	perm [512]int
}

func newNoise2(seed int) *noise2 {
	n := &noise2{}
	p := make([]int, 256)
	for i := 0; i < 256; i++ {
		p[i] = i
	}
	s := uint32(seed)
	if s == 0 {
		s = 1
	}
	for i := 255; i > 0; i-- {
		s = s*1664525 + 1013904223
		j := int(s>>16) % (i + 1)
		if j < 0 {
			j = -j
		}
		p[i], p[j] = p[j], p[i]
	}
	for i := 0; i < 256; i++ {
		n.perm[i] = p[i]
		n.perm[i+256] = p[i]
	}
	return n
}

func fade(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp64(a, b, t float64) float64 { return a + t*(b-a) }

func grad2(h int, x, z float64) float64 {
	switch h & 3 {
	case 0:
		return x + z
	case 1:
		return -x + z
	case 2:
		return x - z
	default:
		return -x - z
	}
}

func (n *noise2) noise(x, z float64) float64 {
	if n == nil {
		return 0
	}
	xi := int(math.Floor(x)) & 255
	zi := int(math.Floor(z)) & 255
	xf := x - math.Floor(x)
	zf := z - math.Floor(z)
	u, v := fade(xf), fade(zf)
	aa := n.perm[n.perm[xi]+zi]
	ab := n.perm[n.perm[xi]+zi+1]
	ba := n.perm[n.perm[xi+1]+zi]
	bb := n.perm[n.perm[xi+1]+zi+1]
	x1 := lerp64(grad2(aa, xf, zf), grad2(ba, xf-1, zf), u)
	x2 := lerp64(grad2(ab, xf, zf-1), grad2(bb, xf-1, zf-1), u)
	return lerp64(x1, x2, v)
}

func (n *noise2) fbm(x, z float64, octaves int, persist, lacunarity float64) float64 {
	if octaves < 1 {
		octaves = 1
	}
	if octaves > 8 {
		octaves = 8
	}
	if persist <= 0 {
		persist = 0.5
	}
	if lacunarity <= 0 {
		lacunarity = 2
	}
	amp, freq, sum, norm := 1.0, 1.0, 0.0, 0.0
	for i := 0; i < octaves; i++ {
		sum += n.noise(x*freq, z*freq) * amp
		norm += amp
		amp *= persist
		freq *= lacunarity
	}
	if norm <= 0 {
		return 0
	}
	return sum / norm
}
