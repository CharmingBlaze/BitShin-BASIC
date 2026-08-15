package runtime

import "math"

// Terrain-OpenGL (rickie95 / mark99106) value-noise height.
// Clean-room: same hash + quintic interpolant + FBM + pow() as their TES,
// not a copy of their C++. Used for CPU mesh / TerrainHeight so walk queries
// match the displaced grid. Hardware tessellation is not required (GL 3.3).

func fract64(x float64) float64 { return x - math.Floor(x) }

func terrainGLRandom2(x, y, seedX, seedY float64) float64 {
	s := x*(12.9898+seedX) + y*(78.233+seedY)
	return fract64(math.Sin(s) * 43758.5453123)
}

func terrainGLInterpNoise(x, y, seedX, seedY float64) float64 {
	ix, iy := math.Floor(x), math.Floor(y)
	fx, fy := x-ix, y-iy
	a := terrainGLRandom2(ix, iy, seedX, seedY)
	b := terrainGLRandom2(ix+1, iy, seedX, seedY)
	c := terrainGLRandom2(ix, iy+1, seedX, seedY)
	d := terrainGLRandom2(ix+1, iy+1, seedX, seedY)
	wx := fx * fx * fx * (10 + fx*(-15+6*fx))
	wy := fy * fy * fy * (10 + fy*(-15+6*fy))
	k0 := a
	k1 := b - a
	k2 := c - a
	k3 := d - c - b + a
	return k0 + k1*wx + k2*wy + k3*wx*wy
}

func terrainGLHeight(x, z float64, octaves int, freq, disp, power, seedX, seedY float64) float64 {
	if octaves < 1 {
		octaves = 1
	}
	if octaves > 12 {
		octaves = 12
	}
	if freq <= 0 {
		freq = 0.035
	}
	if disp <= 0 {
		disp = 16
	}
	if power <= 0 {
		power = 1.25
	}
	// Their TES used frequency = 0.005*freq on kilometer-scale tiles, then
	// multiplied *before* the first sample (so the first octave was 0.01*freq).
	// Our chunks are tens of world units; 0.005*0.035 makes a nearly flat field.
	// freq is therefore spatial frequency in cycles per world unit.
	// disp is first-octave amplitude in world units (visible hills).
	persistence := 0.5
	total := 0.0
	frequency := freq
	amplitude := disp
	for i := 0; i < octaves; i++ {
		total += terrainGLInterpNoise(x*frequency, z*frequency, seedX, seedY) * amplitude
		frequency *= 2
		amplitude *= persistence
	}
	if total < 0 {
		total = 0
	}
	// Mild reshape so peaks read; keep mid-slopes (power 2+ flattened everything).
	if power > 1.6 {
		power = 1.6
	}
	return math.Pow(total, power)
}

func terrainGLNormal(x, z, eps float64, octaves int, freq, disp, power, seedX, seedY float64) (nx, ny, nz float64) {
	if eps < 0.05 {
		eps = 0.35
	}
	hl := terrainGLHeight(x-eps, z, octaves, freq, disp, power, seedX, seedY)
	hr := terrainGLHeight(x+eps, z, octaves, freq, disp, power, seedX, seedY)
	hd := terrainGLHeight(x, z-eps, octaves, freq, disp, power, seedX, seedY)
	hu := terrainGLHeight(x, z+eps, octaves, freq, disp, power, seedX, seedY)
	// Blitz XZ; slope from finite differences (same idea as their computeNormals).
	tx, ty, tz := 2*eps, hr-hl, 0.0
	bx, by, bz := 0.0, hu-hd, 2*eps
	nx = ty*bz - tz*by
	ny = tz*bx - tx*bz
	nz = tx*by - ty*bx
	lenN := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if lenN < 1e-8 {
		return 0, 1, 0
	}
	return nx / lenN, ny / lenN, nz / lenN
}

// terrainSplatWeights returns sand, grass, rock, snow in 0–1 (sum ≈ 1).
// Matches Terrain-OpenGL getTexture: sand near water, grass on flats, rock on slopes.
func smoothstep64(e0, e1, x float64) float64 {
	if e1 <= e0 {
		if x < e0 {
			return 0
		}
		return 1
	}
	t := (x - e0) / (e1 - e0)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return t * t * (3 - 2*t)
}

func terrainSplatWeights(height, waterY, blend, grassCover, ny float64, snowOn bool, snowH float64) (sand, grass, rock, snow float64) {
	if blend < 0.15 {
		blend = 0.15
	}
	_ = grassCover
	slope := ny
	if slope < 0 {
		slope = 0
	}
	if slope > 1 {
		slope = 1
	}
	rock = 1 - smoothstep64(0.4, 0.75, slope)
	sand = 1 - smoothstep64(waterY, waterY+blend*2, height)
	if snowOn {
		snow = smoothstep64(snowH-1.2, snowH+2, height) * smoothstep64(0.38, 0.78, slope)
	}
	rest := 1 - sand - snow
	if rest < 0 {
		rest = 0
	}
	grass = rest * (1 - rock)
	rock *= rest
	sum := sand + grass + rock + snow
	if sum < 1e-8 {
		return 0, 1, 0, 0
	}
	return sand / sum, grass / sum, rock / sum, snow / sum
}
