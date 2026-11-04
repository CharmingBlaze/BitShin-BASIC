package runtime

import (
	"fmt"
	"strings"

	"moonbasic/internal/value"
)

// Quality tiers. Low is a 2016 laptop iGPU (Intel HD 520 class) at 720p.
// Ultra assumes a current discrete card.
const (
	qualityLow = iota
	qualityMedium
	qualityHigh
	qualityUltra
)

var qualityNames = [...]string{"low", "medium", "high", "ultra"}

// tierSettings is everything one tier changes. A script that sets one of
// these itself after SetGraphicsQuality keeps its own value until the next
// tier change.
type tierSettings struct {
	shadow     int     // shadow map size
	reflect    int     // water reflection target, 0 = off
	refract    int     // water refraction + depth target
	post       bool    // tonemap / bloom / FXAA pass
	radius     int     // detailed terrain chunks around the camera
	detailPx   float32 // SetTerrainDetail
	scatter    float32 // tree / prop / grass count scale
	treeReachM float32 // horizon trees stop here (still inside the fog)
	ao         bool    // screen-space ambient occlusion
	msaa       int     // scene samples when post is on; 0 is FXAA alone. On a 780M the island pays ~1.8 ms for 2x (10% of frames past 16.7 ms), so high has none; 4x ~4 ms
	softShadow bool    // PCSS: shadow edges soften with distance from the caster
	cascades   int     // sun shadow cascades
	aoSlices   int     // horizon directions per ambient occlusion sample
	maxTexture int     // largest side a loaded texture keeps; 0 is no limit
}

var tiers = [...]tierSettings{
	// Low keeps the post pass. Without it the scene draws into G3N's 8x
	// multisampled window: about 3x slower on a 780M, with a stall of seconds
	// on the way there. It only loses bloom (see bloomAllowed).
	qualityLow:    {shadow: 1024, reflect: 0, refract: 0, post: true, radius: 1, detailPx: 14, scatter: 0.45, treeReachM: 220, cascades: 2, maxTexture: 1024},
	qualityMedium: {shadow: 1024, reflect: 512, refract: 512, post: true, radius: 2, detailPx: 10, scatter: 0.7, treeReachM: 320, ao: true, aoSlices: 2, cascades: 3, maxTexture: 2048},
	qualityHigh:   {shadow: 2048, reflect: 1024, refract: 1024, post: true, radius: 2, detailPx: 8, scatter: 1, treeReachM: 420, ao: true, aoSlices: 4, cascades: 4},
	qualityUltra:  {shadow: 2048, reflect: 2048, refract: 1024, post: true, radius: 3, detailPx: 6, scatter: 1, treeReachM: 480, ao: true, aoSlices: 6, softShadow: true, msaa: 4, cascades: 4},
}

type qualityState struct {
	set   bool // a tier has been chosen; otherwise every setting is the script's own
	auto  bool
	tier  int
	reach float32 // horizon tree reach; 0 = the old radius+6 chunks
	// Auto watch: frame time is averaged over a window after a warm-up.
	warm   float64
	acc    float64
	frames int
	drops  int
}

// autoTier guesses a starting tier from the GL renderer string. The frame
// watch corrects a wrong guess downward within a few seconds, but it never goes up,
// and starting too high costs a stall to step down, so a guess errs low. The families
// are from their published specs: none of these but the 780M has been run here.
func autoTier(renderer string) int {
	r := strings.NewReplacer("(tm)", "", "(r)", "").Replace(strings.ToLower(renderer))
	has := func(names ...string) bool {
		for _, n := range names {
			if strings.Contains(r, n) {
				return true
			}
		}
		return false
	}
	switch {
	case r == "", has("llvmpipe", "softpipe", "basic render", "swiftshader"):
		return qualityLow
	case has("intel"):
		switch {
		case has(" arc"):
			return qualityHigh
		case has("hd graphics", "gma"):
			return qualityLow
		}
		return qualityMedium
	case has("mali", "adreno"):
		return qualityMedium
	case has("gt 1030", "gtx 6", "gtx 7", "gtx 950", "gtx 960", "940m", "930m", "920m", "mx1", "mx2"):
		return qualityMedium
	case has("gt 6", "gt 7", "geforce 9", "geforce 8", "geforce 7", "geforce 6", "gts ", "gtx 2", "gtx 4", "gtx 5", "gt 2", "gt 3", "gt 4", "gt 5"):
		return qualityLow
	case has("radeon hd 2", "radeon hd 3", "radeon hd 4", "radeon hd 5", "radeon hd 6", "radeon hd 8", "radeon r2", "radeon r3", "radeon r4", "radeon r5"):
		return qualityLow
	case has("radeon hd", "radeon r7", "radeon r9", "vega 3", "vega 6", "vega 8", "vega 10"):
		return qualityMedium
	}
	return qualityHigh
}

func parseQuality(s string) (tier int, auto bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low", "classic", "0":
		return qualityLow, false
	case "medium", "med", "enhanced", "1":
		return qualityMedium, false
	case "high", "advanced", "2":
		return qualityHigh, false
	case "ultra", "3":
		return qualityUltra, false
	}
	return qualityHigh, true
}

func (w *World) setQuality(tier int) {
	if tier < qualityLow {
		tier = qualityLow
	}
	if tier > qualityUltra {
		tier = qualityUltra
	}
	q := &w.quality
	prevScatter := float32(1)
	if q.set {
		prevScatter = tiers[q.tier].scatter
	}
	q.set = true
	q.tier = tier
	s := tiers[tier]
	q.reach = s.treeReachM
	w.shadow.size = s.shadow
	if w.post.on != s.post {
		w.post.on = s.post
		if s.post {
			// The neutral curve, as EnablePostFX gives. The still-default
			// Reinhard turned a tier change into a grey, darker picture.
			if w.post.tonemap == 0 {
				w.post.tonemap = 1
			}
			if w.post.exposure <= 0 {
				w.post.exposure = 1.05
			}
			w.setPostDefaults()
		}
	}
	// A tier can take an effect away, and it can offer one, but what the script turned
	// off stays off: not every game wants every effect.
	w.post.ssao = s.ao && s.post && !w.post.noSSAO
	w.post.aoSlices = s.aoSlices
	w.shadow.cascades = s.cascades
	if w.post.noMSAA {
		w.post.msaa = 0
	} else {
		w.post.msaa = s.msaa
	}
	if w.post.ssao {
		if w.post.ssaoRadius <= 0 {
			w.post.ssaoRadius = 0.8
		}
		if w.post.ssaoIntensity <= 0 {
			w.post.ssaoIntensity = 1.8
		}
		if w.post.ssaoBias <= 0 {
			w.post.ssaoBias = 0.02
		}
	}
	// PCF <-> PCSS only; a moment filter the script chose stays.
	if w.shadow.filter <= 1 {
		w.shadow.filter = 0
		if s.softShadow {
			w.shadow.filter = 1
		}
	}
	for _, t := range w.terrains {
		w.applyTerrainQuality(t, prevScatter != s.scatter)
	}
}

func (w *World) applyTerrainQuality(t *terrain, rescatter bool) {
	if t == nil || !w.quality.set {
		return
	}
	s := tiers[w.quality.tier]
	changed := t.radius != s.radius || t.detailPx != s.detailPx
	t.radius = s.radius
	t.detailPx = s.detailPx
	if changed {
		if rescatter {
			w.releaseMeshTrees(t)
			w.resetGrass(t)
		}
		w.dirtyTerrain(t)
		return
	}
	if rescatter {
		w.refreshScatter(t)
	}
}

// bloomAllowed is false on the low tier, where the blur chain costs too much.
func (w *World) bloomAllowed() bool {
	return !w.quality.set || w.quality.tier > qualityLow
}

// scatterScale thins trees, props, and grass on lower tiers.
func (w *World) scatterScale() float32 {
	if !w.quality.set {
		return 1
	}
	return tiers[w.quality.tier].scatter
}

// waterTargets is the reflection and refraction size; 0 skips that pass.
func (w *World) waterTargets() (reflect, refract int) {
	if !w.quality.set {
		return 1024, 1024
	}
	s := tiers[w.quality.tier]
	return s.reflect, s.refract
}

// defaultQuality gives a script that never calls SetGraphicsQuality the auto
// tier, so tonemap, bloom, AO, and AA are on without asking. It runs inside
// Graphics3D, before the script's own setup, so anything the script sets
// afterwards still wins.
func (w *World) defaultQuality() {
	if w.quality.set {
		return
	}
	tier := autoTier(w.glmod.caps.renderer)
	w.quality.auto = true
	w.quality.warm, w.quality.acc, w.quality.frames = 0, 0, 0
	w.setQuality(tier)
	fmt.Printf("quality: %s (default auto, %s)\n", qualityNames[tier], w.glmod.caps.renderer)
}

// tickQuality steps an auto tier down while frames stay slow. It never
// steps up, so a busy moment cannot make the picture flicker between tiers.
func (w *World) tickQuality() {
	q := &w.quality
	if !q.auto || q.tier == qualityLow || w.delta <= 0 {
		return
	}
	dt := w.delta
	if dt > 0.25 {
		dt = 0.25 // a load stall is not a frame rate
	}
	if q.warm < 3 {
		q.warm += dt
		return
	}
	q.acc += dt
	q.frames++
	if q.acc < 2 {
		return
	}
	avg := q.acc / float64(q.frames)
	q.acc, q.frames = 0, 0
	if avg > 0.026 {
		// Resolution gives way before effects.
		if w.dynActive() && !w.dyn.atFloor() {
			return
		}
		w.setQuality(q.tier - 1)
		q.drops++
		q.warm = 1 // the rebuild after a change is a stall too
		fmt.Printf("quality: %s (auto, %.0f ms frames)\n", qualityNames[q.tier], avg*1000)
	}
}

// applyLook sets a whole picture in one call, using the same commands a script
// would. It adds what the scene lacks (a sky, a sun) and leaves what it has.
// Anything set afterwards still wins.
func (w *World) applyLook(name string) error {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "off":
		w.post.on = false
		w.post.userOn = true
		w.post.ssao = false
		w.iblOn = false
		w.fogMode = 0
		w.linearLight = false
		if w.ready && !w.headless {
			_, err := w.Call("enableshadows", []value.Value{value.Num(0)})
			return err
		}
		return nil
	case "modern", "":
	default:
		return fmt.Errorf("SetLook: want modern or off, got %q", name)
	}
	w.iblOn = true
	if w.headless {
		return nil // nothing to draw
	}
	if err := w.require3D(); err != nil {
		return err
	}
	// Low too: it keeps the post pass, and an old machine should see the same picture
	// with less in it, not a different one.
	if !w.quality.set || tiers[w.quality.tier].post {
		if _, err := w.Call("enablepostfx", []value.Value{value.Num(1)}); err != nil {
			return err
		}
		w.linearLight = true
		w.post.tonemap = 2 // ACES: the film curve linear light was made for
		w.post.exposure = 0.8
		if (!w.quality.set || tiers[w.quality.tier].ao) && !w.post.noSSAO {
			if _, err := w.Call("enablessao", []value.Value{value.Num(1)}); err != nil {
				return err
			}
		}
	}
	if w.skyID == 0 {
		if _, err := w.makeSkyBox("default"); err != nil {
			return err
		}
	}
	sun := 0
	for id, e := range w.ents {
		if e != nil && e.lgtKind == 1 && (sun == 0 || id < sun) {
			sun = id
		}
	}
	if sun == 0 {
		v, err := w.Call("createdirectionallight", nil)
		if err != nil {
			return err
		}
		sun = int(v.Num)
		if _, err := w.Call("setlightdirection", []value.Value{value.Num(float64(sun)), value.Num(45), value.Num(40), value.Num(0)}); err != nil {
			return err
		}
		if _, err := w.Call("setlightcolor", []value.Value{value.Num(float64(sun)), value.Num(255), value.Num(238), value.Num(212)}); err != nil {
			return err
		}
	}
	if _, err := w.Call("setlightshadow", []value.Value{value.Num(float64(sun)), value.Num(1)}); err != nil {
		return err
	}
	if _, err := w.Call("enableshadows", []value.Value{value.Num(1)}); err != nil {
		return err
	}
	// Distance melts into the sky's own horizon, so the world has no hard edge.
	if w.fogMode == 0 {
		far := 1000.0
		if w.cam != nil && w.cam.Far() > 0 {
			far = float64(w.cam.Far())
		}
		r, g, b := w.envColor(1, 0.02, 0)
		w.setFog(float64(r)*255, float64(g)*255, float64(b)*255, far*0.35, far*0.95)
	}
	return nil
}

func (w *World) qualityCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// SetLook name$: "modern" turns on linear lighting, tone mapping, bloom,
		// FXAA, ambient occlusion, image-based light, sun shadows and horizon fog, and adds
		// a sky and a sun if the scene has none. "off" turns them off.
		"setlook": n(func(a []value.Value) (value.Value, error) {
			if err := w.applyLook(argS(a, 0)); err != nil {
				return value.Value{}, err
			}
			return z()
		}),
		"setgraphicsquality": n(func(a []value.Value) (value.Value, error) {
			tier, auto := parseQuality(argS(a, 0))
			if len(a) == 0 {
				auto = true
			}
			if auto {
				w.detectModernGL()
				tier = autoTier(w.glmod.caps.renderer)
			}
			w.quality.auto = auto
			w.quality.warm, w.quality.acc, w.quality.frames = 0, 0, 0
			w.setQuality(tier)
			how := "set"
			if auto {
				how = "auto, " + w.glmod.caps.renderer
			}
			fmt.Printf("quality: %s (%s)\n", qualityNames[tier], how)
			return value.Num(float64(tier)), nil
		}),
		"graphicsquality": n(func(a []value.Value) (value.Value, error) {
			if !w.quality.set {
				return value.Num(qualityHigh), nil
			}
			return value.Num(float64(w.quality.tier)), nil
		}),
	}
}
