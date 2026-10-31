package runtime

import (
	"testing"

	"moonbasic/internal/value"
)

func TestSpriteBlockedBySolidTile(t *testing.T) {
	w := New(".")
	w.ready = true
	w.scrW, w.scrH = 800, 480
	cmds := w.commandTable()

	idv, err := cmds["createtilemap"]([]value.Value{
		value.Num(32), value.Num(32), value.Num(4), value.Num(3),
	})
	if err != nil {
		t.Fatal(err)
	}
	mapID := int(idv.Num)
	// Floor on row 2.
	for x := 0; x < 4; x++ {
		if _, err := cmds["settile"]([]value.Value{
			value.Num(float64(mapID)), value.Num(float64(x)), value.Num(2), value.Num(1),
		}); err != nil {
			t.Fatal(err)
		}
	}
	// Decoration on row 1 must not block.
	if _, err := cmds["settile"]([]value.Value{
		value.Num(float64(mapID)), value.Num(1), value.Num(1), value.Num(2),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := cmds["settilesolid"]([]value.Value{
		value.Num(float64(mapID)), value.Num(2), value.Num(0),
	}); err != nil {
		t.Fatal(err)
	}

	w.sprites[7] = &ebiSprite{id: 7, x: 48, y: 48, sx: 1, sy: 1, fw: 16, fh: 16, alpha: 1, cr: 1, cg: 1, cb: 1}
	hit, err := cmds["spritehittile"]([]value.Value{value.Num(7), value.Num(float64(mapID))})
	if err != nil {
		t.Fatal(err)
	}
	if hit.Num != 0 {
		t.Fatalf("open tile should not block, got %v", hit.Num)
	}

	// Center 80 sits on the floor row (y 64..96) with a 16px body.
	w.sprites[7].x, w.sprites[7].y = 16, 80
	hit, err = cmds["spritehittile"]([]value.Value{value.Num(7), value.Num(float64(mapID))})
	if err != nil {
		t.Fatal(err)
	}
	if hit.Num != 1 {
		t.Fatal("floor tile should block")
	}

	got, err := cmds["gettile"]([]value.Value{value.Num(float64(mapID)), value.Num(0), value.Num(2)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Num != 1 {
		t.Fatalf("GetTile = %v", got.Num)
	}
	at, err := cmds["tileat"]([]value.Value{value.Num(float64(mapID)), value.Num(40), value.Num(70)})
	if err != nil {
		t.Fatal(err)
	}
	if at.Num != 1 {
		t.Fatalf("TileAt = %v", at.Num)
	}

	if _, err := cmds["camerafollow2d"]([]value.Value{value.Num(7), value.Num(0), value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	cx, _ := cmds["camera2dx"](nil)
	cy, _ := cmds["camera2dy"](nil)
	if cx.Num != 16-400 || cy.Num != 80-240 {
		t.Fatalf("camera %v,%v", cx.Num, cy.Num)
	}
}

func TestSpriteSheetAnim(t *testing.T) {
	w := New(".")
	s := &ebiSprite{id: 1, animFrom: 2, animCount: 3, animFPS: 10, frame: 2, alpha: 1}
	w.sprites[1] = s
	w.delta = 0.25
	w.tickSpriteAnims()
	// 0.25s at 10 fps steps twice: frame 2 -> 3 -> 4 (range is 2,3,4).
	if s.frame != 4 {
		t.Fatalf("frame %d", s.frame)
	}
}
