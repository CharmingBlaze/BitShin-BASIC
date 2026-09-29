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
		"setshadowsoftness", "shadowsoftness", "setshadowcolor", "shadowcolor",
		"setshadowfade", "setlightspecular",
		"createdirectionallight", "setlightdirection", "setlightshadow",
		"entitycastshadow", "entityreceiveshadow", "setshadowdistance",
	} {
		if m[name] == nil {
			t.Errorf("missing command %s", name)
		}
	}
	if m["setlightdirection"] == nil {
		t.Fatal("setlightdirection missing")
	}
}

func TestShadowsAreOnByDefault(t *testing.T) {
	w := New(".")
	if !w.shadow.on {
		t.Fatal("new 3D worlds must have shadows enabled by default")
	}
	if got := w.litShaderName(); got != "bsshadow" {
		t.Fatalf("default lit shader = %q, want bsshadow", got)
	}

	w.shadow.on = false
	w.ensureShadowOn()
	if !w.shadow.on {
		t.Fatal("Graphics3D shadow initialization must restore the default")
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
	if w.shadow.splits[2] < 200 || w.shadow.splits[2] > 301 {
		t.Fatalf("far split should track shadow distance ~250, got %v", w.shadow.splits[2])
	}
}

func TestCascadeSplitPracticalMix(t *testing.T) {
	s0 := cascadeSplit(1, 4, 0.1, 250, 0.70)
	s1 := cascadeSplit(2, 4, 0.1, 250, 0.70)
	s2 := cascadeSplit(3, 4, 0.1, 250, 0.70)
	s3 := cascadeSplit(4, 4, 0.1, 250, 0.70)
	if !(s0 < s1 && s1 < s2 && s2 < s3) {
		t.Fatalf("practical splits not increasing: %v %v %v %v", s0, s1, s2, s3)
	}
	if s0 < 8 || s0 > 30 {
		t.Fatalf("first split should be near the 12m starting point, got %v", s0)
	}
	if s3 < 249 || s3 > 251 {
		t.Fatalf("last split should be shadow far, got %v", s3)
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

func TestCascade0CoversFollowCameraPlayer(t *testing.T) {
	w := New(".")
	w.shadow.cascades = 4
	w.shadow.size = 2048
	w.shadow.distance = 250
	id := w.makeLight(1, 0)
	w.ents[id].castShadow = true
	w.shadow.lightID = id
	w.setLightDir(w.ents[id], 55, 40, 0)

	cam := camera.New(16.0 / 9.0)
	cam.SetNear(0.1)
	cam.SetFar(4000)
	// Platform 64 follow: player Blitz (0, 0.5, -4) → G3N (0, 0.5, 4).
	cam.SetPosition(0, 4.15, 12)
	up := math32.Vector3{0, 1, 0}
	cam.LookAt(&math32.Vector3{0, 1.65, 4}, &up)
	cam.UpdateMatrixWorld()
	w.computeCascadeSplits(cam, 4)
	w.buildCascadeViews(cam, 4)

	player := math32.Vector3{0, 0.7, 4}
	if !lightVPContains(&w.shadow.lightVP[0], player) {
		t.Fatalf("follow-cam player %v must sit in cascade 0, splits=%v texel0=%v", player, w.shadow.splits, w.shadow.texelWorld[0])
	}
	if w.shadow.texelWorld[0] > 0.06 {
		t.Fatalf("cascade 0 texels too coarse (%v); shadow will blob", w.shadow.texelWorld[0])
	}
	ring := math32.Vector3{-16, 3.2, 10}
	coin := math32.Vector3{0, 1.2, 6}
	if !lightVPContains(&w.shadow.lightVP[0], coin) && !lightVPContains(&w.shadow.lightVP[1], coin) && !lightVPContains(&w.shadow.lightVP[2], coin) {
		t.Fatalf("coin %v outside every cascade", coin)
	}
	if !lightVPContains(&w.shadow.lightVP[0], ring) && !lightVPContains(&w.shadow.lightVP[1], ring) && !lightVPContains(&w.shadow.lightVP[2], ring) {
		t.Fatalf("ring %v outside every cascade", ring)
	}
	dir := w.shadowLightDir()
	wantX, wantY, wantZ := dirLightOffset(55, 40)
	if abs32(dir.X-wantX) > 0.02 || abs32(dir.Y-wantY) > 0.02 || abs32(dir.Z-wantZ) > 0.02 {
		t.Fatalf("shadow sun %v != light position %v %v %v", dir, wantX, wantY, wantZ)
	}
}

func TestCascadeViewsFitCameraFrustum(t *testing.T) {
	w := New(".")
	w.shadow.cascades = 4
	w.shadow.size = 2048
	w.shadow.distance = 250
	id := w.makeLight(1, 0)
	w.ents[id].castShadow = true
	w.shadow.lightID = id
	w.setLightDir(w.ents[id], 55, 40, 0)

	cam := camera.New(16.0 / 9.0)
	cam.SetNear(0.1)
	cam.SetFar(4000)
	cam.SetPosition(0, 2, 0)
	up := math32.Vector3{0, 1, 0}
	cam.LookAt(&math32.Vector3{0, 2, 20}, &up)
	cam.UpdateMatrixWorld()
	w.computeCascadeSplits(cam, 4)
	w.buildCascadeViews(cam, 4)

	pos, fwd, right, camUp := cameraViewBasis(cam)
	corners := frustumSliceCorners(pos, fwd, right, camUp, cam.Fov(), cam.Aspect(), w.shadowNear(cam), w.shadow.splits[0])
	for i, corner := range corners {
		clip := math32.Vector4{corner.X, corner.Y, corner.Z, 1}
		clip.ApplyMatrix4(&w.shadow.lightVP[0])
		if math32.Abs(clip.W) > 1e-6 {
			clip.X /= clip.W
			clip.Y /= clip.W
		}
		if math32.Abs(clip.X) > 0.90 || math32.Abs(clip.Y) > 0.90 {
			t.Fatalf("cascade guard missing at corner %d: clip=(%v,%v)", i, clip.X, clip.Y)
		}
	}

	alongRay := math32.Vector3{0, 2, 10}
	if !lightVPContains(&w.shadow.lightVP[0], alongRay) && !lightVPContains(&w.shadow.lightVP[1], alongRay) {
		t.Fatalf("frustum point along look %v must be in a near cascade, splits=%v", alongRay, w.shadow.splits)
	}
	side := math32.Vector3{400, 0, 0}
	if lightVPContains(&w.shadow.lightVP[0], side) {
		t.Fatalf("distant side point %v must not sit in cascade 0", side)
	}
}

func TestCascadeViewsCoverPlaySpaceWhenLookingUp(t *testing.T) {
	w := New(".")
	w.shadow.cascades = 2
	w.shadow.size = 2048
	w.shadow.distance = 250
	cam := camera.New(16.0 / 9.0)
	cam.SetNear(0.1)
	cam.SetFar(400)
	cam.SetPosition(0, 14, -8)
	up := math32.Vector3{0, 0, 1}
	cam.LookAt(&math32.Vector3{0, 80, -8}, &up)
	cam.UpdateMatrixWorld()
	w.computeCascadeSplits(cam, 2)
	w.buildCascadeViews(cam, 2)
	alongLook := math32.Vector3{0, 40, -8}
	if !lightVPContains(&w.shadow.lightVP[0], alongLook) && !lightVPContains(&w.shadow.lightVP[1], alongLook) {
		t.Fatalf("look-ray point %v must sit in a cascade when looking up, splits=%v", alongLook, w.shadow.splits)
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
	if _, err := w.Call("setshadowquality", []value.Value{value.Str("high")}); err != nil {
		t.Fatal(err)
	}
	if w.shadow.filter != shadowFilterPCF || w.shadow.cascades != 4 || w.shadow.size != 2048 {
		t.Fatalf("high preset filter=%d cas=%d size=%d", w.shadow.filter, w.shadow.cascades, w.shadow.size)
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
	must("setshadowsoftness", 1.8)
	if w.shadow.softness != 1.8 {
		t.Fatalf("softness %v", w.shadow.softness)
	}
	must("setshadowcolor", 20, 30, 50)
	if w.shadow.color.R < 0.05 {
		t.Fatalf("color %v", w.shadow.color)
	}
	must("setshadowfade", 50, 90)
	if w.shadow.fadeNear != 50 || w.shadow.fadeFar != 90 {
		t.Fatalf("fade %v..%v", w.shadow.fadeNear, w.shadow.fadeFar)
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
	for _, needle := range []string{"evsmAt", "msmAt", "contactAt", "sssAt", "localShadows", "PointVP", "SpotVP", "tileUV", "WorldNormal", "casAt", "smoothstep", "ShadowNormalBias", "ignoise", "gradientNoise", "ShadowMapDyn", "MomentMode", "POISSON_DISK", "sunShadowFactor", "pointShadowFactor", "spotShadowFactor", "ShadowSoftness", "ShadowColor", "calcDynamicBias", "pcf3x3", "MeshReceiveShadow", "emptyDepth"} {
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
	if !strings.Contains(mbshadowSampleGLSL, "POISSON_DISK") {
		t.Fatal("pcfAt missing Poisson Disk kernel")
	}
	if !strings.Contains(mbshadowSampleGLSL, "(farS - prev) * 0.10") {
		t.Fatal("cascade blend band must be last 10%")
	}
	if !strings.Contains(mbshadowSampleGLSL, "sh = min(sh, sh0)") || !strings.Contains(mbshadowSampleGLSL, "sh = min(sh, sh2)") {
		t.Fatal("cascade overlap must conservatively preserve shadows on both sides of a split")
	}
	if !strings.Contains(mbshadowVertex, "wpShadow") || !strings.Contains(mbphysicalVertex, "wpShadow") {
		t.Fatal("LightSpacePos must use normal-offset world position")
	}
	if !strings.Contains(mbshadowSampleGLSL, "sampleDepth(ShadowMap, clamp(uv + o") {
		t.Fatal("contact and screen-space shadows must sample nearby shadow-map texels")
	}
	if strings.Contains(mbshadowSampleGLSL, "float contactAt(vec2 uv, float z, vec2 texel) {\n    return 1.0;\n}") {
		t.Fatal("contactAt is still a fully-lit stub")
	}
	if strings.Contains(mbshadowFragment, "not yet") {
		t.Fatal("shader still mentions not yet")
	}
	if !strings.Contains(mbshadowSampleGLSL, "m1 * exp(c *") {
		t.Fatal("EVSM must bias in exp/depth space so ground self-shadow does not crush lighting")
	}
	if !strings.Contains(mbshadowSampleGLSL, "if (m1 < 0.0001)") {
		t.Fatal("failed EVSM moments must be lit")
	}
	if !strings.Contains(mbshadowSampleGLSL, "if (inside < 0.5) { return 1.0; }") {
		t.Fatal("out-of-cascade samples must be lit")
	}
	if !strings.Contains(mbshadowSampleGLSL, "dFdx(proj.z)") {
		t.Fatal("PCF must use screen-space slope bias so huge ground tris are not a solid umbra")
	}
	if !strings.Contains(mbshadowSampleGLSL, "receiverDepthBias") {
		t.Fatal("casAt must use receiverDepthBias")
	}
	if !strings.Contains(depthEmptyFragmentSrc, "void main() {}") {
		t.Fatal("PCF depth pass must be empty/minimal fragment")
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

func TestShadowsOnByDefault(t *testing.T) {
	w := New(".")
	if !w.shadow.on {
		t.Fatal("shadows must default on like Blitz3D")
	}
}

func TestEnsureShadowOnDefaultsPCF(t *testing.T) {
	w := New(".")
	w.ensureShadowOn()
	if !w.shadow.on {
		t.Fatal("ensureShadowOn should enable the pass")
	}
	if w.shadow.filter != shadowFilterPCF || w.shadow.cascades < 1 || w.shadow.size < 256 {
		t.Fatalf("default quality filter=%d cas=%d size=%d", w.shadow.filter, w.shadow.cascades, w.shadow.size)
	}
}

func TestShadowSampleNeedsWarmCascades(t *testing.T) {
	w := New(".")
	w.ensureShadowOn()
	w.shadow.ready = true
	w.shadow.tex = 1
	w.shadow.texelWorld[0] = 0.04
	w.shadow.lightVP[0][0] = 1
	w.shadow.lightVP[0][5] = 1
	w.shadow.lightVP[0][10] = 1
	w.makeLight(1, 0)
	if w.shadowSampleOK() {
		t.Fatal("first frames must stay lit until cascades are warm")
	}
	w.loopFrames = 2
	w.shadow.warm = shadowWarmupFrames - 1
	if w.shadowSampleOK() {
		t.Fatal("partially warmed cascades must remain hidden")
	}
	w.shadow.warm = shadowWarmupFrames
	if !w.shadowSampleOK() {
		t.Fatal("valid warm cascades should sample")
	}
	w.shadow.texelWorld[0] = 8
	if w.shadowSampleOK() {
		t.Fatal("planet-sized cascade 0 must not stamp an umbra")
	}
	w.shadow.texelWorld[0] = 0.04
	w.loopFrames = 1
	if w.shadowSampleOK() {
		t.Fatal("Flip 1 (no user loop / CameraFollow) must stay lit")
	}
}

func TestCameraFollowSnapsFirstFrame(t *testing.T) {
	w := New(".")
	camID := w.spawnCamera(0)
	tgt := w.createCubeMesh(nil)
	gx, gy, gz := toG3N(0, 0.5, -4)
	w.ents[tgt].node.GetNode().SetPosition(gx, gy, gz)
	camN := w.ents[camID].node.GetNode()
	camN.SetPosition(0, 40, 40)
	w.delta = 0.016
	if _, err := w.cameraFollow([]value.Value{
		value.Num(float64(camID)),
		value.Num(float64(tgt)),
		value.Num(8.2),
		value.Num(3.05),
		value.Num(8.5),
		value.Num(0),
		value.Num(12),
	}); err != nil {
		t.Fatal(err)
	}
	p := camN.Position()
	if p.Y > 12 {
		t.Fatalf("first CameraFollow must snap to the orbit point, pos=%v", p)
	}
	if !w.ents[camID].camFollowed {
		t.Fatal("camFollowed")
	}
}
