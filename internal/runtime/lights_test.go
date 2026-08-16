package runtime

import (
	"testing"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/math32"
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
	delete(w.ents, dir)
	pt := w.makeLight(2, 0)
	w.ents[pt].castShadow = true
	if !w.hasShadowLight() {
		t.Fatal("shadow-casting point light should count")
	}
}

func TestNewMatNotEmissive(t *testing.T) {
	m := New(".").newMat()
	em := m.EmissiveColor()
	if em.R != 0 || em.G != 0 || em.B != 0 {
		t.Fatalf("default material must not be emissive, got %v", em)
	}
}
