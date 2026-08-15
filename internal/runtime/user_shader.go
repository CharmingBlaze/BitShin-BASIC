package runtime

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/glslang"
	"bitshinbasic/internal/value"
)

// Shader-graph lite: GLSL 330 vert+frag programs bound to a mesh.
// Not an Unreal node graph. PBR/Phong stay on their own materials.

type userShader struct {
	name   string
	vert   string
	frag   string
	unis   map[string]shaderUni
	log    string
	ok     bool
	reg    bool
}

type userMat struct {
	*material.Standard
	w  *World
	sh *userShader
}

func (m *userMat) GetMaterial() *material.Material { return m.Standard.GetMaterial() }
func (m *userMat) Dispose()                        { m.Standard.Dispose() }

func (m *userMat) RenderSetup(gs *gls.GLS) {
	m.Standard.RenderSetup(gs)
	if m.w == nil {
		return
	}
	m.w.applyUserUniforms(gs)
	if m.sh != nil {
		for name, u := range m.sh.unis {
			uni := gls.Uniform{}
			uni.Init(name)
			loc := uni.Location(gs)
			if loc < 0 {
				continue
			}
			switch u.n {
			case 1:
				gs.Uniform1f(loc, u.v[0])
			case 2:
				gs.Uniform2f(loc, u.v[0], u.v[1])
			case 3:
				gs.Uniform3f(loc, u.v[0], u.v[1], u.v[2])
			default:
				gs.Uniform4f(loc, u.v[0], u.v[1], u.v[2], u.v[3])
			}
		}
		uni := gls.Uniform{}
		uni.Init("Time")
		if loc := uni.Location(gs); loc >= 0 {
			gs.Uniform1f(loc, float32(wTime(m.w)))
		}
	}
}

func wTime(w *World) float64 {
	if w == nil {
		return 0
	}
	return time.Since(w.started).Seconds()
}

func (w *World) shaderCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createshader": need(func(a []value.Value) (value.Value, error) {
			vert, frag := argS(a, 0), argS(a, 1)
			if looksShaderFile(vert) && looksShaderFile(frag) {
				return w.loadShaderFiles(vert, frag)
			}
			if vert == "" || frag == "" {
				vert, frag = mbuserDefaultVert, mbuserDefaultFrag
			}
			id, err := w.createUserShader(vert, frag)
			return value.Num(float64(id)), err
		}),
		"loadshader": need(func(a []value.Value) (value.Value, error) {
			return w.loadShaderFiles(argS(a, 0), argS(a, 1))
		}),
		"setshader": need(func(a []value.Value) (value.Value, error) {
			return w.bindEntityShader(argI(a, 0, 0), argI(a, 1, 0))
		}),
		"entityshader": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.ushader)), nil
		}),
		"setshaderuniform": n(func(a []value.Value) (value.Value, error) {
			return w.setShaderUniform(a)
		}),
		"setuniform": n(func(a []value.Value) (value.Value, error) {
			return w.setShaderUniform(a)
		}),
		"shaderok": n(func(a []value.Value) (value.Value, error) {
			sh := w.ushaders[argI(a, 0, 0)]
			if sh == nil || !sh.ok {
				return value.Num(0), nil
			}
			return value.Num(1), nil
		}),
	}
}

func looksShaderFile(s string) bool {
	low := strings.ToLower(s)
	return strings.HasSuffix(low, ".vert") || strings.HasSuffix(low, ".frag") ||
		strings.HasSuffix(low, ".glsl") || strings.HasSuffix(low, ".vs") || strings.HasSuffix(low, ".fs")
}

func (w *World) loadShaderFiles(vertPath, fragPath string) (value.Value, error) {
	vb, err := w.readShaderFile(vertPath)
	if err != nil {
		return value.Value{}, err
	}
	fb, err := w.readShaderFile(fragPath)
	if err != nil {
		return value.Value{}, err
	}
	id, err := w.createUserShader(string(vb), string(fb))
	return value.Num(float64(id)), err
}

func (w *World) readShaderFile(rel string) ([]byte, error) {
	path, err := w.openPath(rel)
	if err != nil {
		path = w.resolve(rel)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("LoadShader: %w", err)
	}
	return b, nil
}

func stripGLSLVersion(src string) string {
	lines := strings.Split(src, "\n")
	out := lines[:0]
	for _, ln := range lines {
		trim := strings.TrimSpace(ln)
		if strings.HasPrefix(trim, "#version") {
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

func (w *World) createUserShader(vert, frag string) (int, error) {
	vert = stripGLSLVersion(vert)
	frag = stripGLSLVersion(frag)
	if w.ushaders == nil {
		w.ushaders = map[int]*userShader{}
	}
	id := w.takeHandle(&w.freeUSh, &w.nextUSh)
	name := fmt.Sprintf("mbuser_%d", id)
	sh := &userShader{name: name, vert: vert, frag: frag, unis: map[string]shaderUni{}, ok: true}
	if !strings.Contains(vert, "#include") && !strings.Contains(frag, "#include") {
		rv := glslang.Compile(vert, glslang.StageVert)
		rf := glslang.Compile(frag, glslang.StageFrag)
		if !rv.OK || !rf.OK {
			sh.ok = false
			sh.log = strings.TrimSpace(rv.Log + "\n" + rf.Log)
			fmt.Println("CreateShader:", sh.log)
		}
	}
	if err := w.registerUserShader(sh); err != nil {
		sh.ok = false
		sh.log = err.Error()
		fmt.Println("CreateShader:", err)
	}
	w.ushaders[id] = sh
	return id, nil
}

func (w *World) registerUserShader(sh *userShader) error {
	if sh == nil || sh.reg || w.app == nil || w.app.rend == nil {
		return nil
	}
	w.app.rend.AddShader(sh.name+"_vertex", sh.vert)
	w.app.rend.AddShader(sh.name+"_fragment", sh.frag)
	w.app.rend.AddProgram(sh.name, sh.name+"_vertex", sh.name+"_fragment")
	sh.reg = true
	return nil
}

func (w *World) bindEntityShader(entID, shID int) (value.Value, error) {
	e, err := w.ent(entID)
	if err != nil {
		return value.Value{}, err
	}
	if e.mesh == nil {
		return value.Value{}, fmt.Errorf("SetShader: entity has no mesh")
	}
	sh := w.ushaders[shID]
	if sh == nil {
		return value.Value{}, fmt.Errorf("SetShader: invalid shader %d", shID)
	}
	if err := w.registerUserShader(sh); err != nil {
		return value.Value{}, err
	}
	mat := material.NewStandard(&math32.Color{0.85, 0.55, 0.35})
	mat.SetShader(sh.name)
	um := &userMat{Standard: mat, w: w, sh: sh}
	e.mesh.SetMaterial(um)
	e.mat = mat
	e.ushader = shID
	e.usePBR = false
	return value.Num(0), nil
}

func (w *World) setShaderUniform(a []value.Value) (value.Value, error) {
	off := 0
	var sh *userShader
	if len(a) >= 2 && a[0].Kind != value.KindStr {
		sh = w.ushaders[argI(a, 0, 0)]
		off = 1
	}
	name := argS(a, off)
	if name == "" {
		return value.Value{}, fmt.Errorf("SetShaderUniform: missing name")
	}
	n := len(a) - off - 1
	if n < 1 {
		n = 1
	}
	if n > 4 {
		n = 4
	}
	var v [4]float32
	for i := 0; i < n; i++ {
		v[i] = float32(argN(a, off+1+i, 0))
	}
	u := shaderUni{n: n, v: v}
	if sh != nil {
		if sh.unis == nil {
			sh.unis = map[string]shaderUni{}
		}
		sh.unis[name] = u
	} else {
		if w.shaderUnis == nil {
			w.shaderUnis = map[string]shaderUni{}
		}
		w.shaderUnis[name] = u
	}
	return value.Num(0), nil
}

const mbuserDefaultVert = `#include <attributes>
uniform mat4 MVP;
uniform mat4 ModelMatrix;
uniform mat3 NormalMatrix;
out vec3 vN;
out vec3 vW;
out vec2 vUV;
void main() {
	vW = (ModelMatrix * vec4(VertexPosition, 1.0)).xyz;
	vN = normalize(NormalMatrix * VertexNormal);
	vUV = VertexTexcoord;
	gl_Position = MVP * vec4(VertexPosition, 1.0);
}
`

const mbuserDefaultFrag = `precision highp float;
in vec3 vN;
in vec3 vW;
in vec2 vUV;
uniform vec4 Color;
uniform vec3 LightDir;
uniform float Time;
out vec4 FragColor;
void main() {
	vec3 L = normalize(vec3(0.35, 0.85, 0.4));
	if (length(LightDir) > 0.01) { L = normalize(LightDir); }
	float d = max(dot(normalize(vN), L), 0.12);
	vec3 tint = Color.rgb;
	if (length(tint) < 0.01) { tint = vec3(0.9, 0.45, 0.2); }
	float pulse = 0.65 + 0.35 * sin(Time * 2.0 + vW.y);
	FragColor = vec4(tint * d * pulse, 1.0);
}
`
