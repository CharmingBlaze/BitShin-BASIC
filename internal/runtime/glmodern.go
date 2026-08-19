package runtime

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/g3n/engine/gls"
	"github.com/go-gl/gl/v3.3-core/gl"
	gl4 "github.com/go-gl/gl/v4.6-core/gl"

	"bitshinbasic/internal/glslang"
	"bitshinbasic/internal/mathx"
)

// Required baseline is OpenGL 3.3 core. G3N and extra windows request 3.3 —
// never 4.5. 4.x APIs are detected after context create and never required.
const glRequiredMajor = 3
const glRequiredMinor = 3

type glCaps struct {
	ready                              bool
	major, minor                       int
	version, renderer, vendor, glsl    string
	compute, ssbo, ubo, instance, geom bool
	tess, imageLoad                    bool
}

type glCompute struct {
	prog uint32
	log  string
}

type glBuf struct {
	kind    int // 0 storage, 1 uniform
	handle  uint32
	data    []float32
	binding int
	gpu     bool
}

type glUserShader struct {
	stage string
	src   string
	log   string
	ok    bool
	spirv []uint32
}

type geomPoints struct {
	count      int
	data       []float32 // x y z size r g b per point
	vao, vbo   uint32
	dirty      bool
}

type glModern struct {
	caps                               glCaps
	skipped                            map[string]bool
	gpuInst                            bool
	tessOn                             bool
	gl4OK                              bool
	logged                             bool
	sh                                 mathx.SH9
	computes                           map[int]*glCompute
	freeComp                           []int
	nextComp                           int
	bufs                               map[int]*glBuf
	freeBuf                            []int
	nextBuf                            int
	shaders                            map[int]*glUserShader
	freeSh                             []int
	nextSh                             int
	geoms                              map[int]*geomPoints
	freeGeom                           []int
	nextGeom                           int
	instProg, instDepth, geomProg      uint32
	tessProg                           uint32
	instOK, instDepthOK, geomOK, tessOK bool
	instTried, geomTried, tessTried    bool
}

func (w *World) ensureGLMod() *glModern {
	if w.glmod.skipped == nil {
		w.glmod.skipped = map[string]bool{}
	}
	if w.glmod.computes == nil {
		w.glmod.computes = map[int]*glCompute{}
	}
	if w.glmod.bufs == nil {
		w.glmod.bufs = map[int]*glBuf{}
	}
	if w.glmod.shaders == nil {
		w.glmod.shaders = map[int]*glUserShader{}
	}
	if w.glmod.geoms == nil {
		w.glmod.geoms = map[int]*geomPoints{}
	}
	if w.glmod.nextComp < 1 {
		w.glmod.nextComp = 1
	}
	if w.glmod.nextBuf < 1 {
		w.glmod.nextBuf = 1
	}
	if w.glmod.nextSh < 1 {
		w.glmod.nextSh = 1
	}
	if w.glmod.nextGeom < 1 {
		w.glmod.nextGeom = 1
	}
	return &w.glmod
}

func (w *World) skipGL(feature, need string) {
	m := w.ensureGLMod()
	if m.skipped[feature] {
		return
	}
	have := m.caps.version
	if have == "" {
		have = "no context / OpenGL 3.3"
	}
	fmt.Printf("glmodern: %s skipped (need %s; have %s)\n", feature, need, have)
	m.skipped[feature] = true
}

func (w *World) detectModernGL() {
	m := w.ensureGLMod()
	if m.caps.ready {
		return
	}
	if err := gl.Init(); err != nil {
		fmt.Println("glmodern: gl.Init", err)
		return
	}
	if w.app != nil {
		if gs := w.app.Gls(); gs != nil {
			gs.SetCheckErrors(false)
		}
	}
	var maj, min int32
	gl.GetIntegerv(gl.MAJOR_VERSION, &maj)
	gl.GetIntegerv(gl.MINOR_VERSION, &min)
	m.caps.major, m.caps.minor = int(maj), int(min)
	m.caps.version = gl.GoStr(gl.GetString(gl.VERSION))
	m.caps.renderer = gl.GoStr(gl.GetString(gl.RENDERER))
	m.caps.vendor = gl.GoStr(gl.GetString(gl.VENDOR))
	m.caps.glsl = gl.GoStr(gl.GetString(gl.SHADING_LANGUAGE_VERSION))
	exts := glExtensions()
	ver43 := maj > 4 || (maj == 4 && min >= 3)
	ver40 := maj > 4 || (maj == 4 && min >= 0)
	ver31 := maj > 3 || (maj == 3 && min >= 1)
	ver32 := maj > 3 || (maj == 3 && min >= 2)
	m.caps.instance = ver31 || hasExt(exts, "GL_ARB_draw_instanced")
	m.caps.ubo = ver31 || hasExt(exts, "GL_ARB_uniform_buffer_object")
	m.caps.geom = ver32 || hasExt(exts, "GL_ARB_geometry_shader4")
	m.caps.tess = ver40 || hasExt(exts, "GL_ARB_tessellation_shader")
	m.caps.compute = ver43 || hasExt(exts, "GL_ARB_compute_shader")
	m.caps.ssbo = ver43 || hasExt(exts, "GL_ARB_shader_storage_buffer_object")
	m.caps.imageLoad = ver43 || hasExt(exts, "GL_ARB_shader_image_load_store")
	m.caps.ready = true
	m.gpuInst = m.caps.instance
	if m.caps.compute {
		if err := gl4.Init(); err != nil {
			fmt.Println("glmodern: gl4.Init failed; compute/SSBO disabled:", err)
			m.caps.compute = false
			m.caps.ssbo = false
			m.gl4OK = false
		} else {
			m.gl4OK = true
		}
	}
	if !m.logged {
		fmt.Printf("glmodern: OpenGL %d.%d (%s) instance=%d geom=%d ubo=%d compute=%d ssbo=%d tess=%d\n",
			m.caps.major, m.caps.minor, m.caps.renderer,
			b01(m.caps.instance), b01(m.caps.geom), b01(m.caps.ubo),
			b01(m.caps.compute), b01(m.caps.ssbo), b01(m.caps.tess))
		m.logged = true
	}
	for gl.GetError() != gl.NO_ERROR {
	}
}

func b01(v bool) int {
	if v {
		return 1
	}
	return 0
}

func glExtensions() map[string]bool {
	out := map[string]bool{}
	var n int32
	gl.GetIntegerv(gl.NUM_EXTENSIONS, &n)
	for i := int32(0); i < n; i++ {
		s := gl.GoStr(gl.GetStringi(gl.EXTENSIONS, uint32(i)))
		if s != "" {
			out[s] = true
		}
	}
	return out
}

func hasExt(m map[string]bool, name string) bool { return m[name] }

func saveModernGL(gs *gls.GLS) func() {
	var vp [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &vp[0])
	var prog, vao, arrayBuf, ebo, ubo int32
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &prog)
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &vao)
	gl.GetIntegerv(gl.ARRAY_BUFFER_BINDING, &arrayBuf)
	if vao != 0 {
		gl.GetIntegerv(gl.ELEMENT_ARRAY_BUFFER_BINDING, &ebo)
	}
	gl.GetIntegerv(gl.UNIFORM_BUFFER_BINDING, &ubo)
	return func() {
		gl.UseProgram(uint32(prog))
		gl.BindVertexArray(uint32(vao))
		gl.BindBuffer(gl.ARRAY_BUFFER, uint32(arrayBuf))
		if vao != 0 && ebo != 0 {
			gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, uint32(ebo))
		}
		if ubo != 0 {
			gl.BindBuffer(gl.UNIFORM_BUFFER, uint32(ubo))
		}
		if vp[2] > 0 && vp[3] > 0 {
			// Raw GL — G3N's wrapper exits the process on any leftover error.
			gl.Viewport(vp[0], vp[1], vp[2], vp[3])
		}
		_ = gs
		drainGL("glmodern")
	}
}

func compileLink(stages ...struct {
	kind uint32
	src  string
}) (uint32, error) {
	prog := gl.CreateProgram()
	var shaders []uint32
	for _, s := range stages {
		sh, err := compileGLShader(s.kind, s.src)
		if err != nil {
			gl.DeleteProgram(prog)
			return 0, err
		}
		gl.AttachShader(prog, sh)
		shaders = append(shaders, sh)
	}
	gl.LinkProgram(prog)
	var ok int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &ok)
	for _, sh := range shaders {
		gl.DeleteShader(sh)
	}
	if ok == gl.FALSE {
		log := glInfoLog(prog, true)
		gl.DeleteProgram(prog)
		return 0, fmt.Errorf("%s", log)
	}
	return prog, nil
}

func compileLink4(stages ...struct {
	kind uint32
	src  string
}) (uint32, error) {
	prog := gl4.CreateProgram()
	var shaders []uint32
	for _, s := range stages {
		sh := gl4.CreateShader(s.kind)
		csrc, free := gl4.Strs(s.src + "\x00")
		gl4.ShaderSource(sh, 1, csrc, nil)
		free()
		gl4.CompileShader(sh)
		var ok int32
		gl4.GetShaderiv(sh, gl4.COMPILE_STATUS, &ok)
		if ok == gl4.FALSE {
			var n int32
			gl4.GetShaderiv(sh, gl4.INFO_LOG_LENGTH, &n)
			buf := make([]byte, maxI32(n, 1))
			if n > 1 {
				gl4.GetShaderInfoLog(sh, n, nil, &buf[0])
			}
			gl4.DeleteShader(sh)
			gl4.DeleteProgram(prog)
			return 0, fmt.Errorf("%s", strings.TrimSpace(string(buf)))
		}
		gl4.AttachShader(prog, sh)
		shaders = append(shaders, sh)
	}
	gl4.LinkProgram(prog)
	var ok int32
	gl4.GetProgramiv(prog, gl4.LINK_STATUS, &ok)
	for _, sh := range shaders {
		gl4.DeleteShader(sh)
	}
	if ok == gl4.FALSE {
		var n int32
		gl4.GetProgramiv(prog, gl4.INFO_LOG_LENGTH, &n)
		buf := make([]byte, maxI32(n, 1))
		if n > 1 {
			gl4.GetProgramInfoLog(prog, n, nil, &buf[0])
		}
		gl4.DeleteProgram(prog)
		return 0, fmt.Errorf("%s", strings.TrimSpace(string(buf)))
	}
	return prog, nil
}

func maxI32(n int32, d int32) int32 {
	if n > d {
		return n
	}
	return d
}

func (w *World) ensureInstProgram() bool {
	m := w.ensureGLMod()
	if m.instOK {
		return true
	}
	if m.instTried {
		return false
	}
	m.instTried = true
	if !m.caps.instance {
		w.skipGL("instancing GPU", "OpenGL 3.1 draw instanced")
		return false
	}
	p, err := compileLink(
		struct {
			kind uint32
			src  string
		}{gl.VERTEX_SHADER, mbInstVertex},
		struct {
			kind uint32
			src  string
		}{gl.FRAGMENT_SHADER, mbInstFragment},
	)
	if err != nil {
		fmt.Println("glmodern: instance program", err)
		return false
	}
	m.instProg = p
	m.instOK = true
	return true
}

func (w *World) ensureInstDepthProgram() bool {
	m := w.ensureGLMod()
	if m.instDepthOK {
		return true
	}
	if !m.caps.instance {
		return false
	}
	p, err := compileLink(
		struct {
			kind uint32
			src  string
		}{gl.VERTEX_SHADER, mbInstDepthVertex},
		struct {
			kind uint32
			src  string
		}{gl.FRAGMENT_SHADER, mbInstDepthFragment},
	)
	if err != nil {
		return false
	}
	m.instDepth = p
	m.instDepthOK = true
	return true
}

func (w *World) ensureGeomProgram() bool {
	m := w.ensureGLMod()
	if m.geomOK {
		return true
	}
	if m.geomTried {
		return false
	}
	m.geomTried = true
	if !m.caps.geom {
		w.skipGL("geometry shader", "OpenGL 3.2 geometry shaders")
		return false
	}
	p, err := compileLink(
		struct {
			kind uint32
			src  string
		}{gl.VERTEX_SHADER, mbGeomVertex},
		struct {
			kind uint32
			src  string
		}{gl.GEOMETRY_SHADER, mbGeomGeometry},
		struct {
			kind uint32
			src  string
		}{gl.FRAGMENT_SHADER, mbGeomFragment},
	)
	if err != nil {
		fmt.Println("glmodern: geometry program", err)
		return false
	}
	m.geomProg = p
	m.geomOK = true
	return true
}

func (w *World) ensureTessProgram() bool {
	m := w.ensureGLMod()
	if m.tessOK {
		return true
	}
	if m.tessTried {
		return false
	}
	m.tessTried = true
	if !m.caps.tess {
		w.skipGL("tessellation", "OpenGL 4.0 or GL_ARB_tessellation_shader")
		return false
	}
	p, err := compileLink(
		struct {
			kind uint32
			src  string
		}{gl.VERTEX_SHADER, mbTessVert},
		struct {
			kind uint32
			src  string
		}{0x8E88, mbTessCtrl}, // TESS_CONTROL_SHADER
		struct {
			kind uint32
			src  string
		}{0x8E87, mbTessEval}, // TESS_EVALUATION_SHADER
		struct {
			kind uint32
			src  string
		}{gl.FRAGMENT_SHADER, mbTessFrag},
	)
	if err != nil {
		w.skipGL("tessellation", "tess shaders failed to compile: "+err.Error())
		return false
	}
	m.tessProg = p
	m.tessOK = true
	return true
}

func (w *World) createComputeProg(src string) (int, string) {
	m := w.ensureGLMod()
	w.detectModernGL()
	if !m.caps.compute || !m.gl4OK {
		w.skipGL("compute", "OpenGL 4.3 or GL_ARB_compute_shader")
		return 0, "compute not available"
	}
	if strings.TrimSpace(src) == "" {
		src = mbComputeFill
	}
	vr := glslang.Validate(src, glslang.StageComp)
	if !vr.OK {
		return 0, vr.Log
	}
	p, err := compileLink4(struct {
		kind uint32
		src  string
	}{gl4.COMPUTE_SHADER, src})
	if err != nil {
		return 0, err.Error()
	}
	id := w.takeHandle(&m.freeComp, &m.nextComp)
	m.computes[id] = &glCompute{prog: p, log: "ok"}
	return id, "ok"
}

func (w *World) dispatchCompute(id, gx, gy, gz int) int {
	m := w.ensureGLMod()
	w.detectModernGL()
	if !m.caps.compute || !m.gl4OK {
		w.skipGL("DispatchCompute", "OpenGL 4.3 or GL_ARB_compute_shader")
		return 0
	}
	c := m.computes[id]
	if c == nil || c.prog == 0 {
		return 0
	}
	if gx < 1 {
		gx = 1
	}
	if gy < 1 {
		gy = 1
	}
	if gz < 1 {
		gz = 1
	}
	restore := saveModernGL(nil)
	defer restore()
	gl4.UseProgram(c.prog)
	if loc := gl4.GetUniformLocation(c.prog, gl4.Str("Time\x00")); loc >= 0 {
		gl4.Uniform1f(loc, float32(w.delta)*60)
	}
	if loc := gl4.GetUniformLocation(c.prog, gl4.Str("Count\x00")); loc >= 0 {
		n := uint32(0)
		for _, b := range m.bufs {
			if b != nil && b.kind == 0 {
				n = uint32(len(b.data))
				break
			}
		}
		gl4.Uniform1ui(loc, n)
	}
	gl4.DispatchCompute(uint32(gx), uint32(gy), uint32(gz))
	gl4.MemoryBarrier(gl4.SHADER_STORAGE_BARRIER_BIT | gl4.BUFFER_UPDATE_BARRIER_BIT)
	return 1
}

func (w *World) createGLBuffer(kind, count int) int {
	m := w.ensureGLMod()
	w.detectModernGL()
	if count < 1 {
		count = 4
	}
	if count > 1<<20 {
		count = 1 << 20
	}
	b := &glBuf{kind: kind, data: make([]float32, count)}
	wantGPU := (kind == 0 && m.caps.ssbo && m.gl4OK) || (kind == 1 && m.caps.ubo)
	if kind == 0 && !m.caps.ssbo {
		w.skipGL("SSBO GPU", "OpenGL 4.3 or GL_ARB_shader_storage_buffer_object")
	}
	if wantGPU {
		var h uint32
		if kind == 0 {
			gl4.GenBuffers(1, &h)
			gl4.BindBuffer(gl4.SHADER_STORAGE_BUFFER, h)
			gl4.BufferData(gl4.SHADER_STORAGE_BUFFER, count*4, gl.Ptr(b.data), gl4.DYNAMIC_DRAW)
			gl4.BindBuffer(gl4.SHADER_STORAGE_BUFFER, 0)
		} else {
			gl.GenBuffers(1, &h)
			gl.BindBuffer(gl.UNIFORM_BUFFER, h)
			gl.BufferData(gl.UNIFORM_BUFFER, count*4, gl.Ptr(b.data), gl.DYNAMIC_DRAW)
			gl.BindBuffer(gl.UNIFORM_BUFFER, 0)
		}
		b.handle = h
		b.gpu = h != 0
	}
	id := w.takeHandle(&m.freeBuf, &m.nextBuf)
	m.bufs[id] = b
	return id
}

func (w *World) setGLBuffer(id, index int, values []float32) {
	m := w.ensureGLMod()
	b := m.bufs[id]
	if b == nil || index < 0 {
		return
	}
	for i, v := range values {
		if index+i >= len(b.data) {
			break
		}
		b.data[index+i] = v
	}
	w.uploadGLBuffer(b)
}

func (w *World) uploadGLBuffer(b *glBuf) {
	if b == nil || !b.gpu || b.handle == 0 || len(b.data) == 0 {
		return
	}
	if b.kind == 0 && w.glmod.gl4OK {
		gl4.BindBuffer(gl4.SHADER_STORAGE_BUFFER, b.handle)
		gl4.BufferSubData(gl4.SHADER_STORAGE_BUFFER, 0, len(b.data)*4, gl.Ptr(b.data))
		gl4.BindBuffer(gl4.SHADER_STORAGE_BUFFER, 0)
		return
	}
	if b.kind == 1 {
		gl.BindBuffer(gl.UNIFORM_BUFFER, b.handle)
		gl.BufferSubData(gl.UNIFORM_BUFFER, 0, len(b.data)*4, gl.Ptr(b.data))
		gl.BindBuffer(gl.UNIFORM_BUFFER, 0)
	}
}

func (w *World) bindGLBuffer(id, binding int) int {
	m := w.ensureGLMod()
	w.detectModernGL()
	b := m.bufs[id]
	if b == nil {
		return 0
	}
	b.binding = binding
	if !b.gpu {
		if b.kind == 0 {
			w.skipGL("BindStorageBuffer", "OpenGL 4.3 SSBO")
		} else {
			w.skipGL("BindUniformBuffer", "OpenGL 3.1 UBO")
		}
		return 0
	}
	if b.kind == 0 && m.gl4OK {
		gl4.BindBufferBase(gl4.SHADER_STORAGE_BUFFER, uint32(binding), b.handle)
		return 1
	}
	if b.kind == 1 {
		gl.BindBufferBase(gl.UNIFORM_BUFFER, uint32(binding), b.handle)
		return 1
	}
	return 0
}

func (w *World) getGLBuffer(id, index int) float64 {
	m := w.ensureGLMod()
	b := m.bufs[id]
	if b == nil || index < 0 || index >= len(b.data) {
		return 0
	}
	if b.gpu && b.handle != 0 {
		if b.kind == 0 && m.gl4OK {
			gl4.BindBuffer(gl4.SHADER_STORAGE_BUFFER, b.handle)
			gl4.GetBufferSubData(gl4.SHADER_STORAGE_BUFFER, index*4, 4, gl.Ptr(&b.data[index]))
			gl4.BindBuffer(gl4.SHADER_STORAGE_BUFFER, 0)
		} else if b.kind == 1 {
			gl.BindBuffer(gl.UNIFORM_BUFFER, b.handle)
			gl.GetBufferSubData(gl.UNIFORM_BUFFER, index*4, 4, gl.Ptr(&b.data[index]))
			gl.BindBuffer(gl.UNIFORM_BUFFER, 0)
		}
	}
	return float64(b.data[index])
}

func (w *World) compileUserShader(src, stage string) int {
	m := w.ensureGLMod()
	st := glslang.ParseStage(stage)
	r := glslang.Compile(src, st)
	id := w.takeHandle(&m.freeSh, &m.nextSh)
	u := &glUserShader{stage: string(st), src: src, log: r.Log, ok: r.OK, spirv: r.SPIRV}
	if r.OK && w.app != nil && w.glmod.caps.ready {
		kind := uint32(gl.VERTEX_SHADER)
		switch st {
		case glslang.StageFrag:
			kind = gl.FRAGMENT_SHADER
		case glslang.StageGeom:
			kind = gl.GEOMETRY_SHADER
		case glslang.StageComp:
			if m.caps.compute && m.gl4OK {
				if _, err := compileLink4(struct {
					kind uint32
					src  string
				}{gl4.COMPUTE_SHADER, src}); err != nil {
					u.ok = false
					u.log = err.Error()
				} else {
					u.log = "ok (GL compile)"
				}
				m.shaders[id] = u
				return id
			}
			w.skipGL("CompileShader compute", "OpenGL 4.3")
			m.shaders[id] = u
			return id
		case glslang.StageTesc:
			kind = 0x8E88
		case glslang.StageTese:
			kind = 0x8E87
		}
		if sh, err := compileGLShader(kind, src); err != nil {
			u.ok = false
			u.log = err.Error()
		} else {
			gl.DeleteShader(sh)
			u.log = "ok (GL compile)"
		}
	}
	m.shaders[id] = u
	return id
}

func parseGLVersion(s string) (major, minor int) {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i > 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return major, minor
}
