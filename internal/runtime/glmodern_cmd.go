package runtime

import (
	"strconv"
	"strings"

	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/glslang"
	"bitshinbasic/internal/mathx"
	"bitshinbasic/internal/value"
)

func (w *World) glmodernCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"glversion": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			s := w.glmod.caps.version
			if s == "" {
				s = "OpenGL 3.3 required (no context yet)"
			}
			return value.Str(s), nil
		}),
		"glmajor": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			return value.Num(float64(w.glmod.caps.major)), nil
		}),
		"glminor": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			return value.Num(float64(w.glmod.caps.minor)), nil
		}),
		"glrenderer": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			return value.Str(w.glmod.caps.renderer), nil
		}),
		"glhascompute": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			return value.Num(float64(b01(w.glmod.caps.compute))), nil
		}),
		"glhasssbo": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			return value.Num(float64(b01(w.glmod.caps.ssbo))), nil
		}),
		"glhasubo": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			return value.Num(float64(b01(w.glmod.caps.ubo))), nil
		}),
		"glhasinstancing": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			return value.Num(float64(b01(w.glmod.caps.instance))), nil
		}),
		"glhasgeometry": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			return value.Num(float64(b01(w.glmod.caps.geom))), nil
		}),
		"glhastessellation": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			return value.Num(float64(b01(w.glmod.caps.tess))), nil
		}),
		"glfeature": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			name := strings.ToLower(argS(a, 0))
			ok := false
			switch name {
			case "compute":
				ok = w.glmod.caps.compute
			case "ssbo", "storage":
				ok = w.glmod.caps.ssbo
			case "ubo", "uniform":
				ok = w.glmod.caps.ubo
			case "instance", "instancing":
				ok = w.glmod.caps.instance
			case "geom", "geometry":
				ok = w.glmod.caps.geom
			case "tess", "tessellation":
				ok = w.glmod.caps.tess
			}
			return value.Num(float64(b01(ok))), nil
		}),
		"createcomputeshader": need(func(a []value.Value) (value.Value, error) {
			id, _ := w.createComputeProg(argS(a, 0))
			return value.Num(float64(id)), nil
		}),
		"dispatchcompute": n(func(a []value.Value) (value.Value, error) {
			ok := w.dispatchCompute(argI(a, 0, 0), argI(a, 1, 1), argI(a, 2, 1), argI(a, 3, 1))
			return value.Num(float64(ok)), nil
		}),
		"computelog": n(func(a []value.Value) (value.Value, error) {
			c := w.ensureGLMod().computes[argI(a, 0, 0)]
			if c == nil {
				return value.Str(""), nil
			}
			return value.Str(c.log), nil
		}),
		"createstoragebuffer": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.createGLBuffer(0, argI(a, 0, 64)))), nil
		}),
		"setstoragebuffer": n(func(a []value.Value) (value.Value, error) {
			vals := make([]float32, 0, len(a)-2)
			for i := 2; i < len(a); i++ {
				vals = append(vals, float32(argN(a, i, 0)))
			}
			w.setGLBuffer(argI(a, 0, 0), argI(a, 1, 0), vals)
			return z()
		}),
		"getstoragebuffer": n(func(a []value.Value) (value.Value, error) {
			return value.Num(w.getGLBuffer(argI(a, 0, 0), argI(a, 1, 0))), nil
		}),
		"bindstoragebuffer": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.bindGLBuffer(argI(a, 0, 0), argI(a, 1, 0)))), nil
		}),
		"storagebuffersize": n(func(a []value.Value) (value.Value, error) {
			b := w.ensureGLMod().bufs[argI(a, 0, 0)]
			if b == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(len(b.data))), nil
		}),
		"createuniformbuffer": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.createGLBuffer(1, argI(a, 0, 16)))), nil
		}),
		"setuniformbuffer": n(func(a []value.Value) (value.Value, error) {
			vals := make([]float32, 0, len(a)-2)
			for i := 2; i < len(a); i++ {
				vals = append(vals, float32(argN(a, i, 0)))
			}
			w.setGLBuffer(argI(a, 0, 0), argI(a, 1, 0), vals)
			return z()
		}),
		"getuniformbuffer": n(func(a []value.Value) (value.Value, error) {
			return value.Num(w.getGLBuffer(argI(a, 0, 0), argI(a, 1, 0))), nil
		}),
		"binduniformbuffer": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.bindGLBuffer(argI(a, 0, 0), argI(a, 1, 0)))), nil
		}),
		"enablegpuinstances": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			on := argI(a, 0, 1) != 0
			if on && !w.glmod.caps.instance {
				w.skipGL("EnableGPUInstances", "OpenGL 3.1 instancing")
				w.glmod.gpuInst = false
				return value.Num(0), nil
			}
			w.glmod.gpuInst = on
			for id, im := range w.instances {
				if im != nil {
					im.dirty = true
					_ = id
				}
			}
			return value.Num(float64(b01(w.glmod.gpuInst))), nil
		}),
		"gpuinstances": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(b01(w.glmod.gpuInst && w.glmod.caps.instance))), nil
		}),
		"creategeompoints": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.createGeomPoints(argI(a, 0, 8)))), nil
		}),
		"setgeompoint": n(func(a []value.Value) (value.Value, error) {
			g := w.ensureGLMod().geoms[argI(a, 0, 0)]
			i := argI(a, 1, 0)
			if g == nil || i < 0 || i >= g.count {
				return z()
			}
			gx, gy, gz := toG3N(float32(argN(a, 2, 0)), float32(argN(a, 3, 0)), float32(argN(a, 4, 0)))
			g.data[i*7+0] = gx
			g.data[i*7+1] = gy
			g.data[i*7+2] = gz
			if len(a) > 5 {
				g.data[i*7+3] = float32(argN(a, 5, 0.25))
			}
			if len(a) > 8 {
				c := rgb(argN(a, 6, 255), argN(a, 7, 255), argN(a, 8, 255))
				g.data[i*7+4] = c.R
				g.data[i*7+5] = c.G
				g.data[i*7+6] = c.B
			}
			g.dirty = true
			return z()
		}),
		"geompointcount": n(func(a []value.Value) (value.Value, error) {
			g := w.ensureGLMod().geoms[argI(a, 0, 0)]
			if g == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(g.count)), nil
		}),
		"enabletessellation": n(func(a []value.Value) (value.Value, error) {
			w.maybeDetect()
			on := argI(a, 0, 1) != 0
			if on && !w.ensureTessProgram() {
				w.glmod.tessOn = false
				return value.Num(0), nil
			}
			w.glmod.tessOn = on && w.glmod.tessOK
			return value.Num(float64(b01(w.glmod.tessOn))), nil
		}),
		"tessellation": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(b01(w.glmod.tessOn && w.glmod.tessOK))), nil
		}),
		"compileshader": n(func(a []value.Value) (value.Value, error) {
			src := argS(a, 0)
			stage := argS(a, 1)
			if stage == "" {
				stage = "vert"
			}
			if w.app != nil {
				w.detectModernGL()
			}
			id := w.compileUserShader(src, stage)
			return value.Num(float64(id)), nil
		}),
		"shaderlog": n(func(a []value.Value) (value.Value, error) {
			u := w.ensureGLMod().shaders[argI(a, 0, 0)]
			if u == nil {
				return value.Str(""), nil
			}
			return value.Str(u.log), nil
		}),
		"shaderspirvsize": n(func(a []value.Value) (value.Value, error) {
			u := w.ensureGLMod().shaders[argI(a, 0, 0)]
			if u == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(len(u.spirv))), nil
		}),
		"glslangnative": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(b01(glslang.HasNative()))), nil
		}),
		"noisefbm": n(func(a []value.Value) (value.Value, error) {
			return value.Num(mathx.FBM(argN(a, 0, 0), argN(a, 1, 0), argI(a, 2, 4), argN(a, 3, 0.5), argN(a, 4, 2))), nil
		}),
		"sheval": n(func(a []value.Value) (value.Value, error) {
			r, _, _ := mathx.EvalSH(w.glmod.sh, argN(a, 0, 0), argN(a, 1, 1), argN(a, 2, 0))
			return value.Num(r), nil
		}),
		"jobxform": n(func(a []value.Value) (value.Value, error) {
			nmat := argI(a, 0, 64)
			if nmat < 1 {
				nmat = 1
			}
			if nmat > 8000 {
				nmat = 8000
			}
			src := make([]float64, nmat*16)
			for i := 0; i < nmat; i++ {
				src[i*16+0] = 1
				src[i*16+5] = 1
				src[i*16+10] = 1
				src[i*16+15] = 1
			}
			var ident math32.Matrix4
			ident.Identity()
			id := w.ensureJobs().submit(func() {
				_ = mathx.BatchMul4(src, nmat, mathx.FromMat4(ident))
			})
			return value.Num(float64(id)), nil
		}),
	}
}

func (w *World) maybeDetect() {
	if w.app != nil && !w.glmod.caps.ready {
		w.detectModernGL()
	}
}

func glFeatureName(caps glCaps) string {
	return strconv.Itoa(caps.major) + "." + strconv.Itoa(caps.minor)
}
