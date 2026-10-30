package runtime

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
)

func TestMergeLevelBatchesByCellAndMaterial(t *testing.T) {
	w := New(t.TempDir())
	root := w.addEntity(&Entity{node: core.NewNode()}, 0)
	colors := []math32.Color{{R: 1}, {G: 1}, {B: 1}, {R: 1, G: 1}}
	var probe int
	for i := 0; i < 2000; i++ {
		id := w.meshEnt(geometry.NewCube(1), root)
		e := w.ents[id]
		c := colors[i%4]
		e.mat.SetColor(&c)
		x, z := float32(i%50)*8-200, float32(i/50)*8-160
		e.node.GetNode().SetPosition(x, 0.5, z)
		if i == 777 {
			probe = id
		}
	}
	lv, err := w.mergeLevel(root, 64, true)
	if err != nil {
		t.Fatal(err)
	}
	// 400 x 320 m in 64 m cells is 7 x 6 cells (some partial), 4 colors each.
	if lv.stats.sourceMeshes != 2000 {
		t.Fatalf("merged %d meshes, want 2000", lv.stats.sourceMeshes)
	}
	if lv.stats.cells > 7*6*4 || lv.stats.cells < 4 {
		t.Fatalf("%d cells", lv.stats.cells)
	}
	if w.ents[probe].node.GetNode().Visible() {
		t.Fatal("a merged source mesh should be hidden")
	}
	// Every merged vertex keeps its world position: the probe cube's
	// center must be inside some cell's vertex range.
	want := worldPos(w.ents[probe].node.GetNode())
	found := false
	for _, c := range lv.cells {
		buf := w.ents[c.ent].mesh.GetGeometry().VBO(gls.VertexPosition).Buffer()
		for i := 0; i+2 < len(*buf); i += 3 {
			if math32.Abs((*buf)[i]-want.X) <= 0.51 && math32.Abs((*buf)[i+2]-want.Z) <= 0.51 && math32.Abs((*buf)[i+1]-1) < 1e-4 {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("no merged vertex near the probe cube at %v", want)
	}
}

func TestMergeLevelSkipsHiddenTemplates(t *testing.T) {
	w := New(t.TempDir())
	root := w.addEntity(&Entity{node: core.NewNode()}, 0)
	tmpl := w.meshEnt(geometry.NewCube(1), root)
	w.ents[tmpl].node.GetNode().SetVisible(false)
	w.meshEnt(geometry.NewCube(1), tmpl) // child of a hidden template
	w.meshEnt(geometry.NewCube(1), root)
	lv, err := w.mergeLevel(root, 64, true)
	if err != nil {
		t.Fatal(err)
	}
	if lv.stats.sourceMeshes != 1 {
		t.Fatalf("merged %d meshes; the hidden template and its child must stay out", lv.stats.sourceMeshes)
	}
}

// writeTestLevel writes a glTF with n box nodes over a grid, cycling through
// mats materials, like an exported city block.
func writeTestLevel(t *testing.T, path string, n, mats int) {
	t.Helper()
	// One unit box: 8 corners, 12 triangles.
	pos := []float32{-.5, 0, -.5, .5, 0, -.5, .5, 0, .5, -.5, 0, .5, -.5, 1, -.5, .5, 1, -.5, .5, 1, .5, -.5, 1, .5}
	idx := []uint16{0, 2, 1, 0, 3, 2, 4, 5, 6, 4, 6, 7, 0, 1, 5, 0, 5, 4, 1, 2, 6, 1, 6, 5, 2, 3, 7, 2, 7, 6, 3, 0, 4, 3, 4, 7}
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, pos)
	binary.Write(&buf, binary.LittleEndian, idx)
	doc := map[string]any{
		"asset":       map[string]any{"version": "2.0"},
		"buffers":     []any{map[string]any{"byteLength": buf.Len(), "uri": "data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())}},
		"bufferViews": []any{map[string]any{"buffer": 0, "byteOffset": 0, "byteLength": len(pos) * 4}, map[string]any{"buffer": 0, "byteOffset": len(pos) * 4, "byteLength": len(idx) * 2}},
		"accessors": []any{
			map[string]any{"bufferView": 0, "componentType": 5126, "count": 8, "type": "VEC3", "min": []float32{-.5, 0, -.5}, "max": []float32{.5, 1, .5}},
			map[string]any{"bufferView": 1, "componentType": 5123, "count": len(idx), "type": "SCALAR"},
		},
	}
	var materials, meshes, nodes []any
	var roots []int
	for m := 0; m < mats; m++ {
		materials = append(materials, map[string]any{"pbrMetallicRoughness": map[string]any{"baseColorFactor": []float32{float32(m) / float32(mats), 0.5, 0.5, 1}}})
		meshes = append(meshes, map[string]any{"primitives": []any{map[string]any{"attributes": map[string]any{"POSITION": 0}, "indices": 1, "material": m}}})
	}
	for i := 0; i < n; i++ {
		nodes = append(nodes, map[string]any{"mesh": i % mats, "translation": []float32{float32(i%40) * 6, 0, float32(i/40) * 6}, "scale": []float32{4, 2 + float32(i%5), 4}})
		roots = append(roots, i)
	}
	doc["materials"], doc["meshes"], doc["nodes"] = materials, meshes, nodes
	doc["scenes"] = []any{map[string]any{"nodes": roots}}
	doc["scene"] = 0
	b, _ := json.Marshal(doc)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLevelMergesAGLTFCity(t *testing.T) {
	dir := t.TempDir()
	w := New(dir)
	path := filepath.Join(dir, "city.gltf")
	writeTestLevel(t, path, 1500, 3)
	root, err := w.loadMeshFile(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	lv, err := w.mergeLevel(root, 64, false)
	if err != nil {
		t.Fatal(err)
	}
	if lv.stats.sourceMeshes != 1500 {
		t.Fatalf("merged %d of 1500 boxes (skipped %d)", lv.stats.sourceMeshes, lv.stats.skipped)
	}
	// 240 x 228 m is 4 x 4 cells, 3 materials: at most 48 draws for 1500.
	if lv.stats.cells > 48 {
		t.Fatalf("%d cells for 1500 boxes in 3 materials; materials did not group", lv.stats.cells)
	}
	t.Logf("1500 glTF nodes -> %d draws", lv.stats.cells)
}

func TestMergeLevelPutsColorsInVertices(t *testing.T) {
	w := New(t.TempDir())
	w.shadow.on = true // newMat picks the lit shader that reads VertexColor
	root := w.addEntity(&Entity{node: core.NewNode()}, 0)
	colors := []math32.Color{{R: 1}, {G: 1}, {B: 1}, {R: 1, G: 1}}
	for i := 0; i < 2000; i++ {
		id := w.meshEnt(geometry.NewCube(1), root)
		c := colors[i%4]
		w.ents[id].mat.SetColor(&c)
		w.ents[id].node.GetNode().SetPosition(float32(i%50)*8-200, 0.5, float32(i/50)*8-160)
	}
	lv, err := w.mergeLevel(root, 64, true)
	if err != nil {
		t.Fatal(err)
	}
	if lv.stats.cells > 8*6 { // -200..192 x -160..152 in 64 m cells
		t.Fatalf("%d cells; four colors should share one draw per cell", lv.stats.cells)
	}
	m, ok := w.ents[lv.cells[0].ent].mesh.Materials()[0].IMaterial().(*litMat)
	if !ok || !m.vertexColor {
		t.Fatal("merged cell should draw with vertex color")
	}
	if w.ents[lv.cells[0].ent].mesh.GetGeometry().VBO(gls.VertexColor) == nil {
		t.Fatal("merged cell has no VertexColor")
	}
}

func TestMergeLevelFreesWhatItBakesButNotGameplayObjects(t *testing.T) {
	w := New(t.TempDir())
	root := w.addEntity(&Entity{node: core.NewNode()}, 0)
	for i := 0; i < 50; i++ {
		id := w.meshEnt(geometry.NewCube(1), root)
		w.ents[id].node.GetNode().SetPosition(float32(i), 0, 0)
	}
	door := w.meshEnt(geometry.NewCube(1), root)
	w.ents[door].collKind = 1
	named := w.meshEnt(geometry.NewCube(1), root)
	w.ents[named].name = "level_box"
	before := len(w.ents)
	lv, err := w.mergeLevel(root, 64, false)
	if err != nil {
		t.Fatal(err)
	}
	if lv.stats.sourceMeshes != 50 || lv.stats.skipped != 2 {
		t.Fatalf("merged %d, skipped %d; want 50 and the door + named box left alone", lv.stats.sourceMeshes, lv.stats.skipped)
	}
	if w.ents[door] == nil || w.ents[named] == nil {
		t.Fatal("collision and named entities must keep their handles")
	}
	if got := len(w.ents); got != before-50+lv.stats.cells {
		t.Fatalf("%d entities after merge, want %d", got, before-50+lv.stats.cells)
	}
}
