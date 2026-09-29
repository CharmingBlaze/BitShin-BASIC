package runtime

import (
	"fmt"
	"reflect"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/texture"
)

// pbrMat is G3N Physical plus the same shadow atlas / fog uniforms as litMat.
type pbrMat struct {
	*material.Physical
	w          *World
	metallic   float32
	roughness  float32
	ao         float32
	albedo     math32.Color4
	emissive   math32.Color
	ibl        int // -1 world default, 0 off, 1 on
	envTex     *texture.Texture2D
	recvShadow bool
}

func (m *pbrMat) GetMaterial() *material.Material { return m.Physical.GetMaterial() }

func (m *pbrMat) Dispose() { m.Physical.Dispose() }

func (m *pbrMat) RenderSetup(gs *gls.GLS) {
	m.Physical.RenderSetup(gs)
	if m.w != nil {
		m.w.applyUserUniforms(gs)
		m.w.bindShadowUniforms(gs)
		m.w.bindFogUniforms(gs)
		m.w.bindPBRUniforms(gs, m)
		recv := 1
		if !m.recvShadow {
			recv = 0
		}
		setUni1i(gs, "MeshReceiveShadow", recv)
	}
}

func (m *pbrMat) applyFactors() {
	if m.Physical == nil {
		return
	}
	m.Physical.SetBaseColorFactor(&m.albedo)
	m.Physical.SetMetallicFactor(m.metallic)
	m.Physical.SetRoughnessFactor(m.roughness)
	m.Physical.SetEmissiveFactor(&m.emissive)
	m.Physical.SetTransparent(m.albedo.A < 1)
}

func (w *World) newPBR() *pbrMat {
	p := material.NewPhysical()
	p.SetShader("mbphysical")
	p.SetMetallicFactor(0)
	p.SetRoughnessFactor(0.5)
	if w.wire {
		p.SetWireframe(true)
	}
	m := &pbrMat{
		Physical:   p,
		w:          w,
		metallic:   0,
		roughness:  0.5,
		ao:         1,
		albedo:     math32.Color4{0.82, 0.84, 0.88, 1},
		emissive:   math32.Color{0, 0, 0},
		ibl:        -1,
		recvShadow: true,
	}
	m.applyFactors()
	return m
}

func (w *World) wrapPhysical(p *material.Physical) *pbrMat {
	if p == nil {
		return nil
	}
	p.SetShader("mbphysical")
	metal, rough, base, emit := readPhysical(p)
	return &pbrMat{
		Physical:   p,
		w:          w,
		metallic:   metal,
		roughness:  rough,
		ao:         1,
		albedo:     base,
		emissive:   emit,
		ibl:        -1,
		recvShadow: true,
	}
}

func readPhysical(p *material.Physical) (metal, rough float32, base math32.Color4, emit math32.Color) {
	metal, rough = 1, 1
	base = math32.Color4{1, 1, 1, 1}
	if p == nil {
		return
	}
	v := reflect.ValueOf(p).Elem().FieldByName("udata")
	if !v.IsValid() {
		return
	}
	if f := v.FieldByName("metallicFactor"); f.IsValid() && f.Kind() == reflect.Float32 {
		metal = float32(f.Float())
	}
	if f := v.FieldByName("roughnessFactor"); f.IsValid() && f.Kind() == reflect.Float32 {
		rough = float32(f.Float())
	}
	if f := v.FieldByName("baseColorFactor"); f.IsValid() {
		if r := f.FieldByName("R"); r.IsValid() {
			base.R = float32(r.Float())
			base.G = float32(f.FieldByName("G").Float())
			base.B = float32(f.FieldByName("B").Float())
			base.A = float32(f.FieldByName("A").Float())
		}
	}
	if f := v.FieldByName("emissiveFactor"); f.IsValid() {
		if r := f.FieldByName("R"); r.IsValid() {
			emit.R = float32(r.Float())
			emit.G = float32(f.FieldByName("G").Float())
			emit.B = float32(f.FieldByName("B").Float())
		}
	}
	return
}

func unwrapStandard(im material.IMaterial) *material.Standard {
	switch m := im.(type) {
	case *litMat:
		return m.Standard
	case *material.Standard:
		return m
	}
	return nil
}

func (w *World) wrapMeshMaterials(mesh *graphic.Mesh, convertStandard bool) *pbrMat {
	if mesh == nil {
		return nil
	}
	mats := mesh.Materials()
	if len(mats) == 0 {
		return nil
	}
	wrapped := make([]material.IMaterial, len(mats))
	var first *pbrMat
	changed := false
	for i, gm := range mats {
		im := gm.IMaterial()
		switch m := im.(type) {
		case *pbrMat:
			m.w = w
			if m.Physical != nil {
				m.Physical.SetShader("mbphysical")
			}
			wrapped[i] = m
			if first == nil {
				first = m
			}
		case *material.Physical:
			wm := w.wrapPhysical(m)
			wrapped[i] = wm
			changed = true
			if first == nil {
				first = wm
			}
		default:
			if convertStandard {
				if std := unwrapStandard(im); std != nil {
					wm := w.newPBR()
					c := std.AmbientColor()
					wm.albedo = math32.Color4{c.R, c.G, c.B, 1}
					wm.applyFactors()
					wrapped[i] = wm
					changed = true
					if first == nil {
						first = wm
					}
					continue
				}
			}
			wrapped[i] = im
		}
	}
	if !changed {
		return first
	}
	geom := mesh.GetGeometry()
	groups := 0
	if geom != nil {
		groups = geom.GroupCount()
	}
	mesh.ClearMaterials()
	if groups == len(wrapped) && groups > 0 {
		for i, im := range wrapped {
			mesh.AddGroupMaterial(im, i)
		}
	} else if len(wrapped) > 0 {
		mesh.SetMaterial(wrapped[0])
	}
	return first
}

func (w *World) walkMeshes(n core.INode, fn func(*graphic.Mesh)) {
	if n == nil {
		return
	}
	if mesh, ok := n.(*graphic.Mesh); ok {
		fn(mesh)
	}
	node := n.GetNode()
	if node == nil {
		return
	}
	for _, ch := range node.Children() {
		w.walkMeshes(ch, fn)
	}
}

func (w *World) attachLoadedPBR(e *Entity) {
	if e == nil || e.node == nil {
		return
	}
	var first *pbrMat
	var firstMesh *graphic.Mesh
	w.walkMeshes(e.node, func(mesh *graphic.Mesh) {
		if pm := w.wrapMeshMaterials(mesh, false); pm != nil && first == nil {
			first = pm
			firstMesh = mesh
		}
	})
	if first == nil {
		return
	}
	e.pbr = first.Physical
	e.pbrWrap = first
	e.usePBR = true
	if e.mesh == nil {
		e.mesh = firstMesh
	}
}

func (w *World) convertEntityPBR(e *Entity) *pbrMat {
	if e == nil {
		return nil
	}
	if e.usePBR && e.pbrWrap != nil {
		return e.pbrWrap
	}
	var first *pbrMat
	if e.node != nil {
		w.walkMeshes(e.node, func(mesh *graphic.Mesh) {
			if pm := w.wrapMeshMaterials(mesh, true); pm != nil && first == nil {
				first = pm
				if e.mesh == nil {
					e.mesh = mesh
				}
			}
		})
	}
	if first == nil && e.mesh != nil {
		first = w.newPBR()
		if e.mat != nil {
			c := e.mat.AmbientColor()
			a := e.tint.A
			if a <= 0 {
				a = 1
			}
			first.albedo = math32.Color4{c.R, c.G, c.B, a}
			first.applyFactors()
		} else if e.tint.A > 0 || e.tint.R+e.tint.G+e.tint.B > 0 {
			first.albedo = e.tint
			if first.albedo.A <= 0 {
				first.albedo.A = 1
			}
			first.applyFactors()
		}
		e.mesh.SetMaterial(first)
	}
	if first == nil {
		return nil
	}
	e.pbr = first.Physical
	e.pbrWrap = first
	e.usePBR = true
	return first
}

func (w *World) convertEntityPhong(e *Entity) {
	if e == nil {
		return
	}
	if e.mat == nil {
		e.mat = w.newMat()
		if e.tint.A > 0 || e.tint.R+e.tint.G+e.tint.B > 0 {
			e.mat.SetColor(&math32.Color{e.tint.R, e.tint.G, e.tint.B})
			e.mat.SetOpacity(e.tint.A)
		}
	} else {
		e.mat.SetShader(w.litShaderName())
	}
	if e.mesh != nil {
		e.mesh.SetMaterial(newLitMat(w, e.mat))
	} else if e.node != nil {
		w.walkMeshes(e.node, func(mesh *graphic.Mesh) {
			mesh.SetMaterial(newLitMat(w, e.mat))
		})
	}
	e.usePBR = false
	e.pbr = nil
	e.pbrWrap = nil
}

func (w *World) createPBRMaterial() int {
	if w.pbrLib == nil {
		w.pbrLib = map[int]*pbrMat{}
	}
	if w.nextPBR < 1 {
		w.nextPBR = 1
	}
	id := w.nextPBR
	w.nextPBR++
	w.pbrLib[id] = w.newPBR()
	return id
}

func (w *World) applyPBRMaterial(e *Entity, matID int) bool {
	pm := w.pbrLib[matID]
	if e == nil || pm == nil {
		return false
	}
	if e.mesh != nil {
		e.mesh.SetMaterial(pm)
	} else if e.node != nil {
		w.walkMeshes(e.node, func(mesh *graphic.Mesh) {
			mesh.SetMaterial(pm)
		})
	} else {
		return false
	}
	e.pbr = pm.Physical
	e.pbrWrap = pm
	e.usePBR = true
	e.tint = pm.albedo
	return true
}

func (w *World) eachEntityPBR(e *Entity, fn func(*pbrMat)) {
	if e == nil || fn == nil {
		return
	}
	seen := map[*pbrMat]bool{}
	if e.node != nil {
		w.walkMeshes(e.node, func(mesh *graphic.Mesh) {
			for _, gm := range mesh.Materials() {
				if pm, ok := gm.IMaterial().(*pbrMat); ok && !seen[pm] {
					seen[pm] = true
					fn(pm)
				}
			}
		})
	}
	if e.pbrWrap != nil && !seen[e.pbrWrap] {
		fn(e.pbrWrap)
	}
}

func (w *World) pbrOf(id int, create bool) (*pbrMat, *Entity, error) {
	if m := w.pbrLib[id]; m != nil {
		if e := w.ents[id]; e != nil && e.usePBR && e.pbrWrap != nil {
			return e.pbrWrap, e, nil
		}
		return m, nil, nil
	}
	e, err := w.ent(id)
	if err != nil {
		return nil, nil, err
	}
	if e.pbrWrap != nil {
		return e.pbrWrap, e, nil
	}
	if !create {
		return nil, e, nil
	}
	pm := w.convertEntityPBR(e)
	if pm == nil {
		return nil, e, fmt.Errorf("PBR: entity %d has no mesh", id)
	}
	return pm, e, nil
}

func (w *World) retargetPBRShaders(n core.INode) {
	if n == nil {
		return
	}
	if mesh, ok := n.(*graphic.Mesh); ok {
		w.wrapMeshMaterials(mesh, false)
		for _, gm := range mesh.Materials() {
			switch m := gm.IMaterial().(type) {
			case *pbrMat:
				m.w = w
				if m.Physical != nil {
					m.Physical.SetShader("mbphysical")
				}
			case *material.Physical:
				m.SetShader("mbphysical")
			}
		}
	}
	node := n.GetNode()
	if node == nil {
		return
	}
	for _, ch := range node.Children() {
		w.retargetPBRShaders(ch)
	}
}

func (w *World) bindPBRUniforms(gs *gls.GLS, m *pbrMat) {
	if gs == nil {
		return
	}
	use := 0
	if w.iblOn {
		use = 1
	}
	if m != nil {
		if m.ibl == 0 {
			use = 0
		} else if m.ibl == 1 {
			use = 1
		}
	}
	setUni1i(gs, "UseIBL", use)
	inten := w.iblIntensity
	if inten <= 0 {
		inten = 1
	}
	setUni1f(gs, "IBLIntensity", inten)
	sky := math32.Color{0.42, 0.52, 0.72}
	ground := math32.Color{0.16, 0.12, 0.09}
	if w.skyVisible() {
		sky = math32.Color{0.55, 0.62, 0.82}
	}
	setUni3f(gs, "IBLSky", sky.R, sky.G, sky.B)
	setUni3f(gs, "IBLGround", ground.R, ground.G, ground.B)
	w.bindEnvCube(gs)
	ao := float32(1)
	if m != nil {
		ao = m.ao
	}
	if ao < 0 {
		ao = 0
	}
	setUni1f(gs, "uAO", ao)
	setUni1f(gs, "uOcclusionStrength", 1)
}

func (w *World) setPBREnvMap(m *pbrMat, tex *texture.Texture2D) {
	if m == nil {
		return
	}
	if m.envTex != nil {
		m.RemoveTexture(m.envTex)
		m.ShaderDefines.Unset("HAS_ENVMAP")
		m.envTex = nil
	}
	if tex == nil {
		return
	}
	tex.SetUniformNames("uEnvSampler", "uEnvTexParams")
	m.ShaderDefines.Set("HAS_ENVMAP", "")
	m.AddTexture(tex)
	m.envTex = tex
}

func (w *World) texByID(id int) *texture.Texture2D {
	if t := w.texs[id]; t != nil {
		return t.tex
	}
	return nil
}
