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
	imgs    [6]*image.RGBA
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
		m.SetDepthTest(true)
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
	return texturesFromSky(clearSkyImages(-0.35, 0.62, 0.70, nil))
}

func clearSkyImages(sx, sy, sz float64, haze *math32.Color) [6]*image.RGBA {
	return generateSkyImages(
		math32.Color{0.16, 0.40, 0.82},
		math32.Color{0.42, 0.66, 0.90},
		sx, sy, sz, haze,
	)
}

func texturesFromSky(imgs [6]*image.RGBA) [6]*texture.Texture2D {
	var out [6]*texture.Texture2D
	for i, img := range imgs {
		if img != nil {
			out[i] = texture.NewTexture2DFromRGBA(img)
		}
	}
	return out
}

func (w *World) publishSky(imgs [6]*image.RGBA) (int, error) {
	id, err := w.createSkyBoxFromFaces(texturesFromSky(imgs))
	if err != nil {
		return 0, err
	}
	if s := w.skies[id]; s != nil {
		s.imgs = imgs
	}
	w.useSkyImages(imgs)
	return id, nil
}

func (w *World) useSkyImages(imgs [6]*image.RGBA) {
	if imgs[0] == nil {
		return
	}
	if w.skyImgs[0] == imgs[0] {
		return
	}
	w.skyImgs = imgs
	w.skyRev++
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
	haze := w.skyHaze()
	hazeKey := math32.Color{-1, -1, -1}
	if haze != nil {
		hazeKey = *haze
	}
	sameHaze := colorDist(hazeKey, w.skyHazeKey) < 0.02
	// A running clock blends the sky colors too (setPlayHour)
	sameCols := colorDist(w.skyTop, w.skyTopKey)+colorDist(w.skyBot, w.skyBotKey) < 0.012
	if w.skySunOK && sameHaze && sameCols && math32.Abs(w.skySun.X-d.X)+math32.Abs(w.skySun.Y-d.Y)+math32.Abs(w.skySun.Z-d.Z) < 0.004 {
		return
	}
	if w.scene == nil || w.skies == nil || len(w.skies) == 0 {
		w.skySun = d
		w.skySunOK = true
		w.skyHazeKey = hazeKey
		return
	}
	// One repaint at a time and at most every 6th frame: a fast clock
	// moves the sun every frame. tickSky asks again once this one lands.
	if w.skyBusy || (w.renderSerial > 0 && w.renderSerial-w.skyPaintAt < 6) {
		w.skyAgain = true
		return
	}
	w.skyPaintAt = w.renderSerial
	w.skySun = d
	w.skySunOK = true
	w.skyHazeKey = hazeKey
	w.skyTopKey, w.skyBotKey = w.skyTop, w.skyBot
	top, bot := w.skyTop, w.skyBot
	var hz *math32.Color
	if haze != nil {
		c := *haze
		hz = &c
	}
	sx, sy, sz := float64(d.X), float64(d.Y), float64(d.Z)
	paint := func() [6]*image.RGBA {
		if top.R+top.G+top.B > 0 {
			return generateSkyImages(top, bot, sx, sy, sz, hz)
		}
		return clearSkyImages(sx, sy, sz, hz)
	}
	// Before the first frame, paint now so the scene opens on the right
	// sky. After, a moving sun or clock repaints on a worker: six 192 px
	// faces were a 10-20 ms hitch each time.
	if w.renderSerial == 0 {
		w.swapProcSky(paint())
		return
	}
	w.skyBusy = true
	jobs := w.ensureJobs()
	jobs.submit(func() {
		imgs := paint()
		jobs.enqueueGL(func() {
			w.skyBusy = false
			w.swapProcSky(imgs)
		})
	})
}

// tickSky repaints a sky change that arrived while one was painting or
// too soon after the last, so the final hour of a stopped clock still lands.
func (w *World) tickSky() {
	if w.skyAgain && !w.skyBusy && w.renderSerial-w.skyPaintAt >= 6 {
		w.skyAgain = false
		w.syncVisualSun()
	}
}

func colorDist(a, b math32.Color) float32 {
	return math32.Abs(a.R-b.R) + math32.Abs(a.G-b.G) + math32.Abs(a.B-b.B)
}

// swapProcSky shows a repainted procedural sky and frees the one it
// replaces, so a running clock does not pile up skyboxes.
func (w *World) swapProcSky(imgs [6]*image.RGBA) {
	if w.scene == nil {
		return
	}
	id, err := w.publishSky(imgs)
	if err != nil {
		return
	}
	prev := w.skyProcID
	w.setSkyBox(id)
	w.skyProcID = id
	if prev != 0 && prev != id {
		w.freeSkyBox(prev)
	}
}

func generateSkyFaces(top, bot math32.Color, sunX, sunY, sunZ float64) [6]*texture.Texture2D {
	return texturesFromSky(generateSkyImages(top, bot, sunX, sunY, sunZ, nil))
}

func generateSkyImages(top, bot math32.Color, sunX, sunY, sunZ float64, haze *math32.Color) [6]*image.RGBA {
	// 192 a face: at 128 the sun could only be a soft blob.
	const n = 192
	sl := math.Sqrt(sunX*sunX + sunY*sunY + sunZ*sunZ)
	if sl > 1e-6 {
		sunX, sunY, sunZ = sunX/sl, sunY/sl, sunZ/sl
	}
	// A low sun warms its halo toward orange.
	low := clamp01f(1 - sunY*2.2)
	// A dark top is night: stars, and a moon opposite the sun, held above
	// the horizon.
	night := clamp01f((0.16 - (float64(top.R)+float64(top.G)+float64(top.B))/3) / 0.06)
	mx, my, mz := -sunX, -sunY, -sunZ
	if my < 0.35 {
		my = 0.35
	}
	if ml := math.Sqrt(mx*mx + my*my + mz*mz); ml > 1e-6 {
		mx, my, mz = mx/ml, my/ml, mz/ml
	}
	var out [6]*image.RGBA
	for face := 0; face < 6; face++ {
		img := image.NewRGBA(image.Rect(0, 0, n, n))
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				dx, dy, dz := skyFaceDir(face, x, y, n)
				inv := 1 / math.Sqrt(dx*dx+dy*dy+dz*dz)
				dx, dy, dz = dx*inv, dy*inv, dz*inv
				// Terrain-OpenGL sky.frag: mix bottom→top by elevation.
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
				// The horizon band is the fog color, so fogged land at the
				// edge of view melts into the sky instead of standing out.
				if haze != nil {
					h := math.Exp(-math.Abs(dy) * 9)
					if dy < 0 {
						h = math.Max(h, clamp01f(-dy*6))
					}
					h *= 0.92
					r += (float64(haze.R) - r) * h
					g += (float64(haze.G) - g) * h
					b += (float64(haze.B) - b) * h
				}
				if night > 0 && dy > -0.02 {
					// Stars: one in ~140 texels, most faint, a few bright.
					hs := skyHash(face, x, y)
					if hs > 0.9928 {
						br := math.Pow((hs-0.9928)/0.0072, 3) * 0.9
						br *= clamp01f(dy*4) * night
						r += br * 0.92
						g += br * 0.95
						b += br
					}
					// Moon: a pale disc with soft maria, and a cool halo.
					mc := dx*mx + dy*my + dz*mz
					if mc > 0.99 {
						disc := smoothstep64(0.99955, 0.99968, mc)
						maria := 0.82 + 0.18*skyHash(face, x/3, y/3)
						glow := math.Pow(math.Max(mc, 0), 400) * 0.22
						r += (disc*0.85*maria + glow*0.7) * night
						g += (disc*0.88*maria + glow*0.8) * night
						b += (disc*0.92*maria + glow) * night
					}
				}
				cosA := dx*sunX + dy*sunY + dz*sunZ
				// The night sun stays a dim light above the horizon; its disc
				// fades out so the moon is the only one in the sky.
				if day := 1 - night; cosA > 0 && day > 0 {
					// A crisp disc (about 1.4 degrees), then halos from tight
					// to broad, warmer as the sun nears the horizon.
					disc := smoothstep64(0.99962, 0.99975, cosA) * day
					halo := (math.Pow(cosA, 900)*0.55 + math.Pow(cosA, 64)*0.22 + math.Pow(cosA, 8)*0.10) * day
					hr, hg, hb := 1.0, 0.86-0.18*low, 0.62-0.34*low
					r += disc*1.2 + halo*hr
					g += disc*1.1 + halo*hg
					b += disc*0.9 + halo*hb
				}
				img.SetRGBA(x, y, color.RGBA{u8f(r), u8f(g), u8f(b), 255})
			}
		}
		out[face] = img
	}
	return out
}

// skyHash is a stable 0..1 per face texel (stars, moon maria).
func skyHash(face, x, y int) float64 {
	h := uint32(face*73856093) ^ uint32(x*19349663) ^ uint32(y*83492791)
	h ^= h >> 13
	h *= 0x5bd1e995
	h ^= h >> 15
	return float64(h&0xffffff) / float64(0xffffff)
}

// skyHaze is the color the sky's horizon blends into. Off: a script's fog
// is usually a fixed color, and blending a bright sky toward it painted a
// dark band above the horizon. fogTint matches the sun side instead.
func (w *World) skyHaze() *math32.Color {
	return nil
}

func (w *World) applySkyPreset(name string) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "sunset":
		// Dusky blue-violet over a warm peach horizon. The old olive top
		// (177,174,119) read as a sickly sky, and its blue-grey fog made the
		// land a dark band under a pink horizon.
		w.skyTop = math32.Color{0.30, 0.37, 0.62}
		w.skyBot = math32.Color{0.98, 0.62, 0.44}
		w.fogRGB = math32.Color{0.74, 0.55, 0.47}
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
	id, err := w.publishSky(generateSkyImages(w.skyTop, w.skyBot, sx, sy, sz, w.skyHaze()))
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

func (w *World) loadSkyImages(paths [6]string) ([6]*image.RGBA, error) {
	var imgs [6]*image.RGBA
	for i, p := range paths {
		path, err := w.openPath(p)
		if err != nil {
			path = w.resolve(p)
		}
		img, err := loadRGBAFile(path)
		if err != nil {
			return imgs, err
		}
		imgs[i] = img
	}
	return imgs, nil
}

func loadRGBAFile(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba, nil
	}
	b := img.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x, y, img.At(x, y))
		}
	}
	return out, nil
}

func (w *World) trySkyPrefix(prefix string) ([6]*image.RGBA, bool) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || strings.EqualFold(prefix, "default") {
		return [6]*image.RGBA{}, false
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
			imgs, err := w.loadSkyImages(paths)
			if err == nil {
				return imgs, true
			}
		}
	}
	return [6]*image.RGBA{}, false
}

func (w *World) makeSkyBox(prefix string) (int, error) {
	if imgs, ok := w.trySkyPrefix(prefix); ok {
		w.skyProc = false
		return w.publishSky(imgs)
	}
	w.skyProc = true
	sx, sy, sz := w.proceduralSkySun()
	if w.skyTop.R+w.skyTop.G+w.skyTop.B > 0 {
		return w.publishSky(generateSkyImages(w.skyTop, w.skyBot, sx, sy, sz, w.skyHaze()))
	}
	return w.publishSky(clearSkyImages(sx, sy, sz, w.skyHaze()))
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
	if s := w.skies[id]; s != nil {
		w.skyID = id
		w.useSkyImages(s.imgs)
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
	if s.box != nil {
		s.box.Dispose() // its six face textures
	}
	delete(w.skies, id)
	if w.skyID == id {
		w.skyID = 0
	}
	if w.skyProcID == id {
		w.skyProcID = 0
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
