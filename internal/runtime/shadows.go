package runtime

import (
	"fmt"
	"math"
	"strings"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/go-gl/gl/v3.3-core/gl"
)

const (
	maxShadowCascades = 4
	maxShadowPoints   = 2
	maxPointFaces     = 12
	maxShadowSpots    = 2
	maxShadowTiles    = 18

	shadowFilterPCF  = 0
	shadowFilterPCSS = 1
	shadowFilterEVSM = 2
	shadowFilterMSM  = 3

	// Let the user camera, scene transforms, and all CSM tiles settle before
	// exposing the atlas. This prevents a valid-but-transient startup cascade
	// from sweeping an oversized shadow across the first visible frames.
	shadowWarmupFrames = 6
)

type shaderUni struct {
	n int
	v [4]float32
}

type nodeTween struct {
	id                     int
	kind                   int
	x0, y0, z0, x1, y1, z1 float32
	t, dur                 float32
	hook                   string
}

// shadowMap is a depth atlas: directional CSM plus optional point/spot tiles.
type shadowMap struct {
	on, ready, glOK                                                           bool
	size                                                                      int
	extent                                                                    float32
	distance                                                                  float32
	zPad                                                                      float32
	lightID                                                                   int
	cascades                                                                  int
	filter                                                                    int // 0 PCF, 1 PCSS, 2 EVSM, 3 MSM
	pcf                                                                       int
	bias                                                                      float32
	lightSize                                                                 float32
	cache                                                                     bool
	atlas                                                                     bool
	contact                                                                   bool
	sss                                                                       int // 0 off, 1 on (8 taps), >1 high (16 taps)
	softness                                                                  float32
	color                                                                     math32.Color
	fadeNear                                                                  float32
	fadeFar                                                                   float32
	dirty                                                                     bool // MarkShadowDirty / deforming casters
	cacheKey                                                                  uint64
	staticKey                                                                 uint64
	staticReady                                                               bool
	fbo, tex                                                                  uint32
	dynTex, depthTex                                                          uint32
	dummyTex                                                                  uint32
	allocW, allocH                                                            int32
	mainT, staticT, dynT                                                      shadowTarget
	blurG3N                                                                   *gls.Program
	blitVAO, blitVBO                                                          uint32
	normalBias                                                                float32
	texelWorld                                                                [maxShadowCascades]float32
	evsmC                                                                     float32
	momentFallback                                                            bool
	atlasCols                                                                 int
	atlasRows                                                                 int
	depthG3N                                                                  *gls.Program
	depthMomentG3N                                                            *gls.Program
	depthAlphaG3N                                                             *gls.Program
	primed                                                                    map[*geometry.Geometry]bool
	okLogged                                                                  bool
	warm                                                                      int // consecutive good depth frames
	lastCam                                                                   math32.Vector3
	camTrack                                                                  bool
	tmpMVP, tmpView, tmpProj                                                  math32.Matrix4
	tmpCamPos, tmpCamDir, tmpCenter, tmpEye, tmpDir, tmpOff, tmpUp, tmpTarget math32.Vector3
	tmpOrtho, tmpPersp                                                        *camera.Camera
	lightVP                                                                   [maxShadowCascades]math32.Matrix4
	splits                                                                    [maxShadowCascades]float32
	pointN                                                                    int
	spotN                                                                     int
	pointPos                                                                  [maxShadowPoints]math32.Vector3
	pointRange                                                                [maxShadowPoints]float32
	pointTile                                                                 [maxShadowPoints]int
	pointVP                                                                   [maxPointFaces]math32.Matrix4
	spotPos                                                                   [maxShadowSpots]math32.Vector3
	spotDir                                                                   [maxShadowSpots]math32.Vector3
	spotRange                                                                 [maxShadowSpots]float32
	spotCos                                                                   [maxShadowSpots]float32
	spotTile                                                                  [maxShadowSpots]int
	spotVP                                                                    [maxShadowSpots]math32.Matrix4
}

type litMat struct {
	*material.Standard
	w          *World
	recvShadow bool
}

func (m *litMat) GetMaterial() *material.Material { return m.Standard.GetMaterial() }

func newLitMat(w *World, mat *material.Standard) *litMat {
	return &litMat{Standard: mat, w: w, recvShadow: true}
}

func (m *litMat) RenderSetup(gs *gls.GLS) {
	m.Standard.RenderSetup(gs)
	if m.w != nil {
		m.w.applyUserUniforms(gs)
		m.w.bindShadowUniforms(gs)
		m.w.bindFogUniforms(gs)
		recv := 1
		if !m.recvShadow {
			recv = 0
		}
		setUni1i(gs, "MeshReceiveShadow", recv)
	}
}

func (m *litMat) Dispose() { m.Standard.Dispose() }

func registerShadowShaders(r *renderer.Renderer) {
	r.AddShader("mbshadow_vertex", mbshadowVertex)
	r.AddShader("mbshadow_fragment", mbshadowFragment)
	r.AddProgram("bsshadow", "mbshadow_vertex", "mbshadow_fragment")
	r.AddShader("mbpart_vertex", mbpartVertex)
	r.AddShader("mbpart_fragment", mbpartFragment)
	r.AddProgram("mbpart", "mbpart_vertex", "mbpart_fragment")
	r.AddShader("mbflake_fragment", mbflakeFragment)
	r.AddProgram("mbflake", "mbpart_vertex", "mbflake_fragment")
	r.AddShader("mbphysical_vertex", mbphysicalVertex)
	r.AddShader("mbphysical_fragment", mbphysicalFragment)
	r.AddProgram("mbphysical", "mbphysical_vertex", "mbphysical_fragment")
}

func (w *World) applyUserUniforms(gs *gls.GLS) {
	if gs == nil {
		return
	}
	for name, u := range w.shaderUnis {
		uni := gls.Uniform{}
		uni.Init(name)
		loc := uni.Location(gs)
		if loc < 0 {
			continue
		}
		switch u.n {
		case 1:
			if name == "ProbeEnabled" || name == "FogMode" || name == "ShadowEnabled" || name == "UseIBL" {
				gs.Uniform1i(loc, int32(u.v[0]))
			} else {
				gs.Uniform1f(loc, u.v[0])
			}
		case 2:
			gs.Uniform2f(loc, u.v[0], u.v[1])
		case 3:
			gs.Uniform3f(loc, u.v[0], u.v[1], u.v[2])
		default:
			gs.Uniform4f(loc, u.v[0], u.v[1], u.v[2], u.v[3])
		}
	}
}

func (w *World) bindShadowUniforms(gs *gls.GLS) {
	if gs == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("EnableShadows: uniform panic", r)
		}
	}()
	if !w.shadowSampleOK() {
		dummy := w.ensureDummyShadowTex()
		w.bindShadowSampler(gs, dummy, 8, "ShadowMap")
		w.bindShadowSampler(gs, dummy, 9, "ShadowMapDyn")
		setUni1i(gs, "ShadowEnabled", 0)
		setUni1i(gs, "MeshReceiveShadow", 0)
		setUni1i(gs, "ShadowCacheSplit", 0)
		setUni1i(gs, "ShadowContact", 0)
		setUni1i(gs, "ShadowSSS", 0)
		return
	}
	w.bindShadowSampler(gs, w.shadow.tex, 8, "ShadowMap")
	split := 0
	dyn := w.shadow.dynTex
	if w.shadow.cache && dyn != 0 && dyn != w.ensureDummyShadowTex() {
		w.bindShadowSampler(gs, dyn, 9, "ShadowMapDyn")
		split = 1
	} else {
		w.bindShadowSampler(gs, w.ensureDummyShadowTex(), 9, "ShadowMapDyn")
	}
	setUni1i(gs, "ShadowCacheSplit", split)
	setUni1i(gs, "ShadowEnabled", 1)
	setUni1i(gs, "MeshReceiveShadow", 1)
	setUni1i(gs, "ShadowCascades", w.shadow.cascades)
	setUni1i(gs, "ShadowFilter", w.shadow.filter)
	setUni1i(gs, "ShadowPCF", w.shadow.pcf)
	setUni1f(gs, "ShadowBias", w.shadow.bias)
	setUni1f(gs, "ShadowNormalBias", w.shadow.normalBias)
	soft := w.shadow.softness
	if soft <= 0.001 {
		soft = 1.0
	}
	setUni1f(gs, "ShadowSoftness", soft)
	setUni3f(gs, "ShadowColor", w.shadow.color.R, w.shadow.color.G, w.shadow.color.B)
	setUni1f(gs, "ShadowFadeNear", w.shadow.fadeNear)
	setUni1f(gs, "ShadowFadeFar", w.shadow.fadeFar)
	sdir := w.shadowLightDir()
	setUni3f(gs, "ShadowSunDir", sdir.X, sdir.Y, sdir.Z)
	setUni4f(gs, "ShadowTexelWorld", w.shadow.texelWorld[0], w.shadow.texelWorld[1], w.shadow.texelWorld[2], w.shadow.texelWorld[3])
	c := w.shadow.evsmC
	if c <= 0 || c > 14 {
		c = 8
	}
	setUni1f(gs, "EvsmC", c)
	setUni1f(gs, "ShadowLightSize", w.shadow.lightSize)
	setUni1i(gs, "ShadowContact", bool01(w.shadow.contact))
	setUni1i(gs, "ShadowSSS", w.shadow.sss)
	cols, rows := w.shadow.atlasCols, w.shadow.atlasRows
	if cols < 1 {
		cols = w.shadow.cascades
		if cols < 1 {
			cols = 1
		}
	}
	if rows < 1 {
		rows = 1
	}
	setUni1i(gs, "AtlasCols", cols)
	setUni1i(gs, "AtlasRows", rows)
	setUni4f(gs, "ShadowSplit", w.shadow.splits[0], w.shadow.splits[1], w.shadow.splits[2], w.shadow.splits[3])
	setUniMat4s(gs, "LightVP", w.shadow.lightVP[:])
	setUni1i(gs, "ShadowPoints", w.shadow.pointN)
	for i := 0; i < maxShadowPoints; i++ {
		p := w.shadow.pointPos[i]
		setUni3f(gs, fmt.Sprintf("PointPos[%d]", i), p.X, p.Y, p.Z)
		setUni1f(gs, fmt.Sprintf("PointRange[%d]", i), w.shadow.pointRange[i])
		setUni1i(gs, fmt.Sprintf("PointTile[%d]", i), w.shadow.pointTile[i])
	}
	setUniMat4s(gs, "PointVP", w.shadow.pointVP[:])
	setUni1i(gs, "ShadowSpots", w.shadow.spotN)
	for i := 0; i < maxShadowSpots; i++ {
		p := w.shadow.spotPos[i]
		d := w.shadow.spotDir[i]
		setUni3f(gs, fmt.Sprintf("SpotPos[%d]", i), p.X, p.Y, p.Z)
		setUni3f(gs, fmt.Sprintf("SpotDir[%d]", i), d.X, d.Y, d.Z)
		setUni1f(gs, fmt.Sprintf("SpotRange[%d]", i), w.shadow.spotRange[i])
		setUni1f(gs, fmt.Sprintf("SpotCos[%d]", i), w.shadow.spotCos[i])
		setUni1i(gs, fmt.Sprintf("SpotTile[%d]", i), w.shadow.spotTile[i])
	}
	setUniMat4s(gs, "SpotVP", w.shadow.spotVP[:])
}

func (w *World) bindShadowSampler(gs *gls.GLS, tex uint32, unit uint32, name string) {
	if gs == nil || tex == 0 {
		return
	}
	gs.ActiveTexture(gls.TEXTURE0 + unit)
	gs.BindTexture(gls.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_COMPARE_MODE, gl.NONE)
	setUni1i(gs, name, int(unit))
	gs.ActiveTexture(gls.TEXTURE0)
}

// ensureDummyShadowTex creates a 1x1 white (depth=1) sampler so unused
// texture(ShadowMap) in disabled branches cannot hit unbound unit 0.
func (w *World) ensureDummyShadowTex() uint32 {
	if w.shadow.dummyTex != 0 {
		return w.shadow.dummyTex
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("EnableShadows: dummy map", r)
		}
	}()
	var tex uint32
	gl.GenTextures(1, &tex)
	if tex == 0 {
		return 0
	}
	gl.BindTexture(gl.TEXTURE_2D, tex)
	white := []uint8{255, 255, 255, 255}
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 1, 1, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(white))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	w.shadow.dummyTex = tex
	return tex
}

func bool01(v bool) int {
	if v {
		return 1
	}
	return 0
}

func setUni1i(gs *gls.GLS, name string, v int) {
	u := gls.Uniform{}
	u.Init(name)
	if loc := u.Location(gs); loc >= 0 {
		gs.Uniform1i(loc, int32(v))
	}
}

func setUni1f(gs *gls.GLS, name string, v float32) {
	u := gls.Uniform{}
	u.Init(name)
	if loc := u.Location(gs); loc >= 0 {
		gs.Uniform1f(loc, v)
	}
}

func setUni3f(gs *gls.GLS, name string, x, y, z float32) {
	u := gls.Uniform{}
	u.Init(name)
	if loc := u.Location(gs); loc >= 0 {
		gs.Uniform3f(loc, x, y, z)
	}
}

func setUniMat4(gs *gls.GLS, name string, m *math32.Matrix4) {
	u := gls.Uniform{}
	u.Init(name)
	if loc := u.Location(gs); loc >= 0 {
		gs.UniformMatrix4fv(loc, 1, false, &m[0])
	}
}

func setUniMat4s(gs *gls.GLS, name string, mats []math32.Matrix4) {
	if gs == nil || len(mats) == 0 {
		return
	}
	u := gls.Uniform{}
	u.Init(name + "[0]")
	loc := u.Location(gs)
	if loc < 0 {
		u.Init(name)
		loc = u.Location(gs)
	}
	if loc >= 0 {
		gs.UniformMatrix4fv(loc, int32(len(mats)), false, &mats[0][0])
	}
}

func (w *World) shadowSampleOK() bool {
	if w == nil || !w.shadow.on || !w.shadow.ready || w.shadow.tex == 0 || !w.hasShadowLight() {
		return false
	}
	// Flip 1 presents setup without the user loop (no CameraFollow yet).
	if w.loopFrames < 2 {
		return false
	}
	if w.shadow.warm < shadowWarmupFrames {
		return false
	}
	if !w.cascadeReady() {
		return false
	}
	return true
}

func (w *World) cascadeReady() bool {
	tw := w.shadow.texelWorld[0]
	if tw < 1e-6 || tw > 2 {
		return false
	}
	m := w.shadow.lightVP[0]
	if m[0] == 0 && m[5] == 0 && m[10] == 0 {
		return false
	}
	return true
}

func (w *World) renderShadows(rend *renderer.Renderer, cam *camera.Camera) {
	if !w.shadow.on || w.app == nil || cam == nil || !w.hasShadowLight() {
		return
	}
	if w.loopFrames < 2 {
		w.shadow.ready = false
		w.shadow.warm = 0
		return
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("EnableShadows: panic", r)
			gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		}
	}()
	if err := w.ensureShadowGL(); err != nil {
		fmt.Println("EnableShadows:", err)
		return
	}
	if w.shadow.depthG3N == nil || w.shadow.depthG3N.Handle() == 0 {
		fmt.Println("EnableShadows: depth program not linked")
		return
	}
	n := w.shadow.cascades
	if n < 1 {
		n = 4
	}
	if n > maxShadowCascades {
		n = maxShadowCascades
	}
	w.shadow.cascades = n
	if w.shadow.size < 256 {
		w.shadow.size = 256
	}
	if w.shadow.pcf < 3 {
		w.shadow.pcf = 3
	}
	if w.shadow.bias <= 0 {
		w.shadow.bias = 0.0025
	}
	if w.shadow.lightSize <= 0 {
		w.shadow.lightSize = 0.04
	}
	if w.scene != nil {
		w.scene.UpdateMatrixWorld()
	}
	w.computeCascadeSplits(cam, n)
	w.buildCascadeViews(cam, n)
	if w.shadow.camTrack && w.shadow.tmpCamPos.DistanceTo(&w.shadow.lastCam) > 6 {
		w.shadow.warm = 0
		w.shadow.ready = false
	}
	w.shadow.lastCam = w.shadow.tmpCamPos
	w.shadow.camTrack = true
	if !w.cascadeReady() {
		w.shadow.warm = 0
		w.shadow.ready = false
		return
	}
	tiles := w.collectLocalLightViews(n)
	cols, rows := atlasGrid(tiles, !w.shadow.atlas && w.shadow.pointN == 0 && w.shadow.spotN == 0)
	w.shadow.atlasCols, w.shadow.atlasRows = cols, rows
	if err := w.ensureShadowFBO(cols, rows); err != nil {
		fmt.Println("EnableShadows:", err)
		return
	}
	w.markShadowPrimed()
	if len(w.shadow.primed) == 0 {
		return
	}
	gs := w.app.Gls()
	if gs == nil {
		return
	}
	restore := saveShadowGL(gs)
	defer restore()
	depthProg := w.shadowDepthProgram()
	gs.UseProgram(depthProg)
	mvpLoc := depthProg.GetUniformLocation("MVP")
	if mvpLoc < 0 {
		fmt.Println("EnableShadows: MVP uniform missing")
		return
	}
	w.setDepthMomentUniforms(gs)
	gs.Enable(gls.DEPTH_TEST)
	gs.DepthMask(true)
	gs.Enable(gls.CULL_FACE)
	// Render the light-facing surface into the depth map. Front-face culling
	// records the exit face of thick primitives (notably the large platform
	// cube), which can make the whole receiver look like one giant shadow.
	// Normal bias and polygon offset already handle surface acne.
	gs.CullFace(gls.BACK)
	gs.Disable(gls.SCISSOR_TEST)
	gl.Disable(gl.SCISSOR_TEST)
	gl.DepthMask(true)
	gl.ClearDepth(1)
	gs.Enable(gls.POLYGON_OFFSET_FILL)
	gs.PolygonOffset(1.1, 4)

	drew := w.renderShadowLayer(gs, &w.shadow.mainT, mvpLoc, n, shadowLayerAll)
	w.shadow.tex = w.shadow.mainT.lightTex()
	w.shadow.dynTex = 0
	w.shadow.depthTex = w.shadow.mainT.depth
	w.shadow.fbo = w.shadow.mainT.fbo
	w.shadow.allocW, w.shadow.allocH = w.shadow.mainT.w, w.shadow.mainT.h
	w.shadow.staticReady = false
	if drew > 0 && w.cascadeReady() {
		w.shadow.ready = true
		if w.shadow.warm < shadowWarmupFrames {
			w.shadow.warm++
		}
		w.shadow.cacheKey = w.shadowSceneKey()
		w.clearShadowDirty()
		if !w.shadow.okLogged {
			fmt.Println("shadows: ok")
			w.shadow.okLogged = true
		}
		return
	}
	w.shadow.ready = false
	w.shadow.warm = 0
	_ = rend
}

func (w *World) shadowDepthProgram() *gls.Program {
	if w.shadow.filter == shadowFilterEVSM || w.shadow.filter == shadowFilterMSM {
		if w.shadow.depthMomentG3N != nil && w.shadow.depthMomentG3N.Handle() != 0 {
			return w.shadow.depthMomentG3N
		}
	}
	return w.shadow.depthG3N
}

func (w *World) setDepthMomentUniforms(gs *gls.GLS) {
	prog := w.shadowDepthProgram()
	if gs == nil || prog == nil {
		return
	}
	mode := 0
	if w.shadow.filter == shadowFilterEVSM || w.shadow.filter == shadowFilterMSM {
		mode = w.shadow.filter
	}
	if loc := prog.GetUniformLocation("MomentMode"); loc >= 0 {
		gs.Uniform1i(loc, int32(mode))
	}
	c := w.shadow.evsmC
	if c <= 0 {
		c = 40
	}
	if loc := prog.GetUniformLocation("EvsmC"); loc >= 0 {
		gs.Uniform1f(loc, c)
	}
}

func (w *World) renderShadowLayer(gs *gls.GLS, tgt *shadowTarget, mvpLoc int32, n, layer int) int {
	if gs == nil || tgt == nil || tgt.fbo == 0 {
		return 0
	}
	gs.UseProgram(w.shadowDepthProgram())
	w.setDepthMomentUniforms(gs)
	w.bindShadowTarget(gs, tgt)
	drew := 0
	for i := 0; i < n; i++ {
		w.shadowTileViewport(gs, i)
		drew += w.drawShadowCasters(gs, &w.shadow.lightVP[i], mvpLoc, layer)
	}
	for i := 0; i < w.shadow.pointN; i++ {
		base := w.shadow.pointTile[i]
		for f := 0; f < 6; f++ {
			w.shadowTileViewport(gs, base+f)
			drew += w.drawShadowCasters(gs, &w.shadow.pointVP[i*6+f], mvpLoc, layer)
		}
	}
	for i := 0; i < w.shadow.spotN; i++ {
		w.shadowTileViewport(gs, w.shadow.spotTile[i])
		drew += w.drawShadowCasters(gs, &w.shadow.spotVP[i], mvpLoc, layer)
	}
	if tgt.color != 0 {
		w.blurMoments(gs, tgt)
		gs.UseProgram(w.shadowDepthProgram())
	}
	return drew
}

func saveShadowGL(gs *gls.GLS) func() {
	var vp [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &vp[0])
	var fbo int32
	gl.GetIntegerv(gl.FRAMEBUFFER_BINDING, &fbo)
	return func() {
		if gs != nil {
			gs.Disable(gls.POLYGON_OFFSET_FILL)
			gs.Enable(gls.CULL_FACE)
			gs.CullFace(gls.BACK)
			gs.Enable(gls.DEPTH_TEST)
			gs.DepthMask(true)
			gs.Viewport(vp[0], vp[1], vp[2], vp[3])
		}
		gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(fbo))
		gl.DepthMask(true)
		gl.Disable(gl.SCISSOR_TEST)
	}
}

func (w *World) shadowTileViewport(gs *gls.GLS, tile int) {
	cols := w.shadow.atlasCols
	if cols < 1 {
		cols = 1
	}
	sz := int32(w.shadow.size)
	col := int32(tile % cols)
	row := int32(tile / cols)
	if gs != nil {
		gs.Viewport(col*sz, row*sz, sz, sz)
		return
	}
	gl.Viewport(col*sz, row*sz, sz, sz)
}

func atlasGrid(tiles int, strip bool) (cols, rows int) {
	if tiles < 1 {
		tiles = 1
	}
	if strip {
		return tiles, 1
	}
	cols = 1
	for cols*cols < tiles {
		cols++
	}
	rows = (tiles + cols - 1) / cols
	if rows < 1 {
		rows = 1
	}
	return cols, rows
}

func compileShadowProgram(gs *gls.GLS, vs, fs, label string) (*gls.Program, error) {
	prog := gs.NewProgram()
	prog.AddShader(gls.VERTEX_SHADER, vs)
	prog.AddShader(gls.FRAGMENT_SHADER, fs)
	if err := prog.Build(); err != nil {
		return nil, fmt.Errorf("%s shader:\n%s", label, err)
	}
	if prog.Handle() == 0 {
		return nil, fmt.Errorf("%s shader: link produced program 0", label)
	}
	return prog, nil
}

func (w *World) ensureShadowGL() error {
	if w.shadow.glOK && w.shadow.depthG3N != nil && w.shadow.depthG3N.Handle() != 0 {
		return nil
	}
	if w.app == nil {
		return fmt.Errorf("no GL context")
	}
	gs := w.app.Gls()
	if gs == nil {
		return fmt.Errorf("no GLS")
	}
	if err := gl.Init(); err != nil {
		return err
	}
	empty, err := compileShadowProgram(gs, depthVertexSrc, depthEmptyFragmentSrc, "depth")
	if err != nil {
		return err
	}
	moment, err := compileShadowProgram(gs, depthVertexSrc, depthFragmentSrc, "depth-moment")
	if err != nil {
		return err
	}
	alpha, err := compileShadowProgram(gs, depthAlphaVertexSrc, depthAlphaFragmentSrc, "depth-alpha")
	if err != nil {
		return err
	}
	w.shadow.depthG3N = empty
	w.shadow.depthMomentG3N = moment
	w.shadow.depthAlphaG3N = alpha
	w.shadow.glOK = true
	return nil
}

func (w *World) markShadowPrimed() {
	if w.shadow.primed == nil {
		w.shadow.primed = map[*geometry.Geometry]bool{}
	}
	for _, e := range w.ents {
		if e != nil && e.mesh != nil && e.name != "water" && !e.sky {
			w.shadow.primed[e.mesh.GetGeometry()] = true
		}
	}
}

func (w *World) ensureShadowFBO(cols, rows int) error {
	moment := 0
	if w.shadow.filter == shadowFilterEVSM || w.shadow.filter == shadowFilterMSM {
		moment = w.shadow.filter
	}
	if w.shadow.evsmC <= 0 {
		w.shadow.evsmC = 40
	}
	if w.shadow.cache {
		if err := w.ensureShadowTarget(&w.shadow.staticT, cols, rows, moment); err != nil {
			return err
		}
		if err := w.ensureShadowTarget(&w.shadow.dynT, cols, rows, moment); err != nil {
			return err
		}
		w.shadow.tex = w.shadow.staticT.lightTex()
		w.shadow.dynTex = w.shadow.dynT.lightTex()
		w.shadow.depthTex = w.shadow.staticT.depth
		w.shadow.fbo = w.shadow.staticT.fbo
		w.shadow.allocW, w.shadow.allocH = w.shadow.staticT.w, w.shadow.staticT.h
		return nil
	}
	if err := w.ensureShadowTarget(&w.shadow.mainT, cols, rows, moment); err != nil {
		return err
	}
	w.shadow.tex = w.shadow.mainT.lightTex()
	w.shadow.dynTex = 0
	w.shadow.depthTex = w.shadow.mainT.depth
	w.shadow.fbo = w.shadow.mainT.fbo
	w.shadow.allocW, w.shadow.allocH = w.shadow.mainT.w, w.shadow.mainT.h
	return nil
}

func snapShadow(v, step float32) float32 {
	if step <= 1e-5 {
		return v
	}
	return float32(math.Floor(float64(v/step)+0.5)) * step
}

func (w *World) hasShadowLight() bool {
	if w == nil {
		return false
	}
	if e := w.ents[w.shadow.lightID]; e != nil && e.lgtKind >= 1 && e.lgtKind <= 3 {
		return true
	}
	for _, e := range w.ents {
		if e == nil {
			continue
		}
		if e.lgtKind == 1 {
			return true
		}
		if e.castShadow && (e.lgtKind == 2 || e.lgtKind == 3) {
			return true
		}
	}
	return false
}

func (w *World) shadowLightDir() math32.Vector3 {
	dir := math32.Vector3{0.35, 0.85, 0.4}
	if e := w.ents[w.shadow.lightID]; e != nil && e.node != nil {
		var p math32.Vector3
		e.node.GetNode().WorldPosition(&p)
		if p.Length() > 0.01 {
			dir = p
		} else {
			e.node.GetNode().WorldDirection(&dir)
			dir.Negate()
		}
	} else {
		for _, e := range w.ents {
			if e != nil && e.castShadow && e.node != nil {
				var p math32.Vector3
				e.node.GetNode().WorldPosition(&p)
				if p.Length() > 0.01 {
					dir = p
					break
				}
			}
			if e != nil && e.lgtKind == 1 && e.node != nil {
				if _, ok := e.node.(*light.Directional); ok {
					var p math32.Vector3
					e.node.GetNode().WorldPosition(&p)
					if p.Length() > 0.01 {
						dir = p
						break
					}
				}
			}
		}
	}
	if dir.Length() < 0.01 {
		dir = math32.Vector3{0.35, 0.85, 0.4}
	}
	dir.Normalize()
	return dir
}

func (w *World) collectLocalLightViews(cascadeTiles int) int {
	w.shadow.pointN = 0
	w.shadow.spotN = 0
	tile := cascadeTiles
	for _, e := range w.ents {
		if e == nil || !e.castShadow || e.node == nil {
			continue
		}
		switch e.lgtKind {
		case 2:
			if w.shadow.pointN >= maxShadowPoints {
				continue
			}
			i := w.shadow.pointN
			w.shadow.pointN++
			w.shadow.pointPos[i] = entityWorldPos(e)
			w.shadow.pointRange[i] = entityLightRange(e)
			w.shadow.pointTile[i] = tile
			w.buildPointViews(i)
			tile += 6
		case 3:
			if w.shadow.spotN >= maxShadowSpots {
				continue
			}
			i := w.shadow.spotN
			w.shadow.spotN++
			w.shadow.spotPos[i] = entityWorldPos(e)
			w.shadow.spotDir[i] = entityWorldDir(e)
			w.shadow.spotRange[i] = entityLightRange(e)
			w.shadow.spotCos[i] = entitySpotCos(e)
			w.shadow.spotTile[i] = tile
			w.buildSpotView(i, e)
			tile++
		}
	}
	return tile
}

func entityWorldPos(e *Entity) math32.Vector3 {
	var p math32.Vector3
	e.node.GetNode().WorldPosition(&p)
	return p
}

func entityWorldDir(e *Entity) math32.Vector3 {
	var d math32.Vector3
	e.node.GetNode().WorldDirection(&d)
	d.Negate()
	if d.Length() < 0.01 {
		d = math32.Vector3{0, -1, 0}
	}
	d.Normalize()
	return d
}

func entityLightRange(e *Entity) float32 {
	if p, ok := e.node.(interface{ LinearDecay() float32 }); ok {
		d := p.LinearDecay()
		if d > 0.0001 {
			r := 1 / d
			if r > 1 {
				return r
			}
		}
	}
	return 16
}

func entitySpotCos(e *Entity) float32 {
	ang := float32(45)
	if s, ok := e.node.(*light.Spot); ok {
		ang = s.CutoffAngle()
	}
	if ang < 5 {
		ang = 5
	}
	if ang > 90 {
		ang = 90
	}
	return math32.Cos(ang * math32.Pi / 180)
}

var cubeFaceDir = [6]math32.Vector3{
	{1, 0, 0}, {-1, 0, 0},
	{0, 1, 0}, {0, -1, 0},
	{0, 0, 1}, {0, 0, -1},
}

var cubeFaceUp = [6]math32.Vector3{
	{0, -1, 0}, {0, -1, 0},
	{0, 0, 1}, {0, 0, -1},
	{0, -1, 0}, {0, -1, 0},
}

func (w *World) buildPointViews(i int) {
	s := &w.shadow
	pos := s.pointPos[i]
	far := s.pointRange[i]
	if far < 2 {
		far = 2
	}
	cam := s.perspCam(1, 0.08, far, 90)
	for f := 0; f < 6; f++ {
		s.tmpTarget.Copy(&pos).Add(&cubeFaceDir[f])
		cam.SetFar(far)
		cam.SetFov(90)
		cam.SetPosition(pos.X, pos.Y, pos.Z)
		s.tmpUp = cubeFaceUp[f]
		cam.LookAt(&s.tmpTarget, &s.tmpUp)
		cam.ViewMatrix(&s.tmpView)
		cam.ProjMatrix(&s.tmpProj)
		s.pointVP[i*6+f].MultiplyMatrices(&s.tmpProj, &s.tmpView)
	}
}

func (w *World) buildSpotView(i int, e *Entity) {
	s := &w.shadow
	pos := s.spotPos[i]
	dir := s.spotDir[i]
	far := s.spotRange[i]
	if far < 2 {
		far = 2
	}
	s.tmpTarget.Copy(&pos).Add(&dir)
	s.tmpUp = math32.Vector3{0, 1, 0}
	if math32.Abs(dir.Dot(&s.tmpUp)) > 0.94 {
		s.tmpUp = math32.Vector3{0, 0, 1}
	}
	fov := float32(70)
	if sl, ok := e.node.(*light.Spot); ok {
		fov = sl.CutoffAngle() * 2
	}
	if fov < 20 {
		fov = 20
	}
	if fov > 160 {
		fov = 160
	}
	cam := s.perspCam(1, 0.08, far, fov)
	cam.SetPosition(pos.X, pos.Y, pos.Z)
	cam.LookAt(&s.tmpTarget, &s.tmpUp)
	cam.ViewMatrix(&s.tmpView)
	cam.ProjMatrix(&s.tmpProj)
	s.spotVP[i].MultiplyMatrices(&s.tmpProj, &s.tmpView)
}

func shadowCasterStatic(e *Entity) bool {
	if e == nil || e.mesh == nil {
		return false
	}
	if e.name == "terrain" || e.name == "terrain_root" {
		return true
	}
	if e.bodyType == 2 {
		return true
	}
	return false
}

func (w *World) shadowStaticKey() uint64 {
	h := uint64(14695981039346656037)
	mix := func(u uint64) {
		h ^= u
		h *= 1099511628211
	}
	mixF := func(v float32) { mix(uint64(math.Float32bits(v))) }
	for i := 0; i < w.shadow.cascades; i++ {
		for _, v := range w.shadow.lightVP[i] {
			mixF(v)
		}
	}
	for i := 0; i < w.shadow.pointN*6; i++ {
		for _, v := range w.shadow.pointVP[i] {
			mixF(v)
		}
	}
	for i := 0; i < w.shadow.spotN; i++ {
		for _, v := range w.shadow.spotVP[i] {
			mixF(v)
		}
	}
	mix(uint64(w.shadow.filter))
	mix(uint64(w.shadow.size))
	mix(uint64(w.shadow.cascades))
	mix(uint64(bool01(w.shadow.atlas)))
	for _, e := range w.ents {
		if e == nil || e.mesh == nil || !e.mesh.Visible() || !shadowCasterStatic(e) {
			continue
		}
		m := e.mesh.GetNode().MatrixWorld()
		for _, v := range m {
			mixF(v)
		}
	}
	return h
}

func (w *World) shadowStaticStale() bool {
	if w.shadow.dirty {
		return true
	}
	for _, e := range w.ents {
		if e == nil || !shadowCasterStatic(e) {
			continue
		}
		if e.shadowGeomDirty {
			return true
		}
	}
	return false
}

func (w *World) shadowSceneKey() uint64 {
	h := uint64(14695981039346656037)
	mix := func(u uint64) {
		h ^= u
		h *= 1099511628211
	}
	mixF := func(v float32) { mix(uint64(math.Float32bits(v))) }
	for i := 0; i < w.shadow.cascades; i++ {
		for _, v := range w.shadow.lightVP[i] {
			mixF(v)
		}
	}
	for i := 0; i < w.shadow.pointN*6; i++ {
		for _, v := range w.shadow.pointVP[i] {
			mixF(v)
		}
	}
	for i := 0; i < w.shadow.spotN; i++ {
		for _, v := range w.shadow.spotVP[i] {
			mixF(v)
		}
	}
	mix(uint64(w.shadow.filter))
	mix(uint64(w.shadow.size))
	mix(uint64(w.shadow.cascades))
	mix(uint64(bool01(w.shadow.atlas)))
	for _, e := range w.ents {
		if e == nil || e.mesh == nil || !e.mesh.Visible() {
			continue
		}
		m := e.mesh.GetNode().MatrixWorld()
		for _, v := range m {
			mixF(v)
		}
	}
	return h
}

// MarkShadowDirty forces the next depth pass even when EnableShadowCache is on.
// Call after vertex/morph edits that do not change MatrixWorld().
func (w *World) MarkShadowDirty() {
	w.shadow.dirty = true
	w.shadow.cacheKey = 0
}

func (w *World) clearShadowDirty() {
	w.shadow.dirty = false
	for _, e := range w.ents {
		if e != nil {
			e.shadowGeomDirty = false
		}
	}
}

func (w *World) shadowCacheStale() bool {
	if w.shadow.dirty {
		return true
	}
	for _, e := range w.ents {
		if e == nil {
			continue
		}
		if e.shadowGeomDirty {
			return true
		}
		if e.anim != nil && e.anim.mode != 0 {
			return true
		}
	}
	return false
}

func (s *shadowMap) orthoCam(aspect, near, far, size float32) *camera.Camera {
	if s.tmpOrtho == nil {
		s.tmpOrtho = camera.NewOrthographic(aspect, near, far, size, camera.Vertical)
		return s.tmpOrtho
	}
	s.tmpOrtho.SetProjection(camera.Orthographic)
	s.tmpOrtho.SetAxis(camera.Vertical)
	s.tmpOrtho.SetAspect(aspect)
	s.tmpOrtho.SetNear(near)
	s.tmpOrtho.SetFar(far)
	s.tmpOrtho.SetSize(size)
	return s.tmpOrtho
}

func (s *shadowMap) perspCam(aspect, near, far, fov float32) *camera.Camera {
	if s.tmpPersp == nil {
		s.tmpPersp = camera.NewPerspective(aspect, near, far, fov, camera.Vertical)
		return s.tmpPersp
	}
	s.tmpPersp.SetProjection(camera.Perspective)
	s.tmpPersp.SetAxis(camera.Vertical)
	s.tmpPersp.SetAspect(aspect)
	s.tmpPersp.SetNear(near)
	s.tmpPersp.SetFar(far)
	s.tmpPersp.SetFov(fov)
	return s.tmpPersp
}

func shadowSkipMesh(e *Entity) bool {
	if e == nil || e.sky || e.meshNoCast {
		return true
	}
	if e.name == "water" || e.name == "atmosphere" {
		return true
	}
	if e.mat != nil {
		switch e.mat.Shader() {
		case "mbpart", "mbflake", "mbclouds", "mbatmo", "mbwater":
			return true
		}
	}
	return false
}

func (w *World) drawShadowCasters(gs *gls.GLS, lightVP *math32.Matrix4, mvpLoc int32, layer int) (drew int) {
	base := w.shadowDepthProgram()
	if gs == nil || base == nil || base.Handle() == 0 {
		return 0
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("EnableShadows: draw panic", r)
		}
	}()
	s := &w.shadow
	for _, e := range w.ents {
		if shadowSkipMesh(e) {
			continue
		}
		if layer == shadowLayerStatic && !shadowCasterStatic(e) {
			continue
		}
		if layer == shadowLayerDynamic && shadowCasterStatic(e) {
			continue
		}
		if e.mesh == nil || !e.mesh.Renderable() || !e.mesh.Visible() {
			continue
		}
		geom := e.mesh.GetGeometry()
		if geom == nil || s.primed == nil || !s.primed[geom] {
			continue
		}
		if e.mat != nil && e.mat.Side() == material.SideDouble {
			gs.Disable(gls.CULL_FACE)
		} else {
			gs.Enable(gls.CULL_FACE)
			gs.CullFace(gls.BACK)
		}
		n := e.mesh.GetNode()
		n.UpdateMatrixWorld()
		mm := n.MatrixWorld()
		s.tmpMVP.MultiplyMatrices(lightVP, &mm)
		prog := base
		cutout := e.albedoTex != nil && s.depthAlphaG3N != nil && s.depthAlphaG3N.Handle() != 0 &&
			w.shadow.filter != shadowFilterEVSM && w.shadow.filter != shadowFilterMSM
		if cutout {
			prog = s.depthAlphaG3N
		}
		gs.UseProgram(prog)
		loc := prog.GetUniformLocation("MVP")
		if loc < 0 {
			loc = mvpLoc
		}
		gs.UniformMatrix4fv(loc, 1, false, &s.tmpMVP[0])
		if cutout {
			e.albedoTex.RenderSetup(gs, 0, 0)
			if ul := prog.GetUniformLocation("Cutout"); ul >= 0 {
				gs.Uniform1i(ul, 0)
			}
			if ul := prog.GetUniformLocation("CutoutAlpha"); ul >= 0 {
				gs.Uniform1f(ul, 0.5)
			}
		} else {
			w.setDepthMomentUniforms(gs)
		}
		geom.RenderSetup(gs)
		idx := geom.Indices()
		if len(idx) > 0 {
			gs.DrawElements(gls.TRIANGLES, int32(len(idx)), gls.UNSIGNED_INT, 0)
		} else {
			gs.DrawArrays(gls.TRIANGLES, 0, int32(geom.Items()))
		}
		drew++
	}
	if layer != shadowLayerStatic {
		if n := w.drawGPUInstanceDepth(lightVP); n > 0 {
			drew += n
		}
	}
	return drew
}

func compileGLShader(kind uint32, src string) (uint32, error) {
	sh := gl.CreateShader(kind)
	csrc, free := gl.Strs(src + "\x00")
	gl.ShaderSource(sh, 1, csrc, nil)
	free()
	gl.CompileShader(sh)
	var ok int32
	gl.GetShaderiv(sh, gl.COMPILE_STATUS, &ok)
	if ok == gl.FALSE {
		log := glInfoLog(sh, false)
		gl.DeleteShader(sh)
		return 0, fmt.Errorf("shadow depth compile: %s", log)
	}
	return sh, nil
}

func glInfoLog(id uint32, program bool) string {
	var n int32
	if program {
		gl.GetProgramiv(id, gl.INFO_LOG_LENGTH, &n)
	} else {
		gl.GetShaderiv(id, gl.INFO_LOG_LENGTH, &n)
	}
	if n <= 1 {
		return "unknown"
	}
	buf := make([]byte, n)
	if program {
		gl.GetProgramInfoLog(id, n, nil, &buf[0])
	} else {
		gl.GetShaderInfoLog(id, n, nil, &buf[0])
	}
	return strings.TrimSpace(string(buf))
}

func (w *World) litShaderName() string {
	if w.shadow.on || w.fogMode != 0 {
		return "bsshadow"
	}
	return "standard"
}

func (w *World) useShadowMaterials(on bool) {
	_ = on
	w.applyLitShaders()
}

func keepCustomLitShader(e *Entity) bool {
	if e == nil {
		return false
	}
	if e.sky || e.name == "water" || e.name == "atmosphere" {
		return true
	}
	if e.mat == nil {
		return false
	}
	switch e.mat.Shader() {
	case "mbwater", "mbterrain", "mbclouds", "mbatmo", "mbpart", "mbflake":
		return true
	}
	return false
}

func (w *World) setMeshReceiveShadow(e *Entity, on bool) {
	if e == nil {
		return
	}
	if e.pbrWrap != nil {
		e.pbrWrap.recvShadow = on
	}
	apply := func(mesh *graphic.Mesh) {
		if mesh == nil {
			return
		}
		im := mesh.GetMaterial(0)
		if lm, ok := im.(*litMat); ok {
			lm.recvShadow = on
		}
		if pm, ok := im.(*pbrMat); ok {
			pm.recvShadow = on
		}
	}
	if e.mesh != nil {
		apply(e.mesh)
		return
	}
	if e.node != nil {
		w.walkMeshes(e.node, apply)
	}
}

func (w *World) applyLitShaders() {
	name := w.litShaderName()
	for _, e := range w.ents {
		if e == nil || e.sky {
			continue
		}
		if e.name == "water" || (e.mat != nil && e.mat.Shader() == "mbwater") {
			bindWaterMaterial(e.mat)
			continue
		}
		if keepCustomLitShader(e) {
			continue
		}
		if e.pbr != nil {
			e.pbr.SetShader("mbphysical")
		}
		if e.pbrWrap != nil && e.pbrWrap.Physical != nil {
			e.pbrWrap.Physical.SetShader("mbphysical")
		}
		if e.mat != nil && !e.usePBR {
			e.mat.SetShader(name)
		}
		if e.usePBR && e.node != nil {
			w.retargetPBRShaders(e.node)
		}
	}
}

func (w *World) bindFogUniforms(gs *gls.GLS) {
	if gs == nil {
		return
	}
	setUni1i(gs, "FogMode", w.fogMode)
	setUni3f(gs, "FogColor", w.fogRGB.R, w.fogRGB.G, w.fogRGB.B)
	setUni1f(gs, "FogNear", w.fogNear)
	setUni1f(gs, "FogFar", w.fogFar)
	dens := w.fogDensity
	if dens <= 0 {
		dens = 2 / max32(w.fogFar, 1)
	}
	setUni1f(gs, "FogDensity", dens)
	setUni1f(gs, "FogHeight", w.fogHeight)
	setUni1f(gs, "FogHeightFalloff", w.fogHFall)
	setUni1f(gs, "Wetness", w.wetness)
	w.bindWaterExtras(gs)
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func (w *World) ensureShadowOn() {
	w.shadow.on = true
	if w.shadow.size <= 0 {
		w.shadow.size = 2048
	}
	if w.shadow.cascades < 1 {
		w.shadow.cascades = 4
	}
	if w.shadow.pcf < 3 {
		w.shadow.pcf = 3
	}
	if w.shadow.distance < 40 {
		w.shadow.distance = defaultShadowDistance
	}
	if w.shadow.bias <= 0 {
		w.shadow.bias = 0.0025
	}
	w.useShadowMaterials(true)
}
