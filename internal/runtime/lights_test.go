package runtime

import (
	"testing"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

func TestDirLightOffsetAbove(t *testing.T) {
	_, gy, _ := dirLightOffset(50, 35)
	if gy <= 0.3 {
		t.Fatalf("positive pitch should place the sun above, y=%v", gy)
	}
	_, gy2, _ := dirLightOffset(-45, 30)
	if gy2 >= 0 {
		t.Fatalf("negative pitch should place the sun below, y=%v", gy2)
	}
}

func TestCreateLightKinds(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()

	dir := w.makeLight(1, 0)
	if w.ents[dir].lgtKind != 1 {
		t.Fatalf("CreateLight(1) kind=%d", w.ents[dir].lgtKind)
	}
	if _, ok := w.ents[dir].node.(*light.Directional); !ok {
		t.Fatal("CreateLight(1) must be directional")
	}
	p := w.ents[dir].node.GetNode().Position()
	if p.Y <= 0.2 {
		t.Fatalf("default directional sun should be above, pos=%v", p)
	}

	pt := w.makeLight(2, 0)
	if _, ok := w.ents[pt].node.(*light.Point); !ok || w.ents[pt].lgtKind != 2 {
		t.Fatal("CreateLight(2) must be point")
	}
	sp := w.makeLight(3, 0)
	if _, ok := w.ents[sp].node.(*light.Spot); !ok || w.ents[sp].lgtKind != 3 {
		t.Fatal("CreateLight(3) must be spot")
	}
}

func TestSetLightDirectionKeepsSpotPosition(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	id := w.makeLight(3, 0)
	e := w.ents[id]
	gx, gy, gz := toG3N(-3, 4, 1)
	e.node.GetNode().SetPosition(gx, gy, gz)
	w.setLightDir(e, -55, 20, 0)
	got := e.node.GetNode().Position()
	if abs32(got.X-gx) > 1e-4 || abs32(got.Y-gy) > 1e-4 || abs32(got.Z-gz) > 1e-4 {
		t.Fatalf("spot SetLightDirection must not move the lamp, got %v want %v", got, math32.Vector3{gx, gy, gz})
	}
}

func TestRotateEntityAimsDirectional(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	id := w.makeLight(1, 0)
	e := w.ents[id]
	e.pitch, e.yaw, e.roll = 50, 35, 0
	w.applyRot(e)
	wantX, wantY, wantZ := dirLightOffset(50, 35)
	got := e.node.GetNode().Position()
	if abs32(got.X-wantX) > 1e-4 || abs32(got.Y-wantY) > 1e-4 || abs32(got.Z-wantZ) > 1e-4 {
		t.Fatalf("RotateEntity on a directional light must set G3N position, got %v", got)
	}
}

func TestHasShadowLight(t *testing.T) {
	w := New(".")
	if w.hasShadowLight() {
		t.Fatal("empty world has no shadow light")
	}
	w.scene = core.NewNode()
	w.makeLight(2, 0)
	if w.hasShadowLight() {
		t.Fatal("point light without SetLightShadow must not enable CSM")
	}
	dir := w.makeLight(1, 0)
	if !w.hasShadowLight() {
		t.Fatal("directional light should drive shadows")
	}
	if !w.ents[dir].castShadow {
		t.Fatal("CreateDirectionalLight / CreateLight(1) must cast by default")
	}
	if w.shadow.lightID != dir {
		t.Fatalf("first directional should be the CSM sun, lightID=%d dir=%d", w.shadow.lightID, dir)
	}
	delete(w.ents, dir)
	pt := w.makeLight(2, 0)
	w.ents[pt].castShadow = true
	if !w.hasShadowLight() {
		t.Fatal("shadow-casting point light should count")
	}
	pt2 := w.makeLight(2, 0)
	if w.ents[pt2].castShadow {
		t.Fatal("point lights stay opt-in unless SetLightShadow")
	}
}

func TestLocalLightRangeIsWindowed(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	pt := w.makeLight(2, 0)
	p := w.ents[pt].node.(*light.Point)
	if p.QuadraticDecay() != 0 {
		t.Fatalf("point lights use the windowed curve, quadratic=%v", p.QuadraticDecay())
	}
	if abs32(p.LinearDecay()-1.0/14.0) > 1e-4 {
		t.Fatalf("default point range 14, linear=%v", p.LinearDecay())
	}
	applyLightRange(p, 12)
	if abs32(p.LinearDecay()-1.0/12.0) > 1e-4 || p.QuadraticDecay() != 0 {
		t.Fatalf("LightRange 12 should be 1/12 with no quadratic, lin=%v quad=%v", p.LinearDecay(), p.QuadraticDecay())
	}
	if abs32(entityLightRange(w.ents[pt])-12) > 1e-3 {
		t.Fatalf("shadow range should match LightRange, got %v", entityLightRange(w.ents[pt]))
	}

	sp := w.makeLight(3, 0)
	s := w.ents[sp].node.(*light.Spot)
	if s.QuadraticDecay() != 0 || s.CutoffAngle() != 40 || s.AngularDecay() != 18 {
		t.Fatalf("spot cone/range = inner %v outer %v quad %v", s.AngularDecay(), s.CutoffAngle(), s.QuadraticDecay())
	}
}

func TestLightLadder(t *testing.T) {
	w := New(".")
	w.ready = true
	w.scene = core.NewNode()

	for _, name := range []string{
		"entityambient", "entityemissive", "setspecularmap", "setemissionmap", "setdiffusemap",
		"setlightspecular", "setlightambient", "setlightattenuation", "setlightfalloff", "settonemap",
	} {
		if w.commandTable()[name] == nil {
			t.Errorf("missing command %s", name)
		}
	}

	cube, err := w.Call("createcube", nil)
	if err != nil {
		t.Fatal(err)
	}
	id := []value.Value{value.Num(float64(cube.Int()))}
	if _, err := w.Call("entitycolor", append(id, value.Num(200), value.Num(80), value.Num(40))); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("entityambient", append(id, value.Num(10), value.Num(20), value.Num(30))); err != nil {
		t.Fatal(err)
	}
	amb := w.ents[cube.Int()].mat.AmbientColor()
	if abs32(amb.R-10.0/255) > 1e-4 || abs32(amb.B-30.0/255) > 1e-4 {
		t.Fatalf("ambient color = %v", amb)
	}
	if _, err := w.Call("entityshininess", append(id, value.Num(32))); err != nil {
		t.Fatal(err)
	}
	if got := unexportedFloat(w.ents[cube.Int()].mat, "udata", "shininess"); abs32(got-32) > 1e-3 {
		t.Fatalf("exponent shininess = %v", got)
	}
	if _, err := w.Call("entityshininess", append(id, value.Num(0.5))); err != nil {
		t.Fatal(err)
	}
	if got := unexportedFloat(w.ents[cube.Int()].mat, "udata", "shininess"); abs32(got-64) > 1e-3 {
		t.Fatalf("0-1 shininess = %v", got)
	}
	if _, err := w.Call("entityemissive", append(id, value.Num(4), value.Num(8), value.Num(12))); err != nil {
		t.Fatal(err)
	}
	em := w.ents[cube.Int()].mat.EmissiveColor()
	if em.G < 0.02 {
		t.Fatalf("emission = %v", em)
	}

	pt := w.makeLight(2, 0)
	pid := []value.Value{value.Num(float64(pt))}
	if _, err := w.Call("setlightattenuation", append(pid, value.Num(1), value.Num(0.14), value.Num(0.07))); err != nil {
		t.Fatal(err)
	}
	p := w.ents[pt].node.(*light.Point)
	if abs32(p.LinearDecay()-0.14) > 1e-4 || abs32(p.QuadraticDecay()-0.07) > 1e-4 || abs32(lightConstant(p)-1) > 1e-4 {
		t.Fatalf("classic decay lin=%v quad=%v c=%v", p.LinearDecay(), p.QuadraticDecay(), lightConstant(p))
	}
	if w.ents[pt].lgtRange < 10 {
		t.Fatalf("classic shadow range = %v", w.ents[pt].lgtRange)
	}
	if _, err := w.Call("setlightfalloff", []value.Value{value.Num(float64(pt)), value.Str("physical")}); err != nil {
		t.Fatal(err)
	}
	if lightConstant(p) >= 0 || p.QuadraticDecay() != 0 {
		t.Fatalf("physical falloff constant=%v quad=%v", lightConstant(p), p.QuadraticDecay())
	}
	if _, err := w.Call("setlightspecular", append(pid, value.Num(10), value.Num(20), value.Num(30))); err != nil {
		t.Fatal(err)
	}
	if !w.ents[pt].lgtSpecOn || w.ents[pt].lgtSpec.B < 0.1 {
		t.Fatal("light specular was not stored")
	}
	if _, err := w.Call("settonemap", []value.Value{value.Str("neutral")}); err != nil {
		t.Fatal(err)
	}
	if w.post.tonemap != 1 {
		t.Fatalf("tonemap = %d", w.post.tonemap)
	}
	if _, err := w.Call("lightrange", append(pid, value.Num(12))); err != nil {
		t.Fatal(err)
	}
	if lightConstant(p) >= 0 {
		t.Fatal("LightRange on a physical lamp must keep inverse-square falloff")
	}

	if _, err := w.Call("outdoorlighting", nil); err != nil {
		t.Fatal(err)
	}
	if w.lightEnv != 1 {
		t.Fatalf("outdoor env %d", w.lightEnv)
	}
	if w.ambient == nil || w.ambient.Color().B < w.ambient.Color().R {
		t.Fatal("outdoor ambient should be sky-leaning")
	}
	suns := 0
	w.eachLocalLight(func(e *Entity) {
		if e.lgtKind == 1 {
			suns++
		}
	})
	if suns < 1 {
		t.Fatal("outdoor lighting must keep a directional sun")
	}
	if _, err := w.Call("indoorlighting", nil); err != nil {
		t.Fatal(err)
	}
	if w.lightEnv != 2 || w.fogMode != 0 {
		t.Fatalf("indoor env=%d fog=%d", w.lightEnv, w.fogMode)
	}
	if lightConstant(p) >= 0 {
		t.Fatal("indoor point lights use inverse-square falloff")
	}
	if p.Intensity() < 20 {
		t.Fatalf("indoor lamp should be in candela range, intensity=%v", p.Intensity())
	}
	if w.ambient.Color().R > 0.08 {
		t.Fatalf("indoor ambient too bright: %v", w.ambient.Color())
	}
}

func TestExtraLightFeatures(t *testing.T) {
	w := &World{ents: map[int]*Entity{}, ready: true}
	w.scene = core.NewNode()
	id := w.makeLight(2, 0)
	e := w.ents[id]
	lid := []value.Value{value.Num(float64(id))}
	if _, err := w.Call("setlighttemperature", append(lid, value.Num(2000))); err != nil {
		t.Fatal(err)
	}
	col, _ := lightRadiance(e.node)
	if col.R < col.B {
		t.Fatal("2000K", col)
	}
	w.setLightTemperature(e, 6500)
	col, _ = lightRadiance(e.node)
	if col.R < 0.7 || col.B < 0.7 {
		t.Fatal("6500K", col)
	}
	setLightIntensityOf(e, 2.4)
	w.setLightEnabled(e, false)
	cur, _ := lightIntensityOf(e)
	if cur > 0.001 {
		t.Fatal("disabled", cur)
	}
	w.setLightEnabled(e, true)
	cur, _ = lightIntensityOf(e)
	if cur < 2 {
		t.Fatal("restored", cur)
	}
	w.setTimeOfDay(12)
	sun := w.ensureSun()
	if sun == nil || sun.pitch < 40 {
		t.Fatal("noon sun", sun)
	}
	key := w.createThreePoint()
	if w.ents[key] == nil || w.ents[key].lgtKind != 1 {
		t.Fatal("three point")
	}
	if _, err := w.Call("setgammacorrection", []value.Value{value.Num(1)}); err != nil {
		t.Fatal(err)
	}
	if w.shaderUnis["GammaOut"].v[0] != 1 {
		t.Fatal("gamma")
	}
	for _, name := range []string{"getlightintensity", "getlightrange", "getlighttype", "getlightred", "getlightgreen", "getlightblue", "gettimeofday", "setlightcookie", "setentitynormalmap"} {
		args := lid
		if name == "gettimeofday" {
			args = nil
		}
		if name == "setlightcookie" || name == "setentitynormalmap" {
			args = append(lid, value.Num(0))
		}
		if _, err := w.Call(name, args); err != nil {
			t.Fatal(name, err)
		}
	}
}

func TestNewMatNotEmissive(t *testing.T) {
	m := New(".").newMat()
	em := m.EmissiveColor()
	if em.R != 0 || em.G != 0 || em.B != 0 {
		t.Fatalf("default material must not be emissive, got %v", em)
	}
}
