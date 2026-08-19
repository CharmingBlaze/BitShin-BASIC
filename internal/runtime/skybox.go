package runtime

import (
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/texture"
)

// skyBox is G3N's skybox: a unit cube whose view-space translation is
// stripped so the camera is always at the center. A world-space 800-cube
// whose corners sit at ~693 will clip when far < ~700, leaving one face
// (a trapezoid) in the clear color.
type skyBox struct {
	graphic.Graphic
	uniMm   gls.Uniform
	uniMVm  gls.Uniform
	uniMVPm gls.Uniform
	uniNm   gls.Uniform
}

type skySlot struct {
	box     *skyBox
	visible bool
}

func (s *skyBox) RenderSetup(gs *gls.GLS, rinfo *core.RenderInfo) {
	mvm := *s.ModelViewMatrix()
	mvm[12] = 0
	mvm[13] = 0
	mvm[14] = 0

	loc := s.uniMVm.Location(gs)
	gs.UniformMatrix4fv(loc, 1, false, &mvm[0])

	var mvpm math32.Matrix4
	mvpm.MultiplyMatrices(&rinfo.ProjMatrix, &mvm)
	loc = s.uniMVPm.Location(gs)
	gs.UniformMatrix4fv(loc, 1, false, &mvpm[0])

	var nm math32.Matrix3
	nm.GetNormalMatrix(&mvm)
	loc = s.uniNm.Location(gs)
	gs.UniformMatrix3fv(loc, 1, false, &nm[0])

	// Keep ModelMatrix translation-free so any shader using it stays camera-centered.
	var mm math32.Matrix4
	mm.Identity()
	loc = s.uniMm.Location(gs)
	gs.UniformMatrix4fv(loc, 1, false, &mm[0])
}

func prepareSkyTex(tex *texture.Texture2D) *texture.Texture2D {
	if tex == nil {
		tex = texture.NewTexture2DFromRGBA(solidRGBA(8, 8, 80, 140, 200))
	}
	tex.SetWrapS(gls.CLAMP_TO_EDGE)
	tex.SetWrapT(gls.CLAMP_TO_EDGE)
	tex.SetMinFilter(gls.LINEAR)
	tex.SetMagFilter(gls.LINEAR)
	return tex
}

func skyFitScale(near, far float32) float32 {
	if near < 0.05 {
		near = 0.05
	}
	if far < near+2 {
		far = near + 2
	}
	// NewCube(1) spans [-0.5, 0.5]; scaled corners are at 0.5*s*sqrt(3).
	s := far * 0.4
	minS := near * 8
	if minS < 4 {
		minS = 4
	}
	if s < minS {
		s = minS
	}
	maxS := far * 1.05
	if s > maxS {
		s = maxS
	}
	return s
}

func (w *World) createSkyBoxFromFaces(faces [6]*texture.Texture2D) (int, error) {
	sky := new(skyBox)
	geom := geometry.NewCube(1)
	sky.Graphic.Init(sky, geom, gls.TRIANGLES)
	sky.SetCullable(false)
	// G3N opaque pass is front-to-back over a back-to-front sort, so a *high*
	// render order is drawn first. -10 would paint the sky last and cover terrain.
	sky.SetRenderOrder(100)
	sky.uniMm.Init("ModelMatrix")
	sky.uniMVm.Init("ModelViewMatrix")
	sky.uniMVPm.Init("MVP")
	sky.uniNm.Init("NormalMatrix")

	for i := 0; i < 6; i++ {
		m := material.NewStandard(math32.NewColor("white"))
		m.AddTexture(prepareSkyTex(faces[i]))
		// Interior of the cube: SideDouble so a winding mismatch cannot hide 5 faces.
		m.SetSide(material.SideDouble)
		m.SetUseLights(material.UseLightNone)
		m.SetDepthMask(false)
		m.SetDepthTest(false)
		m.SetDepthFunc(gls.LEQUAL)
		sky.AddGroupMaterial(sky, m, i)
	}

	if w.scene != nil {
		w.scene.Add(sky)
	}
	id := w.nextSky
	w.nextSky++
	if w.skies == nil {
		w.skies = map[int]*skySlot{}
	}
	for _, s := range w.skies {
		if s != nil && s.box != nil {
			s.box.SetVisible(false)
			s.visible = false
		}
	}
	w.skies[id] = &skySlot{box: sky, visible: true}
	w.skyID = id
	w.syncSkyboxes()
	return id, nil
}

func solidRGBA(w, h int, r, g, b uint8) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	c := color.RGBA{r, g, b, 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func generateDefaultSkyFaces() [6]*texture.Texture2D {
	return generateAtmosphereCubemap(-0.35, 0.62, 0.70, 1, 1, 1.1)
}

func (w *World) proceduralSkySun() (sx, sy, sz float64) {
	if w.hasShadowLight() || w.dirLightEntity() != nil {
		d := w.shadowLightDir()
		return float64(d.X), float64(d.Y), float64(d.Z)
	}
	return -0.35, 0.62, 0.70
}

func (w *World) dirLightEntity() *Entity {
	if e := w.ents[w.shadow.lightID]; e != nil && e.lgtKind == 1 {
		return e
	}
	for _, e := range w.ents {
		if e != nil && e.lgtKind == 1 {
			return e
		}
	}
	return nil
}

func (w *World) aimDirLightVec(x, y, z float32) {
	d := math32.Vector3{x, y, z}
	if d.Length() < 0.01 {
		return
	}
	d.Normalize()
	e := w.dirLightEntity()
	if e == nil || e.node == nil {
		return
	}
	e.node.GetNode().SetPosition(d.X, d.Y, d.Z)
}

func (w *World) syncVisualSun() {
	d := w.shadowLightDir()
	if w.atmo != nil {
		w.atmo.sunX, w.atmo.sunY, w.atmo.sunZ = d.X, d.Y, d.Z
	}
	if !w.skyProc {
		w.skySun = d
		return
	}
	if w.skySunOK && math32.Abs(w.skySun.X-d.X)+math32.Abs(w.skySun.Y-d.Y)+math32.Abs(w.skySun.Z-d.Z) < 0.002 {
		return
	}
	w.skySun = d
	w.skySunOK = true
	if w.scene == nil || w.skies == nil || len(w.skies) == 0 {
		return
	}
	sx, sy, sz := float64(d.X), float64(d.Y), float64(d.Z)
	faces := generateAtmosphereCubemap(sx, sy, sz, 1, 1, 1.1)
	if w.skyTop.R+w.skyTop.G+w.skyTop.B > 0 {
		faces = generateSkyFaces(w.skyTop, w.skyBot, sx, sy, sz)
	}
	id, err := w.createSkyBoxFromFaces(faces)
	if err == nil {
		w.setSkyBox(id)
	}
}

func generateSkyFaces(top, bot math32.Color, sunX, sunY, sunZ float64) [6]*texture.Texture2D {
	const n = 128
	var out [6]*texture.Texture2D
	for face := 0; face < 6; face++ {
		img := image.NewRGBA(image.Rect(0, 0, n, n))
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				dx, dy, dz := skyFaceDir(face, x, y, n)
				inv := 1 / math.Sqrt(dx*dx+dy*dy+dz*dz)
				dx, dy, dz = dx*inv, dy*inv, dz*inv
				// Terrain-OpenGL sky.frag: mix bottom→top by elevation, plus sun disk.
				elev := clamp01f(dy*0.5 + 0.5)
				fade := clamp01f(1 - math.Exp(8.5-17*elev))
				r := float64(bot.R) + (float64(top.R)-float64(bot.R))*fade
				g := float64(bot.G) + (float64(top.G)-float64(bot.G))*fade
				b := float64(bot.B) + (float64(top.B)-float64(bot.B))*fade
				if dy < 0 {
					ground := -dy
					r = r*(1-ground*0.55) + 0.20*ground
					g = g*(1-ground*0.45) + 0.26*ground
					b = b*(1-ground*0.35) + 0.18*ground
				}
				sun := math.Max(0, dx*sunX+dy*sunY+dz*sunZ)
				disk := math.Pow(sun, 32)
				glow := math.Pow(sun, 8) * 0.35
				r += disk*0.9 + glow
				g += disk*0.75 + glow*0.7
				b += disk * 0.35
				img.SetRGBA(x, y, color.RGBA{u8f(r), u8f(g), u8f(b), 255})
			}
		}
		out[face] = texture.NewTexture2DFromRGBA(img)
	}
	return out
}

func (w *World) applySkyPreset(name string) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "sunset":
		w.skyTop = math32.Color{177.0 / 255, 174.0 / 255, 119.0 / 255}
		w.skyBot = math32.Color{234.0 / 255, 125.0 / 255, 125.0 / 255}
		w.fogRGB = math32.Color{85.0 / 255, 97.0 / 255, 120.0 / 255}
	case "sunset1":
		w.skyTop = math32.Color{133.0 / 255, 158.0 / 255, 214.0 / 255}
		w.skyBot = math32.Color{241.0 / 255, 161.0 / 255, 161.0 / 255}
		w.fogRGB = math32.Color{128.0 / 255, 153.0 / 255, 179.0 / 255}
	default:
		w.skyTop = math32.Color{0.525, 0.735, 0.84}
		w.skyBot = math32.Color{0.9, 0.9, 0.95}
		w.fogRGB = math32.Color{0.5, 0.6, 0.7}
	}
	sx, sy, sz := w.proceduralSkySun()
	id, err := w.createSkyBoxFromFaces(generateSkyFaces(w.skyTop, w.skyBot, sx, sy, sz))
	if err == nil {
		w.skyProc = true
		w.setSkyBox(id)
	}
}

func skyFaceDir(face, x, y, size int) (dx, dy, dz float64) {
	u := (float64(x)+0.5)/float64(size)*2 - 1
	v := (float64(y)+0.5)/float64(size)*2 - 1
	// Bleed past the face so clamp-to-edge samples match the adjacent cube face.
	if size > 1 {
		u *= float64(size) / float64(size-1)
		v *= float64(size) / float64(size-1)
	}
	switch face {
	case 0:
		return 1, -v, -u
	case 1:
		return -1, -v, u
	case 2:
		return u, 1, v
	case 3:
		return u, -1, -v
	case 4:
		return u, -v, 1
	default:
		return -u, -v, -1
	}
}

func (w *World) loadSkyFaces(paths [6]string) ([6]*texture.Texture2D, error) {
	var faces [6]*texture.Texture2D
	for i, p := range paths {
		path, err := w.openPath(p)
		if err != nil {
			path = w.resolve(p)
		}
		tex, err := texture.NewTexture2DFromImage(path)
		if err != nil {
			return faces, err
		}
		faces[i] = tex
	}
	return faces, nil
}

func (w *World) trySkyPrefix(prefix string) ([6]*texture.Texture2D, bool) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || strings.EqualFold(prefix, "default") {
		return [6]*texture.Texture2D{}, false
	}
	sets := [][6]string{
		{"px", "nx", "py", "ny", "pz", "nz"},
		{"_px", "_nx", "_py", "_ny", "_pz", "_nz"},
		{"right", "left", "up", "down", "front", "back"},
		{"posx", "negx", "posy", "negy", "posz", "negz"},
	}
	exts := []string{".png", ".jpg", ".jpeg"}
	base := strings.TrimRight(prefix, `/\`)
	for _, suf := range sets {
		for _, ext := range exts {
			var paths [6]string
			ok := true
			for i := 0; i < 6; i++ {
				p := base + suf[i] + ext
				if _, err := os.Stat(w.resolve(p)); err != nil {
					p2 := filepath.Join(base, suf[i]+ext)
					if _, err2 := os.Stat(w.resolve(p2)); err2 != nil {
						ok = false
						break
					}
					p = p2
				}
				paths[i] = p
			}
			if !ok {
				continue
			}
			faces, err := w.loadSkyFaces(paths)
			if err == nil {
				return faces, true
			}
		}
	}
	return [6]*texture.Texture2D{}, false
}

func (w *World) makeSkyBox(prefix string) (int, error) {
	if faces, ok := w.trySkyPrefix(prefix); ok {
		w.skyProc = false
		return w.createSkyBoxFromFaces(faces)
	}
	w.skyProc = true
	sx, sy, sz := w.proceduralSkySun()
	if w.skyTop.R+w.skyTop.G+w.skyTop.B > 0 {
		return w.createSkyBoxFromFaces(generateSkyFaces(w.skyTop, w.skyBot, sx, sy, sz))
	}
	return w.createSkyBoxFromFaces(generateAtmosphereCubemap(sx, sy, sz, 1, 1, 1.1))
}

func (w *World) setSkyBox(id int) {
	if w.skies == nil {
		return
	}
	for k, s := range w.skies {
		on := k == id && s != nil
		if s != nil && s.box != nil {
			s.box.SetVisible(on)
			s.visible = on
		}
	}
	if w.skies[id] != nil {
		w.skyID = id
	}
}

func (w *World) hideSkyBox(show bool) {
	if s := w.skies[w.skyID]; s != nil && s.box != nil {
		s.box.SetVisible(show)
		s.visible = show
	}
}

func (w *World) freeSkyBox(id int) {
	if id == 0 {
		id = w.skyID
	}
	s := w.skies[id]
	if s == nil {
		return
	}
	if s.box != nil {
		if p := s.box.GetNode().Parent(); p != nil {
			p.GetNode().Remove(s.box)
		}
		s.box.SetVisible(false)
	}
	delete(w.skies, id)
	if w.skyID == id {
		w.skyID = 0
	}
}

func (w *World) clearSkies() {
	for id := range w.skies {
		w.freeSkyBox(id)
	}
}

func (w *World) syncSkyboxes() {
	near, far := float32(0.3), float32(4000)
	var p math32.Vector3
	if w.cam != nil {
		w.cam.WorldPosition(&p)
		near, far = w.cam.Near(), w.cam.Far()
	}
	sc := skyFitScale(near, far)
	for _, s := range w.skies {
		if s == nil || !s.visible || s.box == nil {
			continue
		}
		s.box.SetCullable(false)
		s.box.SetPosition(p.X, p.Y, p.Z)
		s.box.SetScale(sc, sc, sc)
	}
}

func (w *World) skyVisible() bool {
	s := w.skies[w.skyID]
	return s != nil && s.visible
}
