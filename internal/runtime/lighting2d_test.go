package runtime

import (
	"testing"

	"moonbasic/internal/value"
)

func TestTileOccluderEdgesSkipInterior(t *testing.T) {
	w, cmds := newWorld2D(t)
	m := call(t, cmds, "createtilemap", 10, 10, 4, 4).Num
	call(t, cmds, "settile", m, 1, 1, 1)
	call(t, cmds, "settile", m, 2, 1, 1)
	call(t, cmds, "settileoccluders", m, 0)
	edges := w.occluderEdges(20, 15, 100, nil)
	// Two cells side by side: a 2x1 block has 6 outside edges, not 8.
	if len(edges) != 6 {
		t.Fatalf("%d edges, want 6: %v", len(edges), edges)
	}
	for _, e := range edges {
		// Clockwise: the outward normal (by-ay, ax-bx) points away from the
		// block's centre (20, 15).
		nx, ny := e.by-e.ay, e.ax-e.bx
		mx, my := (e.ax+e.bx)/2-20, (e.ay+e.by)/2-15
		if nx*mx+ny*my <= 0 {
			t.Errorf("edge %v faces inward", e)
		}
	}
	if _, err := cmds["settileoccluders"]([]value.Value{value.Num(m), value.Num(3)}); err == nil {
		t.Error("SetTileOccluders accepted a missing layer")
	}
}

func TestFlippedSpriteOccluderWindsOutward(t *testing.T) {
	w, cmds := newWorld2D(t)
	s := &ebiSprite{id: 5, x: 50, y: 50, sx: -2, sy: 1, visible: true, alpha: 1, img: w.solidImage(10, 10, 1, 1, 1)}
	w.sprites[5] = s
	call(t, cmds, "setspriteoccluder", 5)
	edges := w.occluderEdges(50, 50, 100, nil)
	if len(edges) != 4 {
		t.Fatalf("%d edges", len(edges))
	}
	for _, e := range edges {
		nx, ny := e.by-e.ay, e.ax-e.bx
		if nx*((e.ax+e.bx)/2-50)+ny*((e.ay+e.by)/2-50) <= 0 {
			t.Errorf("mirrored sprite edge %v faces inward", e)
		}
	}
}

func TestLight2DCommands(t *testing.T) {
	w, cmds := newWorld2D(t)
	if w.lighting2DOn() {
		t.Fatal("lighting on before EnableLighting2D")
	}
	call(t, cmds, "enablelighting2d")
	call(t, cmds, "setambient2d", 255, 0, 51)
	if a := w.lit2.ambient; !w.lighting2DOn() || a[0] != 1 || a[1] != 0 || a[2] < 0.199 || a[2] > 0.201 {
		t.Fatalf("ambient %v", w.lit2.ambient)
	}
	id := call(t, cmds, "createlight2d", 10, 20, 0).Num // radius clamps to 1
	l := w.lit2.lights[int(id)]
	if l == nil || l.radius != 1 || !l.shadows || !l.visible {
		t.Fatalf("light %+v", l)
	}
	call(t, cmds, "setlight2dcone", id, 90, 45)
	call(t, cmds, "showlight2d", id, 0)
	if l.dir != 90 || l.cone != 45 || l.visible {
		t.Fatalf("cone/visibility %+v", l)
	}
	// A parented light follows the sprite and turns with it.
	w.sprites[3] = &ebiSprite{id: 3, x: 100, y: 100, rot: 90, sx: 1, sy: 1, visible: true, alpha: 1}
	call(t, cmds, "parentlight2d", id, 3, 10, 0)
	x, y, dir := w.lightPos(l)
	if !near2(x, 100) || !near2(y, 110) || dir != 180 {
		t.Fatalf("parented light at %v,%v dir %v", x, y, dir)
	}
	call(t, cmds, "freelight2d", id)
	if _, err := cmds["positionlight2d"]([]value.Value{value.Num(id), value.Num(0), value.Num(0)}); err == nil {
		t.Error("freed light still accepted")
	}
	call(t, cmds, "setspritelit", 3, 0)
	if !w.sprites[3].unlit {
		t.Error("SetSpriteLit 0 did not mark the sprite unlit")
	}
}
