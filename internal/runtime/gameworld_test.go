package runtime

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/g3n/engine/core"

	"bitshinbasic/internal/value"
)

func TestLODStepFollowsDetail(t *testing.T) {
	near := &terrain{lod: 1}
	if near.lodStep(0) != 1 || near.lodStep(3) != 2 || near.lodStep(9) != 4 {
		t.Fatalf("default lod steps: %d %d %d", near.lodStep(0), near.lodStep(3), near.lodStep(9))
	}
	fine := &terrain{lod: 1, detailPx: 4}
	if fine.lodStep(3) != 1 {
		t.Fatalf("finer pixels should stay on step 1 at dist2 3, got %d", fine.lodStep(3))
	}
	coarse := &terrain{lod: 1, detailPx: 16}
	if coarse.lodStep(3) != 4 {
		t.Fatalf("coarser pixels should drop to step 4 quickly, got %d", coarse.lodStep(3))
	}
}

func TestScatterAcceptsFlatGround(t *testing.T) {
	if !scatterAccept("grass", 0.9, 4, 1, 12, true) {
		t.Fatal("meadow should take grass")
	}
	if scatterAccept("grass", 0.4, 4, 1, 12, true) {
		t.Fatal("cliff should skip grass")
	}
	if scatterAccept("tree", 0.9, 12, 1, 12, true) {
		t.Fatal("snow line should skip trees")
	}
	if !scatterAccept("prop", 0.5, 1.2, 1, 12, true) {
		t.Fatal("moderate slope should take a prop")
	}
	if scatterAccept("prop", 0.2, 4, 1, 12, true) {
		t.Fatal("a cliff should skip props")
	}
}

func TestTimeOfDayMovesTheSun(t *testing.T) {
	w := New(t.TempDir())
	sun := &Entity{node: core.NewNode(), lgtKind: 1}
	w.ents[1] = sun
	if _, err := w.Call("settimeofday", []value.Value{value.Num(12)}); err != nil {
		t.Fatal(err)
	}
	if math.Abs(float64(sun.pitch-58)) > 0.2 {
		t.Fatalf("noon pitch %v", sun.pitch)
	}
	if w.fogRGB.R < 0.3 {
		t.Fatalf("noon fog %v", w.fogRGB)
	}
	if _, err := w.Call("settimeofday", []value.Value{value.Num(1)}); err != nil {
		t.Fatal(err)
	}
	if sun.pitch != 8 {
		t.Fatalf("night pitch %v", sun.pitch)
	}
	if w.fogRGB.R > 0.1 {
		t.Fatalf("night fog %v", w.fogRGB)
	}
	got, err := w.Call("gettimeofday", nil)
	if err != nil || got.Number() != 1 {
		t.Fatalf("hour %v %v", got, err)
	}
	head := mbterrainFragmentHead
	if !strings.Contains(head, "TerrainHasBlend") || !strings.Contains(mbterrainFragmentTail, "TerrainBlendMap") {
		t.Fatal("blend map is not in the terrain shader")
	}
}

func TestRoomsDoorsInventoryAndActors(t *testing.T) {
	w := New(t.TempDir())
	a := &Entity{node: core.NewNode()}
	b := &Entity{node: core.NewNode()}
	door := &Entity{node: core.NewNode()}
	item := &Entity{node: core.NewNode()}
	talk := &Entity{node: core.NewNode()}
	player := &Entity{node: core.NewNode()}
	actor := &Entity{node: core.NewNode()}
	w.ents[1], w.ents[2], w.ents[3] = a, b, door
	w.ents[4], w.ents[5], w.ents[6], w.ents[7] = item, talk, player, actor

	rv, err := w.Call("createroom", nil)
	if err != nil {
		t.Fatal(err)
	}
	r1 := rv.Int()
	rv, err = w.Call("createroom", nil)
	if err != nil {
		t.Fatal(err)
	}
	r2 := rv.Int()
	_, _ = w.Call("setroom", []value.Value{value.Num(float64(1)), value.Num(float64(r1))})
	_, _ = w.Call("roomadd", []value.Value{value.Num(float64(r2)), value.Num(2)})
	_, _ = w.Call("roomlink", []value.Value{value.Num(float64(r1)), value.Num(float64(r2))})
	_, _ = w.Call("enterroom", []value.Value{value.Num(float64(r1))})
	if !a.node.GetNode().Visible() || !b.node.GetNode().Visible() {
		t.Fatal("linked rooms should both show")
	}
	_, _ = w.Call("enterroom", []value.Value{value.Num(0)})
	if a.node.GetNode().Visible() || b.node.GetNode().Visible() {
		t.Fatal("outdoor should hide room props")
	}

	_, _ = w.Call("createdoor", []value.Value{value.Num(3), value.Num(float64(r1)), value.Num(float64(r2)), value.Num(0)})
	used, _ := w.Call("use", []value.Value{value.Num(3)})
	if used.Number() != 1 || !w.play.doors[3].open {
		t.Fatal("use should open the door")
	}
	w.tickPlay(1)
	if door.yaw < 50 {
		t.Fatalf("door yaw %v", door.yaw)
	}

	_, _ = w.Call("setitem", []value.Value{value.Num(4), value.Str("key")})
	_, _ = w.Call("use", []value.Value{value.Num(4)})
	has, _ := w.Call("inventoryhas", []value.Value{value.Str("Key")})
	if has.Number() != 1 || item.node.GetNode().Visible() {
		t.Fatal("use should pocket the item")
	}

	_, _ = w.Call("setdialogue", []value.Value{value.Num(5), value.Str("Hello|Bye")})
	_, _ = w.Call("use", []value.Value{value.Num(5)})
	line, _ := w.Call("dialogueline", nil)
	if line.String() != "Hello" {
		t.Fatalf("line %q", line.String())
	}
	_, _ = w.Call("dialogueadvance", nil)
	_, _ = w.Call("dialogueadvance", nil)
	on, _ := w.Call("dialogueon", nil)
	if on.Number() != 0 {
		t.Fatal("dialogue should end")
	}

	w.setScriptPos(6, 0, 0, 0)
	w.setScriptPos(7, 10, 0, 0)
	_, _ = w.Call("setplayer", []value.Value{value.Num(6)})
	_, _ = w.Call("createactor", []value.Value{value.Num(7), value.Num(8), value.Num(4)})
	w.tickPlay(0.5)
	st, _ := w.Call("actorstate", []value.Value{value.Num(7)})
	if st.Number() != 0 {
		t.Fatalf("far actor state %v", st.Number())
	}
	w.setScriptPos(6, 4, 0, 0)
	w.tickPlay(0.5)
	st, _ = w.Call("actorstate", []value.Value{value.Num(7)})
	if st.Number() != 1 {
		t.Fatalf("chase state %v", st.Number())
	}
	ax, _, _, _ := w.scriptPos(7)
	if ax >= 10 {
		t.Fatalf("actor did not step closer: %v", ax)
	}
	w.setScriptPos(7, 4.4, 0, 0)
	w.tickPlay(0.1)
	st, _ = w.Call("actorstate", []value.Value{value.Num(7)})
	if st.Number() != 2 {
		t.Fatalf("attack state %v", st.Number())
	}

	if _, err := w.Call("settimeofday", []value.Value{value.Num(14)}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "slot.json")
	if _, err := w.Call("savegame", []value.Value{value.Str(path)}); err != nil {
		t.Fatal(err)
	}
	w.play.inv = nil
	w.play.doors[3].open = false
	w.play.hour = 0
	if _, err := w.Call("loadgame", []value.Value{value.Str(path)}); err != nil {
		t.Fatal(err)
	}
	if !w.inventoryHas("key") || !w.play.doors[3].open || w.play.hour != 14 {
		t.Fatalf("save mismatch inv=%v open=%v hour=%v", w.play.inv, w.play.doors[3].open, w.play.hour)
	}
	raw, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(raw), `"inventory"`) {
		t.Fatalf("save file %v %s", err, raw)
	}
}
