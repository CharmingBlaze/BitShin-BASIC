package runtime

import (
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/mathx"
	"bitshinbasic/internal/value"
)

type lightProbe struct {
	x, y, z float32
	sky     math32.Color
	ground  math32.Color
}

func (w *World) applyProbes() {
	if len(w.probes) == 0 {
		w.shaderUnis["ProbeEnabled"] = shaderUni{n: 1, v: [4]float32{0}}
		return
	}
	x, y, z := float32(0), float32(2), float32(0)
	if w.cam != nil {
		p := worldPos(w.cam.GetNode())
		x, y, z = fromG3N(p.X, p.Y, p.Z)
	}
	var sky, ground math32.Color
	wsum := float32(0)
	for _, pr := range w.probes {
		if pr == nil {
			continue
		}
		dx, dy, dz := pr.x-x, pr.y-y, pr.z-z
		d2 := dx*dx + dy*dy + dz*dz
		wt := 1 / (d2 + 0.35)
		sky.R += pr.sky.R * wt
		sky.G += pr.sky.G * wt
		sky.B += pr.sky.B * wt
		ground.R += pr.ground.R * wt
		ground.G += pr.ground.G * wt
		ground.B += pr.ground.B * wt
		wsum += wt
	}
	if wsum <= 0 {
		return
	}
	sky.R /= wsum
	sky.G /= wsum
	sky.B /= wsum
	ground.R /= wsum
	ground.G /= wsum
	ground.B /= wsum
	if w.ambient != nil {
		mix := math32.Color{(sky.R + ground.R) * 0.5, (sky.G + ground.G) * 0.5, (sky.B + ground.B) * 0.5}
		w.ambient.SetColor(&mix)
	}
	w.shaderUnis["ProbeEnabled"] = shaderUni{n: 1, v: [4]float32{1}}
	w.shaderUnis["ProbeSky"] = shaderUni{n: 3, v: [4]float32{sky.R, sky.G, sky.B}}
	w.shaderUnis["ProbeGround"] = shaderUni{n: 3, v: [4]float32{ground.R, ground.G, ground.B}}
	w.glmod.sh = mathx.ProjectHemisphereSH(float64(sky.R), float64(sky.G), float64(sky.B), float64(ground.R), float64(ground.G), float64(ground.B))
}

func (w *World) probeCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createlightprobe": n(func(a []value.Value) (value.Value, error) {
			if w.probes == nil {
				w.probes = map[int]*lightProbe{}
			}
			id := w.takeHandle(&w.freeProbes, &w.nextProbe)
			pr := &lightProbe{
				x: float32(argN(a, 0, 0)), y: float32(argN(a, 1, 2)), z: float32(argN(a, 2, 0)),
				sky:    math32.Color{0.22, 0.28, 0.40},
				ground: math32.Color{0.16, 0.12, 0.09},
			}
			if len(a) >= 6 {
				pr.sky = *rgb(argN(a, 3, 80), argN(a, 4, 110), argN(a, 5, 160))
			}
			w.probes[id] = pr
			return value.Num(float64(id)), nil
		}),
		"setprobecolor": n(func(a []value.Value) (value.Value, error) {
			pr := w.probes[argI(a, 0, 0)]
			if pr == nil {
				return z()
			}
			pr.sky = *rgb(argN(a, 1, 80), argN(a, 2, 110), argN(a, 3, 160))
			pr.ground = *rgb(argN(a, 4, 40), argN(a, 5, 30), argN(a, 6, 22))
			return z()
		}),
		"setprobegrid": n(func(a []value.Value) (value.Value, error) {
			ox, oy, oz := float32(argN(a, 0, 0)), float32(argN(a, 1, 2)), float32(argN(a, 2, 0))
			nx, ny, nz := argI(a, 3, 2), argI(a, 4, 1), argI(a, 5, 2)
			sp := float32(argN(a, 6, 16))
			if nx < 1 {
				nx = 1
			}
			if ny < 1 {
				ny = 1
			}
			if nz < 1 {
				nz = 1
			}
			if w.probes == nil {
				w.probes = map[int]*lightProbe{}
			}
			last := 0
			for iz := 0; iz < nz; iz++ {
				for iy := 0; iy < ny; iy++ {
					for ix := 0; ix < nx; ix++ {
						id := w.takeHandle(&w.freeProbes, &w.nextProbe)
						t := float32(iy) / float32(max(ny-1, 1))
						w.probes[id] = &lightProbe{
							x: ox + float32(ix)*sp, y: oy + float32(iy)*sp, z: oz + float32(iz)*sp,
							sky:    math32.Color{0.18 + t*0.12, 0.24 + t*0.10, 0.36 + t*0.08},
							ground: math32.Color{0.14, 0.11, 0.08},
						}
						last = id
					}
				}
			}
			return value.Num(float64(last)), nil
		}),
		"sampleprobe": n(func(a []value.Value) (value.Value, error) {
			w.applyProbes()
			return z()
		}),
		"setlightmap": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if t := w.texs[argI(a, 1, 0)]; t != nil && e.mat != nil {
				e.mat.AddTexture(t.tex)
			}
			return z()
		}),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
