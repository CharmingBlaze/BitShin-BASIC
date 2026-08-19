// Package mathx is the engine numeric library (Gonum).
//
// G3N scene APIs take math32.Vector3 / Matrix4. Do not replace those types
// in the scene graph. Convert at the boundary with FromVec3 / ToVec3 / FromMat4 / ToMat4.
package mathx

import (
	"math"

	"github.com/g3n/engine/math32"
	"gonum.org/v1/gonum/mat"
)

// Vec3 is a Gonum 3-vector (column).
func Vec3(x, y, z float64) *mat.VecDense {
	return mat.NewVecDense(3, []float64{x, y, z})
}

// FromVec3 copies a G3N math32 vector into Gonum.
func FromVec3(v math32.Vector3) *mat.VecDense {
	return Vec3(float64(v.X), float64(v.Y), float64(v.Z))
}

// ToVec3 copies a Gonum 3-vector into math32. v may be longer; first 3 are used.
func ToVec3(v mat.Vector) math32.Vector3 {
	if v == nil || v.Len() < 3 {
		return math32.Vector3{}
	}
	return math32.Vector3{float32(v.AtVec(0)), float32(v.AtVec(1)), float32(v.AtVec(2))}
}

// FromMat4 copies a G3N column-major Matrix4 into a Gonum 4×4 dense matrix.
func FromMat4(m math32.Matrix4) *mat.Dense {
	d := mat.NewDense(4, 4, nil)
	for c := 0; c < 4; c++ {
		for r := 0; r < 4; r++ {
			d.Set(r, c, float64(m[c*4+r]))
		}
	}
	return d
}

// ToMat4 copies a 4×4 Gonum matrix into math32 column-major order.
func ToMat4(m mat.Matrix) math32.Matrix4 {
	var out math32.Matrix4
	out.Identity()
	if m == nil {
		return out
	}
	r, c := m.Dims()
	if r < 4 || c < 4 {
		return out
	}
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			out[col*4+row] = float32(m.At(row, col))
		}
	}
	return out
}

// Mul4 multiplies two 4×4 matrices (a * b) using Gonum.
func Mul4(a, b mat.Matrix) *mat.Dense {
	out := mat.NewDense(4, 4, nil)
	out.Mul(a, b)
	return out
}

// BatchMul4 multiplies each of n 4×4 matrices on the right by right.
// src is n*16 floats, column-major math32 layout. Returns n*16 floats.
func BatchMul4(src []float64, n int, right mat.Matrix) []float64 {
	if n < 1 || len(src) < n*16 {
		return nil
	}
	out := make([]float64, n*16)
	tmp := mat.NewDense(4, 4, nil)
	for i := 0; i < n; i++ {
		off := i * 16
		a := mat.NewDense(4, 4, nil)
		for c := 0; c < 4; c++ {
			for r := 0; r < 4; r++ {
				a.Set(r, c, src[off+c*4+r])
			}
		}
		tmp.Mul(a, right)
		for c := 0; c < 4; c++ {
			for r := 0; r < 4; r++ {
				out[off+c*4+r] = tmp.At(r, c)
			}
		}
	}
	return out
}

// SH9 is order-2 (9 coefficient) RGB spherical harmonics.
type SH9 [9][3]float64

// ProjectHemisphereSH builds a cheap sky/ground SH (Gonum weights).
// This is IBL irradiance, not a full environment capture.
func ProjectHemisphereSH(skyR, skyG, skyB, grR, grG, grB float64) SH9 {
	var sh SH9
	// L0
	sh[0][0] = (skyR + grR) * 0.5 * 0.282095
	sh[0][1] = (skyG + grG) * 0.5 * 0.282095
	sh[0][2] = (skyB + grB) * 0.5 * 0.282095
	// L1 Y (up)
	dy := (skyR - grR) * 0.325735
	sh[2][0] = (skyR - grR) * 0.325735
	sh[2][1] = (skyG - grG) * 0.325735
	sh[2][2] = (skyB - grB) * 0.325735
	_ = dy
	return sh
}

// EvalSH evaluates SH9 at a unit direction (nx,ny,nz).
func EvalSH(sh SH9, nx, ny, nz float64) (r, g, b float64) {
	len2 := nx*nx + ny*ny + nz*nz
	if len2 > 1e-12 {
		inv := 1 / math.Sqrt(len2)
		nx *= inv
		ny *= inv
		nz *= inv
	}
	y := [9]float64{
		0.282095,
		0.488603 * ny,
		0.488603 * nz,
		0.488603 * nx,
		1.092548 * nx * ny,
		1.092548 * ny * nz,
		0.315392 * (3*nz*nz - 1),
		1.092548 * nx * nz,
		0.546274 * (nx*nx - ny*ny),
	}
	for i := 0; i < 9; i++ {
		r += sh[i][0] * y[i]
		g += sh[i][1] * y[i]
		b += sh[i][2] * y[i]
	}
	if r < 0 {
		r = 0
	}
	if g < 0 {
		g = 0
	}
	if b < 0 {
		b = 0
	}
	return r, g, b
}

// FBM is value-noise fractal Brownian motion (terrain / streaming).
func FBM(x, z float64, octaves int, persist, lacunarity float64) float64 {
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
	amp := 1.0
	freq := 1.0
	sum := 0.0
	norm := 0.0
	for i := 0; i < octaves; i++ {
		sum += amp * valueNoise(x*freq, z*freq)
		norm += amp
		amp *= persist
		freq *= lacunarity
	}
	if norm <= 0 {
		return 0
	}
	return sum / norm
}

func valueNoise(x, z float64) float64 {
	x0 := math.Floor(x)
	z0 := math.Floor(z)
	tx := fade(x - x0)
	tz := fade(z - z0)
	n00 := hash2(int(x0), int(z0))
	n10 := hash2(int(x0)+1, int(z0))
	n01 := hash2(int(x0), int(z0)+1)
	n11 := hash2(int(x0)+1, int(z0)+1)
	nx0 := n00 + (n10-n00)*tx
	nx1 := n01 + (n11-n01)*tx
	return nx0 + (nx1-nx0)*tz
}

func fade(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }

func hash2(x, z int) float64 {
	n := x*374761393 + z*668265263
	n = (n ^ (n >> 13)) * 1274126177
	n = n ^ (n >> 16)
	return float64(n&0xffff)/65535*2 - 1
}

// Separate2D is crowd-style 2D separation: returns (ax, az) push for agent i.
func Separate2D(xs, zs []float64, i int, sep float64) (ax, az float64) {
	if i < 0 || i >= len(xs) || i >= len(zs) || sep <= 0 {
		return 0, 0
	}
	px, pz := xs[i], zs[i]
	for j := range xs {
		if j == i || j >= len(zs) {
			continue
		}
		dx, dz := px-xs[j], pz-zs[j]
		d := math.Hypot(dx, dz)
		if d > 1e-6 && d < sep {
			s := (sep - d) / d
			ax += dx * s
			az += dz * s
		}
	}
	return ax, az
}

// Avoid2D is a time-to-collision sidestep for agent i (horizon seconds).
func Avoid2D(xs, zs, vxs, vzs []float64, i int, radius, horizon float64) (ax, az float64) {
	if i < 0 || i >= len(xs) || i >= len(zs) || radius <= 0 || horizon <= 0 {
		return 0, 0
	}
	if i >= len(vxs) || i >= len(vzs) {
		return 0, 0
	}
	px, pz := xs[i], zs[i]
	vx, vz := vxs[i], vzs[i]
	lim := radius * 2
	for j := range xs {
		if j == i || j >= len(zs) || j >= len(vxs) || j >= len(vzs) {
			continue
		}
		rx, rz := px-xs[j], pz-zs[j]
		rvx, rvz := vx-vxs[j], vz-vzs[j]
		a := rvx*rvx + rvz*rvz
		if a < 1e-8 {
			continue
		}
		b := rx*rvx + rz*rvz
		c := rx*rx + rz*rz - lim*lim
		disc := b*b - a*c
		if disc <= 0 {
			continue
		}
		t := (-b - math.Sqrt(disc)) / a
		if t <= 0 || t > horizon {
			continue
		}
		fx := rx + rvx*t
		fz := rz + rvz*t
		d := math.Hypot(fx, fz)
		if d < 1e-6 {
			fx, fz = -rz, rx
			d = math.Hypot(fx, fz)
		}
		if d < 1e-6 {
			continue
		}
		w := (horizon - t) / horizon
		ax += (fx / d) * w
		az += (fz / d) * w
	}
	return ax, az
}
