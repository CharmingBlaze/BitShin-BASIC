package runtime

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/g3n/engine/math32"
)

func newTestEditor(t *testing.T) *editor {
	t.Helper()
	w := New(t.TempDir())
	w.ready = true
	return &editor{w: w, project: w.base, trash: map[int]*editTrash{}, prevKeys: map[int]bool{}, added: map[string]int{}}
}

func nearV(a, b math32.Vector3) bool {
	return math.Abs(float64(a.X-b.X)) < 1e-4 && math.Abs(float64(a.Y-b.Y)) < 1e-4 && math.Abs(float64(a.Z-b.Z)) < 1e-4
}

func TestEditorCreateUndoRedo(t *testing.T) {
	ed := newTestEditor(t)
	ed.target = math32.Vector3{X: 2, Y: 1, Z: -3}
	id := ed.create("Cube", "createcube")
	e := ed.w.ents[id]
	if e == nil || e.name != "Cube 1" || !nearV(e.node.GetNode().Position(), ed.target) {
		t.Fatalf("created %+v", e)
	}
	ed.undoStep()
	if ed.w.ents[id] != nil || len(ed.sel) != 0 {
		t.Fatal("undo did not remove the cube")
	}
	ed.redoStep()
	if ed.w.ents[id] != e {
		t.Fatal("redo did not bring back the same entity")
	}
}

func TestEditorMoveUndo(t *testing.T) {
	ed := newTestEditor(t)
	id := ed.create("Sphere", "createsphere")
	n := ed.w.ents[id].node.GetNode()
	before := ed.snapshot([]int{id})
	n.SetPosition(5, 6, 7)
	ed.w.ents[id].name = "Moved"
	ed.record("move", before, ed.snapshot([]int{id}))
	ed.undoStep()
	if p := n.Position(); p.X == 5 || ed.w.ents[id].name == "Moved" {
		t.Fatalf("undo left %v %q", p, ed.w.ents[id].name)
	}
	ed.redoStep()
	if p := n.Position(); !nearV(p, math32.Vector3{X: 5, Y: 6, Z: 7}) || ed.w.ents[id].name != "Moved" {
		t.Fatalf("redo gave %v %q", p, ed.w.ents[id].name)
	}
}

func TestEditorDeleteKeepsChildren(t *testing.T) {
	ed := newTestEditor(t)
	parent := ed.create("Cube", "createcube")
	child := ed.create("Cone", "createcone")
	ed.reparent(child, parent)
	ed.sel = []int{parent}
	ed.deleteSelected()
	if ed.w.ents[parent] != nil || ed.w.ents[child] != nil {
		t.Fatal("delete left the parent or child")
	}
	ed.undoStep()
	pc, cc := ed.w.ents[parent], ed.w.ents[child]
	if pc == nil || cc == nil || cc.parent != parent || cc.node.GetNode().Parent() != pc.node {
		t.Fatalf("undo delete: parent %v child %v", pc != nil, cc != nil)
	}
}

func TestEditorDuplicate(t *testing.T) {
	ed := newTestEditor(t)
	id := ed.create("Cube", "createcube")
	ed.sel = []int{id}
	ed.duplicate()
	if len(ed.sel) != 1 || ed.sel[0] == id || ed.w.ents[ed.sel[0]] == nil {
		t.Fatalf("duplicate selection %v", ed.sel)
	}
	dup := ed.sel[0]
	if ed.w.ents[dup].name != "Cube 1 copy" {
		t.Fatalf("copy name %q", ed.w.ents[dup].name)
	}
	ed.undoStep()
	if ed.w.ents[dup] != nil || ed.w.ents[id] == nil {
		t.Fatal("undo duplicate removed the wrong entity")
	}
}

func TestEditorReparentKeepsWorldTransform(t *testing.T) {
	ed := newTestEditor(t)
	w := ed.w
	parent := ed.create("Cube", "createcube")
	child := ed.create("Sphere", "createsphere")
	pn := w.ents[parent].node.GetNode()
	pn.SetPosition(4, 1, -2)
	pn.SetRotation(0.3, 1.1, -0.2)
	pn.SetScale(2, 2, 2)
	cn := w.ents[child].node.GetNode()
	cn.SetPosition(-1, 3, 5)
	cn.UpdateMatrixWorld()
	want := worldPos(cn)
	if err := w.reparentKeepWorld(child, parent); err != nil {
		t.Fatal(err)
	}
	pn.UpdateMatrixWorld()
	if got := worldPos(cn); !nearV(got, want) {
		t.Fatalf("world position moved: %v -> %v", want, got)
	}
	if w.ents[child].parent != parent {
		t.Fatal("parent not set")
	}
	// And back out to the scene.
	ed.reparent(child, 0)
	if got := worldPos(cn); !nearV(got, want) || w.ents[child].parent != 0 {
		t.Fatalf("unparent moved it to %v", got)
	}
}

func TestEditorGizmoMatrixAndEuler(t *testing.T) {
	ed := newTestEditor(t)
	w := ed.w
	parent := ed.create("Cube", "createcube")
	child := ed.create("Sphere", "createsphere")
	w.ents[parent].node.GetNode().SetPosition(10, 0, 0)
	w.ents[parent].node.GetNode().SetScale(2, 2, 2)
	w.reparentKeepWorld(child, parent)
	var m math32.Matrix4
	q := math32.NewQuaternion(0, 0, 0, 1)
	q.SetFromEuler(&math32.Vector3{X: 0.2, Y: -0.7, Z: 0.1})
	m.Compose(&math32.Vector3{X: 1, Y: 2, Z: 3}, q, &math32.Vector3{X: 1, Y: 1, Z: 1})
	ed.setWorldMatrix(w.ents[child], m)
	cn := w.ents[child].node.GetNode()
	w.ents[parent].node.GetNode().UpdateMatrixWorld()
	if got := worldPos(cn); !nearV(got, math32.Vector3{X: 1, Y: 2, Z: 3}) {
		t.Fatalf("gizmo world position %v", got)
	}

	// Euler sync is the inverse of applyRot.
	e := w.ents[ed.create("Cube", "createcube")]
	e.pitch, e.yaw, e.roll = 20, -35, 10
	w.applyRot(e)
	e.pitch, e.yaw, e.roll = 0, 0, 0
	w.syncEulerFromNode(e)
	if math.Abs(float64(e.pitch-20)) > 1e-3 || math.Abs(float64(e.yaw+35)) > 1e-3 || math.Abs(float64(e.roll-10)) > 1e-3 {
		t.Fatalf("euler %v %v %v", e.pitch, e.yaw, e.roll)
	}
}

func TestEditorSaveAndReload(t *testing.T) {
	ed := newTestEditor(t)
	a := ed.create("Cube", "createcube")
	ed.create("Sphere", "createsphere")
	ed.w.ents[a].node.GetNode().SetPosition(1, 2, 3)
	path := filepath.Join(ed.project, "scenes", "test.scene.json")
	ed.save(path)
	if ed.dirty || !fileExists(path) {
		t.Fatal("save failed")
	}
	ed2 := newTestEditor(t)
	if err := ed2.load(path); err != nil {
		t.Fatal(err)
	}
	if n := len(ed2.w.ents); n != 2 {
		t.Fatalf("reloaded %d entities", n)
	}
}
