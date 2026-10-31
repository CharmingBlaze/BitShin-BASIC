package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/g3n/engine/light"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

func newSceneWorld(t *testing.T, dir string) *World {
	t.Helper()
	w := New(dir)
	w.ambient = light.NewAmbient(&math32.Color{R: 0.28, G: 0.30, B: 0.36}, 1)
	return w
}

func assetPath(t *testing.T, name string) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "..", "examples", "assets", name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Skip("asset missing:", err)
	}
	return filepath.ToSlash(p)
}

// docOf parses JSON text the way a file read would.
func docOf(t *testing.T, src string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(src), &m); err != nil {
		t.Fatalf("bad test JSON: %v", err)
	}
	return m
}

func roundTrip(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func rowsByUID(t *testing.T, doc map[string]any) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	list, _ := doc["entities"].([]any)
	for _, raw := range list {
		r := raw.(map[string]any)
		uid := strOf(r["uid"])
		if _, dup := out[uid]; dup {
			t.Fatalf("uid %q twice in the scene", uid)
		}
		out[uid] = r
	}
	return out
}

func sameRows(t *testing.T, want, got map[string]map[string]any) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("entity count: want %d got %d", len(want), len(got))
	}
	for uid, a := range want {
		b, ok := got[uid]
		if !ok {
			t.Fatalf("uid %q missing after reload", uid)
		}
		for k := range a {
			if !jsonNear(a[k], b[k]) {
				ja, _ := json.Marshal(a[k])
				jb, _ := json.Marshal(b[k])
				t.Errorf("%s (%v) %s: want %s got %s", uid, a["kind"], k, ja, jb)
			}
		}
		for k := range b {
			if _, ok := a[k]; !ok {
				t.Errorf("%s: extra key %q after reload", uid, k)
			}
		}
	}
}

func everyKindScene(t *testing.T) string {
	mesh := assetPath(t, "walker.gltf")
	stone := assetPath(t, "stone.png")
	grass := assetPath(t, "grass.png")
	return `{
  "version": 2,
  "environment": {
    "ambient": [60, 70, 90], "ambientIntensity": 0.8,
    "clear": [20, 22, 30],
    "fog": {"mode": "linear", "color": [120, 130, 150], "near": 15, "far": 140, "density": 0.02},
    "sky": "sunset",
    "ibl": {"on": true, "intensity": 0.7},
    "post": {"on": true, "tonemap": "aces", "exposure": 1.3, "bloom": 0.4, "lut": "warm", "lutAmount": 0.5}
  },
  "entities": [
    {"uid": "kid", "name": "Kid", "kind": "sphere", "parent": "hub", "pos": [0, 2, 0], "scale": [0.5, 0.5, 0.5],
     "material": {"pbr": true, "shading": "unlit"}},
    {"uid": "hub", "name": "Hub", "kind": "pivot", "pos": [4, 0, -2], "rot": [0, 45, 0]},
    {"uid": "crate", "name": "Crate", "tags": ["pickup", "wood"], "script": "spin=1", "kind": "cube",
     "pos": [1, 2, 3], "rot": [10, 20, 30], "scale": [1, 2, 1], "visible": true, "color": [200, 100, 50, 255],
     "material": {"pbr": true, "metallic": 0.7, "roughness": 0.25, "ao": 0.9, "emissive": [10, 20, 30],
       "shading": "toon", "toon": {"bands": 4, "rim": 0.3, "spec": 0.2}, "fx": 24, "ibl": 1,
       "castShadows": false, "receiveShadows": true,
       "textures": {"albedo": "` + stone + `", "normal": "` + grass + `"}},
     "body": {"type": "dynamic", "shape": "box", "size": [1, 2, 1], "mass": 2, "friction": 0.4, "restitution": 0.3, "layer": 2}},
    {"uid": "ball", "kind": "sphere", "pos": [0, 6, 0], "body": {"type": "dynamic", "shape": "sphere", "radius": 0.5}},
    {"uid": "cyl", "kind": "cylinder", "pos": [3, 0, 0], "body": {"type": "static", "shape": "cylinder", "size": [0.5, 1, 0.5], "radius": 0.5}},
    {"uid": "cone", "kind": "cone", "pos": [5, 0, 0], "visible": false},
    {"uid": "floor", "kind": "plane", "body": {"type": "static", "shape": "mesh"}},
    {"uid": "quad", "kind": "quad", "pos": [0, 1, 9]},
    {"uid": "disk", "kind": "disk", "pos": [0, 1, 10]},
    {"uid": "torus", "kind": "torus", "pos": [0, 1, 11], "color": [10, 200, 10, 128]},
    {"uid": "cap", "kind": "capsule", "pos": [-3, 1, 0], "body": {"type": "kinematic", "shape": "capsule", "size": [0.4, 0.6, 0.4], "radius": 0.4}},
    {"uid": "pyr", "kind": "pyramid", "pos": [-5, 0, 0]},
    {"uid": "wedge", "kind": "wedge", "pos": [-7, 0, 0]},
    {"uid": "tube", "kind": "tube", "pos": [-9, 1, 0], "body": {"type": "static", "shape": "convex"}},
    {"uid": "trigger", "kind": "cube", "pos": [0, 0, 20], "body": {"type": "kinematic", "shape": "box", "size": [2, 1, 2], "sensor": true}},
    {"uid": "walker", "name": "Walker", "kind": "mesh", "src": "` + mesh + `", "pos": [6, 0, 6], "body": {"type": "static", "shape": "auto"}},
    {"uid": "sun", "kind": "light", "rot": [50, 30, 0], "light": {"type": "directional", "color": [255, 240, 220], "intensity": 1.4, "shadows": true}},
    {"uid": "lamp", "kind": "light", "parent": "hub", "pos": [0, 3, 0], "light": {"type": "point", "color": [255, 180, 90], "intensity": 3, "range": 12}},
    {"uid": "spot", "kind": "light", "pos": [0, 5, 0], "rot": [90, 0, 0], "light": {"type": "spot", "intensity": 4, "range": 20, "cone": [10, 35]}},
    {"uid": "amb", "kind": "light", "light": {"type": "ambient", "intensity": 0.3}},
    {"uid": "panel", "kind": "arealight", "pos": [0, 4, -4], "area": {"size": [2, 1], "color": [255, 230, 200], "intensity": 6, "range": 15, "twoSided": true}},
    {"uid": "cam", "name": "Main", "kind": "camera", "pos": [0, 5, -12], "rot": [15, 0, 0], "camera": {"fov": 70, "near": 0.2, "far": 900, "active": true, "clear": [20, 22, 30]}}
  ]
}`
}

func TestSceneV2RoundTripsEveryKind(t *testing.T) {
	src := everyKindScene(t)
	w := newSceneWorld(t, t.TempDir())
	if err := w.loadSceneV2(docOf(t, src), false); err != nil {
		t.Fatalf("load: %v", err)
	}
	first := roundTrip(t, w.sceneV2())
	want := rowsByUID(t, first)
	if len(want) != 22 {
		t.Fatalf("want 22 entities, got %d", len(want))
	}
	// Spot checks that the file's values reached the engine.
	crate := w.ents[w.entityByUID("crate")]
	if crate == nil || crate.pbrWrap == nil || crate.pbrWrap.metallic < 0.69 || crate.pbrWrap.shading != shadeToon || crate.pbrWrap.toonBands != 4 {
		t.Fatalf("crate material not applied: %+v", crate)
	}
	if !crate.pbrWrap.noFog || !crate.meshNoCast {
		t.Fatal("crate fx / cast shadows not applied")
	}
	if crate.bodyType != 1 || crate.collKind != 1 || w.phys3.GetMass(w.entityByUID("crate")) < 1.9 {
		t.Fatal("crate body not built")
	}
	if crate.meta.script != "spin=1" || len(crate.meta.tags) != 2 {
		t.Fatal("crate tags / script")
	}
	if e := w.ents[w.entityByUID("trigger")]; e.collKind != 5 {
		t.Fatalf("sensor box collKind %d", e.collKind)
	}
	if e := w.ents[w.entityByUID("cam")]; w.cam != e.cam {
		t.Fatal("scene camera should be active")
	}
	if w.fogMode != 1 || w.scn.sky != "sunset" || w.post.tonemap != 2 || w.post.lut.name != "warm" {
		t.Fatalf("environment: fog %d sky %q tonemap %d lut %q", w.fogMode, w.scn.sky, w.post.tonemap, w.post.lut.name)
	}
	checks := map[string]map[string]any{
		"spot":   {"light": map[string]any{"type": "spot", "color": []any{255.0, 255.0, 255.0}, "intensity": 4.0, "range": 20.0, "cone": []any{10.0, 35.0}, "shadows": false}},
		"lamp":   {"parent": "hub", "pos": []any{0.0, 3.0, 0.0}},
		"panel":  {"area": map[string]any{"size": []any{2.0, 1.0}, "color": []any{255.0, 230.0, 200.0}, "intensity": 6.0, "range": 15.0, "twoSided": true}},
		"cam":    {"camera": map[string]any{"fov": 70.0, "near": 0.2, "far": 900.0, "active": true, "clear": []any{20.0, 22.0, 30.0}}},
		"cone":   {"visible": false},
		"torus":  {"color": []any{10.0, 200.0, 10.0, 128.0}},
		"walker": {"kind": "mesh", "src": assetPath(t, "walker.gltf")},
	}
	for uid, fields := range checks {
		for k, v := range fields {
			if !jsonNear(want[uid][k], v) {
				got, _ := json.Marshal(want[uid][k])
				t.Errorf("%s.%s = %s", uid, k, got)
			}
		}
	}
	if tex := mapOf(mapOf(want["crate"]["material"])["textures"]); tex["albedo"] == nil || tex["normal"] == nil {
		t.Errorf("crate textures %v", tex)
	}
	if b := mapOf(want["walker"]["body"]); b["shape"] != "auto" {
		t.Errorf("walker body %v", b)
	}
	if b := mapOf(want["floor"]["body"]); b["shape"] != "mesh" || b["type"] != "static" {
		t.Errorf("floor body %v", b)
	}
	if b := mapOf(want["crate"]["body"]); !jsonNear(b["friction"], 0.4) || !jsonNear(b["restitution"], 0.3) || !jsonNear(b["layer"], 2) {
		t.Errorf("crate body %v", b)
	}
	env := mapOf(first["environment"])

	// Save, clear, load in the same world.
	if err := w.loadSceneV2(first, false); err != nil {
		t.Fatalf("reload: %v", err)
	}
	sameRows(t, want, rowsByUID(t, roundTrip(t, w.sceneV2())))

	// And into a fresh world.
	w2 := newSceneWorld(t, t.TempDir())
	if err := w2.loadSceneV2(first, false); err != nil {
		t.Fatalf("fresh load: %v", err)
	}
	second := roundTrip(t, w2.sceneV2())
	sameRows(t, want, rowsByUID(t, second))
	if !jsonNear(env, second["environment"]) {
		a, _ := json.Marshal(env)
		b, _ := json.Marshal(second["environment"])
		t.Fatalf("environment changed:\n%s\n%s", a, b)
	}
}

func TestSceneV2ReadsV1Dumps(t *testing.T) {
	w := newSceneWorld(t, t.TempDir())
	cube := w.createCubeMesh(nil)
	gx, gy, gz := toG3N(1, 2, 3)
	w.ents[cube].node.GetNode().SetPosition(gx, gy, gz)
	w.ents[cube].name = "Box"
	w.setEntityRGB(w.ents[cube], 255, 0, 0)
	w.makeLight(2, 0)
	w.spawnCamera(0)
	v1 := roundTrip(t, w.sceneJSON())
	if sceneVersionOf(v1) != 1 {
		t.Fatal("SceneSave output should read as v1")
	}
	w2 := newSceneWorld(t, t.TempDir())
	if err := w2.loadSceneV2(v1, false); err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for id := range w2.ents {
		row := w2.entityJSON(id)
		if row == nil {
			continue
		}
		kinds[strOf(row["kind"])]++
		if strOf(row["name"]) == "Box" {
			if !jsonNear(row["pos"], []any{1.0, 2.0, 3.0}) {
				t.Fatalf("v1 pos %v", row["pos"])
			}
			if !jsonNear(row["color"], []any{255.0, 0.0, 0.0, 255.0}) {
				t.Fatalf("v1 color %v", row["color"])
			}
		}
	}
	if kinds["cube"] != 1 || kinds["light"] != 1 || kinds["camera"] != 1 {
		t.Fatalf("v1 kinds %v", kinds)
	}
	// Rows with no kind are cubes, as before.
	w3 := newSceneWorld(t, t.TempDir())
	if err := w3.loadSceneV2(docOf(t, `{"entities": [{"id": 4, "x": 1}]}`), false); err != nil {
		t.Fatal(err)
	}
	if len(w3.ents) != 1 {
		t.Fatalf("kindless v1 row: %d entities", len(w3.ents))
	}
	for _, e := range w3.ents {
		if e.kind != "cube" {
			t.Fatalf("kindless row became %q", e.kind)
		}
	}
	// A v1 dump used as a prefab converts too.
	dir := t.TempDir()
	if err := writeJSONFile(filepath.Join(dir, "old.json"), v1); err != nil {
		t.Fatal(err)
	}
	w4 := newSceneWorld(t, dir)
	b := &sceneBuild{w: w4, dir: dir}
	root, err := b.instantiate("old.json", "p1", nil)
	if err != nil || root == 0 {
		t.Fatalf("v1 prefab: %v", err)
	}
}

func TestSceneV2UIDsAreStable(t *testing.T) {
	w := newSceneWorld(t, t.TempDir())
	doc := docOf(t, `{"version": 2, "entities": [
		{"uid": "a", "kind": "cube"}, {"uid": "b", "kind": "sphere", "parent": "a"}, {"kind": "cone"}]}`)
	if err := w.loadSceneV2(doc, false); err != nil {
		t.Fatal(err)
	}
	rows := rowsByUID(t, roundTrip(t, w.sceneV2()))
	if rows["a"] == nil || rows["b"] == nil || len(rows) != 3 {
		t.Fatalf("uids %v", rows)
	}
	var made string
	for uid := range rows {
		if uid != "a" && uid != "b" {
			made = uid
		}
	}
	if made == "" {
		t.Fatal("a row with no uid should get one")
	}
	// Saved and loaded again, every uid is the same.
	again := roundTrip(t, w.sceneV2())
	if err := w.loadSceneV2(again, false); err != nil {
		t.Fatal(err)
	}
	if got := rowsByUID(t, roundTrip(t, w.sceneV2())); got[made] == nil || got["a"] == nil || strOf(got["b"]["parent"]) != "a" {
		t.Fatalf("uids changed: %v", got)
	}
	// A script entity gets a uid on first ask and keeps it.
	id := w.createCubeMesh(nil)
	u := w.entityUID(id)
	if u == "" || w.entityUID(id) != u || w.entityByUID(u) != id {
		t.Fatal("entityUID not stable")
	}
	w.freeSceneEntity(id)
	if w.entityByUID(u) != 0 {
		t.Fatal("freed entity still found by uid")
	}
}

func TestSceneV2ChildBeforeParent(t *testing.T) {
	w := newSceneWorld(t, t.TempDir())
	doc := docOf(t, `{"version": 2, "entities": [
		{"uid": "leaf", "kind": "cube", "parent": "mid", "pos": [0, 0, 1]},
		{"uid": "mid", "kind": "pivot", "parent": "top", "pos": [0, 1, 0]},
		{"uid": "top", "kind": "pivot", "pos": [10, 0, 0]}]}`)
	if err := w.loadSceneV2(doc, false); err != nil {
		t.Fatal(err)
	}
	leaf, mid, top := w.entityByUID("leaf"), w.entityByUID("mid"), w.entityByUID("top")
	if w.ents[leaf].parent != mid || w.ents[mid].parent != top || w.ents[top].parent != 0 {
		t.Fatal("parents not resolved")
	}
	x, y, z := w.blitzPosOf(leaf)
	if x != 10 || y != 1 || z != 1 {
		t.Fatalf("leaf world pos %v %v %v", x, y, z)
	}
	// SaveScene writes parents first.
	list := w.sceneV2()["entities"].([]any)
	order := map[string]int{}
	for i, r := range list {
		order[strOf(r.(map[string]any)["uid"])] = i
	}
	if !(order["top"] < order["mid"] && order["mid"] < order["leaf"]) {
		t.Fatalf("save order %v", order)
	}
}

func TestSceneV2AdditiveLoad(t *testing.T) {
	w := newSceneWorld(t, t.TempDir())
	doc := docOf(t, `{"version": 2, "entities": [
		{"uid": "p", "kind": "pivot"}, {"uid": "c", "kind": "cube", "parent": "p", "pos": [1, 0, 0]}]}`)
	for i := 0; i < 2; i++ {
		if err := w.loadSceneV2(roundTrip(t, doc), true); err != nil {
			t.Fatal(err)
		}
	}
	rows := rowsByUID(t, roundTrip(t, w.sceneV2()))
	if len(rows) != 4 {
		t.Fatalf("two additive loads: %d entities", len(rows))
	}
	// The second copy got fresh uids and its child still found its parent.
	for uid, r := range rows {
		if r["kind"] == "cube" {
			p := w.entityByUID(strOf(r["parent"]))
			if p == 0 || w.ents[p].kind != "pivot" {
				t.Fatalf("%s lost its parent", uid)
			}
		}
	}
	if err := w.loadSceneV2(roundTrip(t, doc), false); err != nil {
		t.Fatal(err)
	}
	if got := rowsByUID(t, roundTrip(t, w.sceneV2())); len(got) != 2 || got["p"] == nil {
		t.Fatalf("replacing load: %v", got)
	}
}

func TestApplyEntityJSONUpdatesInPlace(t *testing.T) {
	w := newSceneWorld(t, t.TempDir())
	id, err := w.applyEntityJSON(docOf(t, `{"uid": "box1", "kind": "cube", "pos": [1, 1, 1],
		"material": {"pbr": true, "metallic": 0.1}, "body": {"type": "static", "shape": "box"}}`), true)
	if err != nil || id == 0 {
		t.Fatal(err)
	}
	child, _ := w.applyEntityJSON(docOf(t, `{"uid": "kid", "kind": "sphere", "parent": "box1"}`), true)
	before := roundTrip(t, w.entityJSON(id))
	edit := roundTrip(t, before)
	edit["pos"] = []any{5.0, 1.0, -2.0}
	edit["rot"] = []any{0.0, 90.0, 0.0}
	edit["tags"] = []any{"moved"}
	mat := mapOf(edit["material"])
	mat["metallic"] = 0.8
	mat["roughness"] = 0.1
	got, err := w.applyEntityJSON(edit, true)
	if err != nil || got != id {
		t.Fatalf("update made a new entity: %d vs %d (%v)", got, id, err)
	}
	after := w.entityJSON(id)
	if !jsonNear(after["pos"], edit["pos"]) || !jsonNear(mapOf(after["material"])["metallic"], 0.8) {
		t.Fatalf("update not applied: %v", after)
	}
	if bx, _, bz := w.blitzPosOf(id); bx != 5 || bz != -2 {
		t.Fatal("node not moved")
	}
	// Undo: the old snapshot puts everything back, tags cleared.
	if got, _ := w.applyEntityJSON(before, true); got != id {
		t.Fatal("undo made a new entity")
	}
	sameRows(t, map[string]map[string]any{"box1": before}, map[string]map[string]any{"box1": roundTrip(t, w.entityJSON(id))})
	// Changing the kind rebuilds under the same uid and keeps the children.
	swap := roundTrip(t, before)
	swap["kind"] = "sphere"
	nid, err := w.applyEntityJSON(swap, true)
	if err != nil || nid == 0 || w.entityUID(nid) != "box1" || w.ents[nid].kind != "sphere" {
		t.Fatalf("kind change: %d %v", nid, err)
	}
	if w.ents[child].parent != nid {
		t.Fatal("child lost on rebuild")
	}
	// keepUID false always makes a new entity.
	dup, _ := w.applyEntityJSON(swap, false)
	if dup == nid || w.entityUID(dup) == "box1" {
		t.Fatal("paste should get a new uid")
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPrefabSaveAndInstantiate(t *testing.T) {
	dir := t.TempDir()
	w := newSceneWorld(t, dir)
	w.ready = true
	root, _ := w.applyEntityJSON(docOf(t, `{"uid": "r", "name": "Tower", "kind": "cube", "pos": [7, 0, 7], "rot": [0, 30, 0],
		"body": {"type": "static", "shape": "box"}}`), true)
	w.applyEntityJSON(docOf(t, `{"uid": "top", "kind": "sphere", "parent": "r", "pos": [0, 3, 0], "color": [0, 0, 255, 255]}`), true)
	if _, err := w.Call("saveprefab", []value.Value{value.Str("tower"), value.Num(float64(root))}); err != nil {
		t.Fatal(err)
	}
	saved, err := readSceneFile(filepath.Join(dir, "prefabs", "tower.json"))
	if err != nil {
		t.Fatal(err)
	}
	rows := sceneRows(saved)
	if len(rows) != 2 || !jsonNear(rows[0]["pos"], []any{0.0, 0.0, 0.0}) || rows[0]["parent"] != nil {
		t.Fatalf("prefab rows %v", rows)
	}
	v, err := w.Call("instantiateprefab", []value.Value{value.Str("tower"), value.Num(-4), value.Num(0), value.Num(2)})
	if err != nil {
		t.Fatal(err)
	}
	inst := int(v.Number())
	ie := w.ents[inst]
	if ie == nil || ie.name != "Tower" || ie.meta.prefab == nil {
		t.Fatal("instance root")
	}
	if x, _, z := w.blitzPosOf(inst); x != -4 || z != 2 {
		t.Fatalf("instance at %v %v", x, z)
	}
	iuid := w.entityUID(inst)
	top := w.entityByUID(iuid + "/top")
	if top == 0 || w.ents[top].parent != inst {
		t.Fatal("member uid / parent")
	}
	if ie.bodyType != 2 {
		t.Fatal("instance body")
	}
	// SpawnPrefab is the same command.
	if v, err := w.Call("spawnprefab", []value.Value{value.Str("tower")}); err != nil || v.Number() == 0 {
		t.Fatal("SpawnPrefab", err)
	}
	// An untouched instance saves as a reference with no overrides.
	row := w.entityJSON(inst)
	if row["kind"] != "prefab" || row["src"] != "tower" || row["overrides"] != nil {
		t.Fatalf("instance row %v", row)
	}
	// A changed member shows up as an override and survives a reload.
	w.setEntityRGB(w.ents[top], 255, 0, 0)
	row = roundTrip(t, w.entityJSON(inst))
	ov := mapOf(mapOf(row["overrides"])["top"])
	if ov == nil || ov["color"] == nil || len(ov) != 1 {
		t.Fatalf("override %v", row["overrides"])
	}
	scene := roundTrip(t, w.sceneV2())
	for _, r := range scene["entities"].([]any) {
		if strings.Contains(strOf(r.(map[string]any)["uid"]), "/") {
			t.Fatal("instance members should not be written out")
		}
	}
	w2 := newSceneWorld(t, dir)
	w2.ready = true
	if err := w2.loadSceneV2(scene, false); err != nil {
		t.Fatal(err)
	}
	top2 := w2.ents[w2.entityByUID(iuid+"/top")]
	if top2 == nil || top2.tint.R < 0.99 || top2.tint.B > 0.01 {
		t.Fatal("override not applied on load")
	}
	sameRows(t, rowsByUID(t, scene), rowsByUID(t, roundTrip(t, w2.sceneV2())))
	// Old one-object SavePrefab files still spawn.
	writeFile(t, filepath.Join(dir, "prefabs", "old.json"), `{"kind": "sphere", "name": "Old", "x": 1, "y": 2, "z": 3}`)
	v, err = w.Call("spawnprefab", []value.Value{value.Str("old")})
	if err != nil || w.ents[int(v.Number())].kind != "sphere" {
		t.Fatal("legacy prefab", err)
	}
	if x, y, z := w.blitzPosOf(int(v.Number())); x != 1 || y != 2 || z != 3 {
		t.Fatal("legacy prefab position")
	}
}

func TestNestedPrefabsWithOverrides(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "lamp.prefab.json"), `{"version": 2, "entities": [
		{"uid": "post", "kind": "cylinder", "scale": [0.2, 2, 0.2]},
		{"uid": "bulb", "kind": "sphere", "parent": "post", "pos": [0, 2, 0], "color": [255, 255, 255, 255]}]}`)
	writeFile(t, filepath.Join(dir, "street.prefab.json"), `{"version": 2, "entities": [
		{"uid": "base", "kind": "pivot"},
		{"uid": "l1", "kind": "prefab", "src": "lamp.prefab.json", "parent": "base", "pos": [-3, 0, 0],
		 "overrides": {"bulb": {"color": [255, 0, 0, 255]}}},
		{"uid": "l2", "kind": "prefab", "src": "lamp.prefab.json", "parent": "base", "pos": [3, 0, 0]}]}`)
	// A variant: the root is itself a prefab instance with overrides.
	writeFile(t, filepath.Join(dir, "bluelamp.prefab.json"), `{"version": 2, "entities": [
		{"uid": "v", "kind": "prefab", "src": "lamp", "overrides": {"bulb": {"color": [0, 0, 255, 255]}}}]}`)
	w := newSceneWorld(t, dir)
	doc := docOf(t, `{"version": 2, "entities": [
		{"uid": "s", "kind": "prefab", "src": "street", "pos": [0, 0, 10],
		 "overrides": {"l2/bulb": {"scale": [2, 2, 2]}, "l1": {"name": "Left"}}},
		{"uid": "b", "kind": "prefab", "src": "bluelamp", "pos": [20, 0, 0]},
		{"uid": "extra", "kind": "cube", "parent": "s/l1/bulb", "pos": [0, 1, 0]}]}`)
	if err := w.loadSceneV2(doc, false); err != nil {
		t.Fatal(err)
	}
	bulb1 := w.ents[w.entityByUID("s/l1/bulb")]
	bulb2 := w.ents[w.entityByUID("s/l2/bulb")]
	if bulb1 == nil || bulb2 == nil {
		t.Fatal("nested members missing")
	}
	if bulb1.tint.G > 0.01 {
		t.Fatal("inner override (red bulb) not applied")
	}
	if s := bulb2.node.GetNode().Scale(); s.X != 2 {
		t.Fatal("outer override (big bulb) not applied")
	}
	if w.ents[w.entityByUID("s/l1")].name != "Left" {
		t.Fatal("override of a nested instance root")
	}
	if blue := w.ents[w.entityByUID("b/bulb")]; blue == nil || blue.tint.B < 0.99 || blue.tint.R > 0.01 {
		t.Fatal("variant override")
	}
	if e := w.ents[w.entityByUID("extra")]; e == nil || e.parent != w.entityByUID("s/l1/bulb") {
		t.Fatal("scene entity parented to a prefab member")
	}
	// The save keeps references: three rows, and the street's overrides are
	// only the scene's own.
	saved := roundTrip(t, w.sceneV2())
	rows := rowsByUID(t, saved)
	if len(rows) != 3 {
		t.Fatalf("saved rows %d: %v", len(rows), rows)
	}
	ov := mapOf(rows["s"]["overrides"])
	if len(ov) != 2 || ov["l2/bulb"] == nil || ov["l1"] == nil || ov["l1/bulb"] != nil {
		t.Fatalf("street overrides %v", ov)
	}
	if rows["b"]["overrides"] != nil {
		t.Fatalf("variant instance should have no overrides of its own: %v", rows["b"]["overrides"])
	}
	w2 := newSceneWorld(t, dir)
	if err := w2.loadSceneV2(saved, false); err != nil {
		t.Fatal(err)
	}
	sameRows(t, rows, rowsByUID(t, roundTrip(t, w2.sceneV2())))
	// Deleting a member is an override too.
	w2.freeSceneTree(w2.entityByUID("s/l2/bulb"))
	ov = mapOf(roundTrip(t, w2.entityJSON(w2.entityByUID("s")))["overrides"])
	if !boolOf(mapOf(ov["l2/bulb"])["deleted"], false) {
		t.Fatalf("deleted member override %v", ov)
	}
	w3 := newSceneWorld(t, dir)
	if err := w3.loadSceneV2(roundTrip(t, w2.sceneV2()), false); err != nil {
		t.Fatal(err)
	}
	if w3.entityByUID("s/l2/bulb") != 0 || w3.entityByUID("s/l1/bulb") == 0 {
		t.Fatal("deleted member came back")
	}
}

func TestPrefabCycleGuard(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "self.json"), `{"version": 2, "entities": [
		{"uid": "r", "kind": "cube"}, {"uid": "again", "kind": "prefab", "src": "self.json", "parent": "r"}]}`)
	writeFile(t, filepath.Join(dir, "a.json"), `{"version": 2, "entities": [{"uid": "x", "kind": "prefab", "src": "b.json"}]}`)
	writeFile(t, filepath.Join(dir, "b.json"), `{"version": 2, "entities": [{"uid": "y", "kind": "prefab", "src": "a.json"}]}`)
	w := newSceneWorld(t, dir)
	err := w.loadSceneV2(docOf(t, `{"version": 2, "entities": [
		{"uid": "one", "kind": "prefab", "src": "self.json"},
		{"uid": "two", "kind": "prefab", "src": "a.json"},
		{"uid": "gone", "kind": "prefab", "src": "missing.json", "overrides": {"q": {"name": "kept"}}}]}`), false)
	if err == nil || !strings.Contains(err.Error(), "includes itself") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want cycle and missing errors, got %v", err)
	}
	if w.entityByUID("one") == 0 || w.entityByUID("one/again") == 0 {
		t.Fatal("the instance and its broken inner reference should both exist")
	}
	if len(w.ents) > 10 {
		t.Fatalf("cycle expanded: %d entities", len(w.ents))
	}
	// A missing prefab is kept as a reference with its overrides.
	rows := rowsByUID(t, roundTrip(t, w.sceneV2()))
	if g := rows["gone"]; g == nil || g["src"] != "missing.json" || mapOf(g["overrides"])["q"] == nil {
		t.Fatalf("missing prefab row %v", rows["gone"])
	}
}

func TestSceneCommands(t *testing.T) {
	dir := t.TempDir()
	w := newSceneWorld(t, dir)
	w.ready = true
	call := func(name string, args ...value.Value) value.Value {
		t.Helper()
		v, err := w.Call(name, args)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return v
	}
	a := w.createCubeMesh(nil)
	b := w.createCubeMesh(nil)
	w.ents[a].name = "Door"
	call("setentitytag", value.Num(float64(a)), value.Str("enemy, boss"))
	call("addentitytag", value.Num(float64(b)), value.Str("enemy"))
	call("addentitytag", value.Num(float64(b)), value.Str("enemy"))
	if call("findentity", value.Str("door")).Number() != float64(a) {
		t.Fatal("FindEntity")
	}
	if call("findentitybytag", value.Str("enemy"), value.Num(2)).Number() != float64(b) {
		t.Fatal("FindEntityByTag n")
	}
	if call("countentitiesbytag", value.Str("enemy")).Number() != 2 {
		t.Fatal("CountEntitiesByTag")
	}
	if call("entitytag", value.Num(float64(a)), value.Num(2)).Str != "boss" || len(w.ents[b].meta.tags) != 1 {
		t.Fatal("EntityTag$ / AddEntityTag")
	}
	call("removeentitytag", value.Num(float64(a)), value.Str("boss"))
	if call("entityhastag", value.Num(float64(a)), value.Str("boss")).Number() != 0 {
		t.Fatal("RemoveEntityTag")
	}
	call("setentityscript", value.Num(float64(a)), value.Str("open=1"))
	if call("entityscript", value.Num(float64(a))).Str != "open=1" {
		t.Fatal("EntityScript$")
	}
	uid := call("entityuid", value.Num(float64(a))).Str
	if uid == "" || call("entitybyuid", value.Str(uid)).Number() != float64(a) {
		t.Fatal("EntityUID$ / EntityByUID")
	}
	call("savescene", value.Str("level.scene.json"))
	saved, err := readSceneFile(filepath.Join(dir, "level.scene.json"))
	if err != nil || numOf(saved["version"], 0) != 2 {
		t.Fatalf("SaveScene: %v", err)
	}
	call("loadscene", value.Str("level.scene.json"), value.Num(1))
	if call("countentitiesbytag", value.Str("enemy")).Number() != 4 {
		t.Fatal("additive LoadScene")
	}
	call("loadscene", value.Str("level.scene.json"))
	if call("countentitiesbytag", value.Str("enemy")).Number() != 2 {
		t.Fatal("replacing LoadScene")
	}
	d := w.ents[int(call("findentity", value.Str("Door")).Number())]
	if d == nil || d.meta.uid != uid || d.meta.script != "open=1" {
		t.Fatal("uid / script after LoadScene")
	}
	var names []string
	for _, n := range CommandNames() {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, want := range []string{"savescene", "loadscene", "saveprefab", "instantiateprefab", "spawnprefab", "findentity", "entityuid"} {
		if i := sort.SearchStrings(names, want); i >= len(names) || names[i] != want {
			t.Fatalf("%s not registered", want)
		}
	}
}
