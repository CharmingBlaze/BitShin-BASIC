package runtime

import (
	"strings"
	"testing"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

func TestCommandTableShadows(t *testing.T) {
	m := New(".").commandTable()
	for _, name := range []string{
		"enableshadows", "shadowcascades", "shadowmapsize",
		"setshadowbias", "setshadowpcf", "setshadowfilter", "setshadowpcss",
		"setshadowevsm", "setshadowmsm", "setshadowquality", "enableshadowcache",
		"enableshadowcaching", "enableshadowatlas",
		"enablecontactshadows", "enablescreenspaceshadows", "setsss",
		"createdirectionallight", "setlightdirection", "setlightshadow",
	} {
		if m[name] == nil {
			t.Errorf("missing command %s", name)
		}
	}
	if m["setlightdirection"] == nil {
		t.Fatal("setlightdirection missing")
	}
}

func TestCascadeSplitsIncrease(t *testing.T) {
	w := New(".")
	cam := camera.New(16.0 / 9.0)
	cam.SetNear(0.1)
	cam.SetFar(400)
	w.computeCascadeSplits(cam, 3)
	if !(w.shadow.splits[0] < w.shadow.splits[1] && w.shadow.splits[1] < w.shadow.splits[2]) {
		t.Fatalf("splits not increasing: %v", w.shadow.splits)
	}
	if w.shadow.splits[2] > 80.1 {
		t.Fatalf("far split should clamp around 80, got %v", w.shadow.splits[2])
	}
}

func lightVPContains(m *math32.Matrix4, p math32.Vector3) bool {
	v := math32.Vector4{p.X, p.Y, p.Z, 1}
	v.ApplyMatrix4(m)
	if math32.Abs(v.W) > 1e-6 {
		v.X /= v.W
		v.Y /= v.W
		v.Z /= v.W
	}
	const pad = 0.98
	return v.X >= -pad && v.X <= pad && v.Y >= -pad && v.Y <= pad && v.Z >= -pad && v.Z <= pad
}

func TestCascadeViewsCoverPlaySpaceWhenLookingUp(t *testing.T) {
	w := New(".")
	w.shadow.cascades = 2
	w.shadow.size = 2048
	cam := camera.New(16.0 / 9.0)
	cam.SetNear(0.1)
	cam.SetFar(400)
	cam.SetPosition(0, 14, -8)
	up := math32.Vector3{0, 0, 1}
	cam.LookAt(&math32.Vector3{0, 80, -8}, &up)
	cam.UpdateMatrixWorld()
	w.computeCascadeSplits(cam, 2)
	w.buildCascadeViews(cam, 2)
	pts := []math32.Vector3{
		{0, 0, 0},
		{4, 4.4, 20},
		{16, 2.6, 12},
		{-12, 0.35, 4},
		{0, 0.7, -4},
	}
	for _, p := range pts {
		if !lightVPContains(&w.shadow.lightVP[0], p) && !lightVPContains(&w.shadow.lightVP[1], p) {
			t.Fatalf("play-space point %v fell outside every cascade while looking up", p)
		}
	}
	if w.shadow.lightVP[0] == (math32.Matrix4{}) {
		t.Fatal("cascade 0 VP empty")
	}
}

func TestAtlasGrid(t *testing.T) {
	c, r := atlasGrid(2, true)
	if c != 2 || r != 1 {
		t.Fatalf("strip 2 tiles: %d x %d", c, r)
	}
	c, r = atlasGrid(17, false)
	if c*r < 17 {
		t.Fatalf("packed 17 tiles: %d x %d", c, r)
	}
}

func TestShadowCommandsDoWork(t *testing.T) {
	w := New(".")
	must := func(name string, args ...float64) {
		t.Helper()
		av := make([]value.Value, len(args))
		for i, n := range args {
			av[i] = value.Num(n)
		}
		if _, err := w.Call(name, av); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	must("setshadowevsm", 1)
	if w.shadow.filter != shadowFilterEVSM || !w.shadow.on {
		t.Fatalf("EVSM filter=%d on=%v", w.shadow.filter, w.shadow.on)
	}
	must("setshadowmsm", 1)
	if w.shadow.filter != shadowFilterMSM {
		t.Fatalf("MSM filter=%d", w.shadow.filter)
	}
	must("enableshadowcache", 1)
	must("setshadowcache", 1)
	must("enableshadowcaching", 1)
	if !w.shadow.cache {
		t.Fatal("cache not enabled")
	}
	must("setshadowquality", 2, 5)
	if w.shadow.filter != shadowFilterEVSM || w.shadow.pcf != 5 {
		t.Fatalf("quality evsm pcf=%d filter=%d", w.shadow.pcf, w.shadow.filter)
	}
	must("setshadowbias", 0.003, 1.5)
	if w.shadow.bias != 0.003 || w.shadow.normalBias != 1.5 {
		t.Fatalf("bias %v normal %v", w.shadow.bias, w.shadow.normalBias)
	}
	must("enableshadowatlas", 1)
	must("setshadowatlas", 1)
	if !w.shadow.atlas {
		t.Fatal("atlas not enabled")
	}
	must("enablecontactshadows", 1)
	must("setcontactshadows", 1)
	if !w.shadow.contact {
		t.Fatal("contact not enabled")
	}
	must("enablescreenspaceshadows", 1)
	must("setscreenspaceshadows", 1)
	if w.shadow.sss == 0 {
		t.Fatal("screen-space not enabled")
	}
	must("setsss", 2)
	if w.shadow.sss != 2 {
		t.Fatalf("setsss quality=%d", w.shadow.sss)
	}
	if _, err := w.Call("setshadowfilter", []value.Value{value.Str("evsm")}); err != nil {
		t.Fatal(err)
	}
	if w.shadow.filter != shadowFilterEVSM {
		t.Fatalf("filter evsm -> %d", w.shadow.filter)
	}
}

func TestShadowCacheKeyChanges(t *testing.T) {
	w := New(".")
	w.shadow.cascades = 2
	w.shadow.lightVP[0][0] = 1
	a := w.shadowSceneKey()
	w.shadow.lightVP[0][0] = 2
	b := w.shadowSceneKey()
	if a == b {
		t.Fatal("cache key should change when light VP changes")
	}
	w.shadow.cache = true
	w.shadow.ready = true
	w.shadow.cacheKey = b
	if w.shadowSceneKey() != w.shadow.cacheKey {
		t.Fatal("full scene key should match after assign")
	}
}

func TestShadowCacheInvalidatesDeformers(t *testing.T) {
	w := New(".")
	w.shadow.cache = true
	w.shadow.ready = true
	w.shadow.cacheKey = w.shadowSceneKey()
	if w.shadowCacheStale() {
		t.Fatal("empty world should not be cache-stale")
	}
	w.MarkShadowDirty()
	if !w.shadowCacheStale() {
		t.Fatal("MarkShadowDirty should skip cache")
	}
	w.clearShadowDirty()
	w.ents[1] = &Entity{anim: &animState{mode: 1}}
	if !w.shadowCacheStale() {
		t.Fatal("playing animation should skip cache")
	}
	w.ents[1].anim.mode = 0
	w.ents[1].shadowGeomDirty = true
	if !w.shadowCacheStale() {
		t.Fatal("dirty geometry should skip cache")
	}
	w.ents[1].shadowGeomDirty = false
	w.emitters[1] = &emitter{visible: true, parts: []particle{{life: 1}}}
	if w.shadowCacheStale() {
		t.Fatal("unlit 3D particles must not skip shadow cache")
	}
}

func TestShadowShaderHasTechniques(t *testing.T) {
	for _, needle := range []string{"evsmAt", "msmAt", "contactAt", "sssAt", "localShadows", "PointVP", "SpotVP", "tileUV", "WorldNormal", "casAt", "smoothstep", "ShadowNormalBias", "ignoise", "gradientNoise", "ShadowMapDyn", "MomentMode"} {
		src := mbshadowFragment
		if needle == "MomentMode" {
			src = depthFragmentSrc
		}
		if !strings.Contains(src, needle) {
			t.Fatalf("shadow GLSL missing %s", needle)
		}
	}
	if !strings.Contains(mbshadowVertex, "WorldNormal") {
		t.Fatal("mbshadow vertex missing WorldNormal")
	}
	if strings.Contains(mbshadowSampleGLSL, "for (int x = -2; x <= 2; x++)") {
		t.Fatal("EVSM/MSM lighting still 5x5 samples depth")
	}
	if strings.Contains(mbshadowFragment, "for (int x = -8; x <= 8; x++)") {
		t.Fatal("PCF still uses -8..8 continue loop")
	}
	if !strings.Contains(mbshadowSampleGLSL, "for (int x = -r; x <= r; x++)") {
		t.Fatal("pcfAt missing unrotated grid PCF")
	}
	if strings.Contains(mbshadowSampleGLSL, "GOLDEN_ANGLE") {
		t.Fatal("pcfAt still uses Vogel disk (sandy without TAA)")
	}
	if !strings.Contains(mbshadowSampleGLSL, "ShadowSplit.x * 0.35") || !strings.Contains(mbshadowSampleGLSL, "6.0") {
		t.Fatal("cascade 0 blend band not widened")
	}
	if !strings.Contains(mbshadowFragment, "texel.x * 2.5") || !strings.Contains(mbphysicalFragment, "texel.x * 2.5") {
		t.Fatal("normal offset WorldPos + WorldNormal * (texel.x * 2.5) missing")
	}
	if !strings.Contains(mbshadowVertex, "wpShadow") || !strings.Contains(mbphysicalVertex, "wpShadow") {
		t.Fatal("LightSpacePos must use normal-offset world position")
	}
	if strings.Contains(mbshadowFragment, "not yet") {
		t.Fatal("shader still mentions not yet")
	}
}

func TestShadowStaticVsDynamicKey(t *testing.T) {
	w := New(".")
	w.shadow.cascades = 2
	w.shadow.lightVP[0][0] = 1
	a := w.shadowStaticKey()
	w.shadow.lightVP[0][0] = 2
	b := w.shadowStaticKey()
	if a == b {
		t.Fatal("static key should change when sun VP changes")
	}
}

func TestShadowCasterStaticClass(t *testing.T) {
	if shadowCasterStatic(nil) || shadowCasterStatic(&Entity{name: "terrain"}) {
		t.Fatal("nil / mesh-less should not be static casters")
	}
	mesh := &graphic.Mesh{}
	if !shadowCasterStatic(&Entity{name: "terrain", mesh: mesh}) {
		t.Fatal("terrain mesh should be static")
	}
	if shadowCasterStatic(&Entity{name: "player", mesh: mesh}) {
		t.Fatal("moving entity should be dynamic")
	}
	if !shadowCasterStatic(&Entity{name: "box", mesh: mesh, bodyType: 2}) {
		t.Fatal("static physics body should be static")
	}
}
