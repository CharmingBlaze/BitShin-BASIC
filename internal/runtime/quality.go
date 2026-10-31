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
}

var tiers = [...]tierSettings{
	qualityLow:    {shadow: 1024, reflect: 0, refract: 0, post: false, radius: 1, detailPx: 14, scatter: 0.45, treeReachM: 220},
	qualityMedium: {shadow: 1024, reflect: 512, refract: 512, post: true, radius: 2, detailPx: 10, scatter: 0.7, treeReachM: 320},
	qualityHigh:   {shadow: 2048, reflect: 1024, refract: 1024, post: true, radius: 2, detailPx: 8, scatter: 1, treeReachM: 420, ao: true},
	qualityUltra:  {shadow: 2048, reflect: 2048, refract: 1024, post: true, radius: 3, detailPx: 6, scatter: 1, treeReachM: 480, ao: true, softShadow: true, msaa: 4},
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
// watch corrects a wrong guess downward within a few seconds.
func autoTier(renderer string) int {
	r := strings.ToLower(renderer)
	switch {
	case r == "", strings.Contains(r, "llvmpipe"), strings.Contains(r, "softpipe"),
		strings.Contains(r, "basic render"), strings.Contains(r, "swiftshader"):
		return qualityLow
	case strings.Contains(r, "intel") && (strings.Contains(r, "hd graphics") || strings.Contains(r, "gma")):
		return qualityLow
	case strings.Contains(r, "intel"), strings.Contains(r, "mali"), strings.Contains(r, "adreno"):
		return qualityMedium
	}
	return qualityHigh
}

func parseQuality(s string) (tier int, auto bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low", "0":
		return qualityLow, false
	case "medium", "med", "1":
		return qualityMedium, false
	case "high", "2":
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
			if w.post.bloom == 0 {
				w.post.bloom = 0.22
			}
			if w.post.contrast <= 0 {
				w.post.contrast = 1
			}
			if w.post.sat <= 0 {
				w.post.sat = 1
			}
			if w.post.tint.R+w.post.tint.G+w.post.tint.B == 0 {
				w.post.tint.R, w.post.tint.G, w.post.tint.B = 1, 1, 1
			}
			w.post.fxaa = true
		}
	}
	w.post.ssao = s.ao && s.post
	w.post.msaa = s.msaa
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
		}
		w.dirtyTerrain(t)
		return
	}
	if rescatter {
		w.refreshScatter(t)
	}
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
		w.setQuality(q.tier - 1)
		q.drops++
		q.warm = 1 // the rebuild after a change is a stall too
		fmt.Printf("quality: %s (auto, %.0f ms frames)\n", qualityNames[q.tier], avg*1000)
	}
}

func (w *World) qualityCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
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
