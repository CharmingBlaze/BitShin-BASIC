package runtime

import (
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/math32"
	"github.com/go-gl/gl/v3.3-core/gl"
)

func (w *World) drawModernPass(cam *camera.Camera) {
	if w.app == nil || cam == nil {
		return
	}
	w.detectModernGL()
	gs := w.app.Gls()
	restore := saveModernGL(gs)
	defer restore()
	var view, proj, vp math32.Matrix4
	cam.ViewMatrix(&view)
	cam.ProjMatrix(&proj)
	vp.MultiplyMatrices(&proj, &view)
	w.drawGPUInstances(&vp, false)
	w.drawGeomPoints(cam, &vp)
}

func (w *World) drawGPUInstanceDepth(lightVP *math32.Matrix4) int {
	if lightVP == nil || !w.ensureInstDepthProgram() {
		return 0
	}
	return w.drawGPUInstances(lightVP, true)
}

func (w *World) drawGPUInstances(viewProj *math32.Matrix4, depth bool) int {
	m := w.ensureGLMod()
	if !m.gpuInst || !m.caps.instance {
		return 0
	}
	prog := m.instProg
	if depth {
		if !m.instDepthOK {
			return 0
		}
		prog = m.instDepth
	} else if !w.ensureInstProgram() {
		return 0
	}
	drew := 0
	gl.UseProgram(prog)
	loc := gl.GetUniformLocation(prog, gl.Str("ViewProj\x00"))
	if loc >= 0 {
		gl.UniformMatrix4fv(loc, 1, false, &viewProj[0])
	}
	if depth {
		mm := 0
		if w.shadow.filter == shadowFilterEVSM || w.shadow.filter == shadowFilterMSM {
			mm = w.shadow.filter
		}
		if ml := gl.GetUniformLocation(prog, gl.Str("MomentMode\x00")); ml >= 0 {
			gl.Uniform1i(ml, int32(mm))
		}
		c := w.shadow.evsmC
		if c <= 0 {
			c = 40
		}
		if cl := gl.GetUniformLocation(prog, gl.Str("EvsmC\x00")); cl >= 0 {
			gl.Uniform1f(cl, c)
		}
	}
	if !depth {
		cl := gl.GetUniformLocation(prog, gl.Str("Color\x00"))
		if cl >= 0 {
			gl.Uniform3f(cl, 0.35, 0.62, 0.28)
		}
	}
	for _, im := range w.instances {
		if im == nil || im.count < 1 || !im.gpuOK {
			continue
		}
		if im.vao == 0 {
			continue
		}
		gl.BindVertexArray(im.vao)
		if im.indexN > 0 {
			gl.DrawElementsInstanced(gl.TRIANGLES, im.indexN, gl.UNSIGNED_INT, nil, int32(im.count))
		} else if im.vertN > 0 {
			gl.DrawArraysInstanced(gl.TRIANGLES, 0, im.vertN, int32(im.count))
		}
		drew++
	}
	gl.BindVertexArray(0)
	return drew
}

func (w *World) uploadInstanceGPU(im *instancedMesh) {
	if im == nil {
		return
	}
	m := w.ensureGLMod()
	if !m.gpuInst || !m.caps.instance {
		im.gpuOK = false
		return
	}
	if len(im.tris) == 0 {
		im.gpuOK = false
		return
	}
	pos := make([]float32, 0, len(im.tris)*9)
	nor := make([]float32, 0, len(im.tris)*9)
	idx := make([]uint32, 0, len(im.tris)*3)
	var vi uint32
	for _, t := range im.tris {
		pos = append(pos, t[0].X, t[0].Y, t[0].Z, t[1].X, t[1].Y, t[1].Z, t[2].X, t[2].Y, t[2].Z)
		e1 := t[1]
		e1.Sub(&t[0])
		e2 := t[2]
		e2.Sub(&t[0])
		n := e1.Cross(&e2)
		if n.LengthSq() > 1e-10 {
			n.Normalize()
		} else {
			n.Set(0, 1, 0)
		}
		nor = append(nor, n.X, n.Y, n.Z, n.X, n.Y, n.Z, n.X, n.Y, n.Z)
		idx = append(idx, vi, vi+1, vi+2)
		vi += 3
	}
	mats := make([]float32, 0, im.count*16)
	for i := 0; i < im.count && i < len(im.xforms); i++ {
		mx := instanceMatrix(im.xforms[i])
		mats = append(mats, mx[:]...)
	}
	if im.vao == 0 {
		gl.GenVertexArrays(1, &im.vao)
		gl.GenBuffers(1, &im.vbo)
		gl.GenBuffers(1, &im.ebo)
		gl.GenBuffers(1, &im.instVBO)
	}
	gl.BindVertexArray(im.vao)
	stride := int32(6 * 4)
	inter := make([]float32, len(im.tris)*6*3)
	for i := 0; i < len(im.tris)*3; i++ {
		inter[i*6+0] = pos[i*3+0]
		inter[i*6+1] = pos[i*3+1]
		inter[i*6+2] = pos[i*3+2]
		inter[i*6+3] = nor[i*3+0]
		inter[i*6+4] = nor[i*3+1]
		inter[i*6+5] = nor[i*3+2]
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, im.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(inter)*4, gl.Ptr(inter), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, nil)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 3, gl.FLOAT, false, stride, 12)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, im.ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(idx)*4, gl.Ptr(idx), gl.STATIC_DRAW)
	gl.BindBuffer(gl.ARRAY_BUFFER, im.instVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(mats)*4, gl.Ptr(mats), gl.DYNAMIC_DRAW)
	for c := uint32(0); c < 4; c++ {
		gl.EnableVertexAttribArray(2 + c)
		gl.VertexAttribPointerWithOffset(2+c, 4, gl.FLOAT, false, 64, uintptr(c*16))
		gl.VertexAttribDivisor(2+c, 1)
	}
	gl.BindVertexArray(0)
	im.indexN = int32(len(idx))
	im.vertN = int32(len(im.tris) * 3)
	im.gpuOK = im.vao != 0
}

func (w *World) drawGeomPoints(cam *camera.Camera, viewProj *math32.Matrix4) {
	m := w.ensureGLMod()
	if len(m.geoms) == 0 {
		return
	}
	if !w.ensureGeomProgram() {
		return
	}
	var view math32.Matrix4
	cam.ViewMatrix(&view)
	// camera right/up from view matrix
	right := math32.Vector3{view[0], view[4], view[8]}
	up := math32.Vector3{view[1], view[5], view[9]}
	right.Normalize()
	up.Normalize()
	gl.UseProgram(m.geomProg)
	if loc := gl.GetUniformLocation(m.geomProg, gl.Str("ViewProj\x00")); loc >= 0 {
		gl.UniformMatrix4fv(loc, 1, false, &viewProj[0])
	}
	if loc := gl.GetUniformLocation(m.geomProg, gl.Str("CamRight\x00")); loc >= 0 {
		gl.Uniform3f(loc, right.X, right.Y, right.Z)
	}
	if loc := gl.GetUniformLocation(m.geomProg, gl.Str("CamUp\x00")); loc >= 0 {
		gl.Uniform3f(loc, up.X, up.Y, up.Z)
	}
	for _, g := range m.geoms {
		if g == nil || g.count < 1 {
			continue
		}
		w.uploadGeomPoints(g)
		if g.vao == 0 {
			continue
		}
		gl.BindVertexArray(g.vao)
		gl.DrawArrays(gl.POINTS, 0, int32(g.count))
	}
	gl.BindVertexArray(0)
}

func (w *World) uploadGeomPoints(g *geomPoints) {
	if g == nil || !g.dirty && g.vao != 0 {
		return
	}
	if g.vao == 0 {
		gl.GenVertexArrays(1, &g.vao)
		gl.GenBuffers(1, &g.vbo)
	}
	gl.BindVertexArray(g.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, g.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(g.data)*4, gl.Ptr(g.data), gl.DYNAMIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 4, gl.FLOAT, false, 7*4, nil)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 3, gl.FLOAT, false, 7*4, 16)
	gl.BindVertexArray(0)
	g.dirty = false
}

func (w *World) createGeomPoints(count int) int {
	m := w.ensureGLMod()
	w.detectModernGL()
	if !m.caps.geom {
		w.skipGL("CreateGeomPoints", "OpenGL 3.2 geometry shaders")
		return 0
	}
	if count < 1 {
		count = 1
	}
	if count > 4096 {
		count = 4096
	}
	g := &geomPoints{count: count, data: make([]float32, count*7), dirty: true}
	for i := 0; i < count; i++ {
		g.data[i*7+3] = 0.25
		g.data[i*7+4] = 1
		g.data[i*7+5] = 1
		g.data[i*7+6] = 1
	}
	id := w.takeHandle(&m.freeGeom, &m.nextGeom)
	m.geoms[id] = g
	return id
}
