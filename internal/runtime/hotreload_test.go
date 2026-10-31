package runtime

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/g3n/engine/texture"

	"moonbasic/internal/interp"
	"moonbasic/internal/parse"
	"moonbasic/internal/value"
)

func writeTestPNG(t *testing.T, path string, size int, c color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// bumpMtime gives path a distinct modification time old enough to pass the
// settle check, so a scan treats the file as a finished save.
func bumpMtime(t *testing.T, path string, step int) {
	t.Helper()
	at := time.Now().Add(-time.Hour).Add(time.Duration(step) * time.Second)
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

func hotCall(t *testing.T, w *World, name string, args ...value.Value) value.Value {
	t.Helper()
	v, err := w.Call(name, args)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return v
}

func TestHotReloadCommands(t *testing.T) {
	w := New(t.TempDir())
	if hotCall(t, w, "hotreloadenabled").Int() != 0 {
		t.Fatal("hot reload must start off")
	}
	hotCall(t, w, "sethotreload", value.Num(1))
	if hotCall(t, w, "hotreloadenabled").Int() != 1 || !w.HotReloadEnabled() {
		t.Fatal("SetHotReload 1 did not turn it on")
	}
	hotCall(t, w, "sethotreload", value.Num(0))
	if w.HotReloadEnabled() {
		t.Fatal("SetHotReload 0 did not turn it off")
	}
	if hotCall(t, w, "assetchanged", value.Str("none.png")).Int() != 0 {
		t.Fatal("nothing reloaded yet")
	}
}

func TestHotReloadTextureInPlace(t *testing.T) {
	dir := t.TempDir()
	w := New(dir)
	w.ready = true
	path := filepath.Join(dir, "skin.png")
	writeTestPNG(t, path, 2, color.RGBA{255, 0, 0, 255})
	tex := hotCall(t, w, "loadtexture", value.Str("skin.png")).Int()
	if tex == 0 {
		t.Fatal("LoadTexture failed")
	}
	cube := hotCall(t, w, "createcube").Int()
	hotCall(t, w, "entitytexture", value.Num(float64(cube)), value.Num(float64(tex)))
	held := w.texs[tex].tex
	if w.ents[cube].albedoTex != held {
		t.Fatal("entity does not hold the texture")
	}
	w.SetHotReload(true)
	if n := w.hotReloadScan(time.Now()); n != 0 {
		t.Fatalf("unchanged files reloaded: %d", n)
	}

	writeTestPNG(t, path, 4, color.RGBA{0, 0, 255, 255})
	bumpMtime(t, path, 1)
	if n := w.hotReloadScan(time.Now()); n != 1 {
		t.Fatalf("want 1 reload, got %d", n)
	}
	if w.texs[tex].tex != held || held.Width() != 4 {
		t.Fatalf("texture not updated in place: same=%v width=%d", w.texs[tex].tex == held, held.Width())
	}
	if w.ents[cube].albedoTex != held {
		t.Fatal("entity lost its texture")
	}
	if hotCall(t, w, "assetchanged", value.Str("skin.png")).Int() != 1 {
		t.Fatal("AssetChanged should latch once")
	}
	if hotCall(t, w, "assetchanged", value.Str("skin.png")).Int() != 0 {
		t.Fatal("AssetChanged should clear after reading")
	}

	// A broken save keeps the old texture and the game running.
	if err := os.WriteFile(path, []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	bumpMtime(t, path, 2)
	if n := w.hotReloadScan(time.Now()); n != 0 {
		t.Fatalf("broken file counted as reloaded: %d", n)
	}
	if held.Width() != 4 || w.texs[tex].tex != held {
		t.Fatal("broken file replaced the texture")
	}

	// A file still being written waits for the next poll.
	writeTestPNG(t, path, 8, color.RGBA{0, 255, 0, 255})
	now := time.Now()
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatal(err)
	}
	if n := w.hotReloadScan(now); n != 0 {
		t.Fatal("fresh file should settle first")
	}
	if n := w.hotReloadScan(now.Add(time.Second)); n != 1 || held.Width() != 8 {
		t.Fatalf("settled file not reloaded: n=%d width=%d", n, held.Width())
	}
}

func TestHotReloadSkipsPakAndMissingFiles(t *testing.T) {
	w := New(t.TempDir())
	tex := texture.NewTexture2DFromRGBA(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	w.texs[7] = &texSlot{tex: tex, path: "inpak/only.png"}
	w.texs[8] = &texSlot{tex: tex, path: "proc:noise"}
	w.images[9] = &ebiImage{path: "gone.png"}
	w.SetHotReload(true)
	if n := w.hotReloadScan(time.Now()); n != 0 {
		t.Fatalf("missing files reloaded: %d", n)
	}
	if len(w.hot.seen) != 0 {
		t.Fatalf("missing files watched: %v", w.hot.seen)
	}
}

const hotTri = "v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n"
const hotQuad = "v 0 0 0\nv 1 0 0\nv 0 1 0\nv 1 1 0\nf 1 2 3\nf 2 4 3\n"

func TestHotReloadMeshKeepsEntity(t *testing.T) {
	dir := t.TempDir()
	w := New(dir)
	w.ready = true
	path := filepath.Join(dir, "rock.obj")
	if err := os.WriteFile(path, []byte(hotTri), 0o644); err != nil {
		t.Fatal(err)
	}
	ent := hotCall(t, w, "loadmesh", value.Str("rock.obj")).Int()
	if ent == 0 {
		t.Fatal("LoadMesh failed")
	}
	hotCall(t, w, "positionentity", value.Num(float64(ent)), value.Num(3), value.Num(4), value.Num(5))
	hotCall(t, w, "entitycolor", value.Num(float64(ent)), value.Num(255), value.Num(0), value.Num(0))
	child := hotCall(t, w, "createpivot", value.Num(float64(ent))).Int()
	before := len(w.ents)
	oldNode := w.ents[ent].node
	w.SetHotReload(true)

	if err := os.WriteFile(path, []byte(hotQuad), 0o644); err != nil {
		t.Fatal(err)
	}
	bumpMtime(t, path, 1)
	if n := w.hotReloadScan(time.Now()); n != 1 {
		t.Fatalf("want 1 reload, got %d", n)
	}
	e := w.ents[ent]
	if e == nil || e.node == oldNode {
		t.Fatal("mesh entity was not swapped in place")
	}
	if len(w.ents) != before {
		t.Fatalf("reload leaked entities: %d -> %d", before, len(w.ents))
	}
	x := hotCall(t, w, "entityx", value.Num(float64(ent))).Number()
	z := hotCall(t, w, "entityz", value.Num(float64(ent))).Number()
	if x != 3 || z != 5 {
		t.Fatalf("transform lost: x=%v z=%v", x, z)
	}
	if e.tint.G != 0 || e.tint.R != 1 {
		t.Fatalf("colour lost: %+v", e.tint)
	}
	if p := w.ents[child].node.GetNode().Parent(); p == nil || p.GetNode() != e.node.GetNode() {
		t.Fatal("child entity did not move to the new node")
	}
	if e.kind != "mesh" || e.src != "rock.obj" {
		t.Fatalf("source lost: %q %q", e.kind, e.src)
	}

	// ReloadMesh(ent) with no file does the same by hand.
	if got := hotCall(t, w, "reloadmesh", value.Num(float64(ent))).Int(); got != ent {
		t.Fatalf("ReloadMesh(ent) returned %d, want %d", got, ent)
	}

	// A broken file keeps the current mesh.
	cur := w.ents[ent].node
	if err := os.WriteFile(path, []byte("f 9 9 9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bumpMtime(t, path, 2)
	w.hotReloadScan(time.Now())
	if w.ents[ent] == nil || w.ents[ent].node != cur {
		t.Fatal("broken mesh file replaced the mesh")
	}
}

func TestHotReloadScriptFunctions(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "game.mb")
	lib := filepath.Join(dir, "lib.mb")
	write := func(p, src string) {
		t.Helper()
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(lib, "Function Speed()\n  Return 1\nEnd Function\n")
	write(main, "Include \"lib.mb\"\nFunction Answer()\n  Return 41\nEnd Function\n")
	prog, err := parse.ParseFile(main)
	if err != nil {
		t.Fatal(err)
	}
	prog, err = parse.ExpandIncludes(prog, dir)
	if err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	w.SetScript(main)
	in := interp.New(prog, w)
	w.Bind(in)
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	w.SetHotReload(true)
	if len(w.hot.scriptFiles) != 2 {
		t.Fatalf("watch the script and its include: %v", w.hot.scriptFiles)
	}

	write(lib, "Function Speed()\n  Return 2\nEnd Function\n")
	bumpMtime(t, lib, 1)
	if n := w.hotReloadScan(time.Now()); n != 1 {
		t.Fatalf("include change: want 1 reload, got %d", n)
	}
	if v, err := in.CallNamed("Speed", nil); err != nil || v.Int() != 2 {
		t.Fatalf("Speed() = %v, %v; want 2", v, err)
	}

	write(main, "Include \"lib.mb\"\nFunction Answer()\n  Return 42\nEnd Function\n")
	bumpMtime(t, main, 2)
	w.hotReloadScan(time.Now())
	if v, _ := in.CallNamed("Answer", nil); v.Int() != 42 {
		t.Fatalf("Answer() = %v; want 42", v)
	}

	// A syntax error keeps the old functions.
	write(main, "Function Answer(\n")
	bumpMtime(t, main, 3)
	if n := w.hotReloadScan(time.Now()); n != 0 {
		t.Fatal("broken script counted as reloaded")
	}
	if v, _ := in.CallNamed("Answer", nil); v.Int() != 42 {
		t.Fatalf("broken script replaced Answer(): %v", v)
	}
}

func TestHotReloadShaderSource(t *testing.T) {
	dir := t.TempDir()
	w := New(dir)
	w.ready = true
	vert := filepath.Join(dir, "glow.vert")
	frag := filepath.Join(dir, "glow.frag")
	if err := os.WriteFile(vert, []byte(mbuserDefaultVert), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(frag, []byte(mbuserDefaultFrag), 0o644); err != nil {
		t.Fatal(err)
	}
	sh := hotCall(t, w, "loadshader", value.Str("glow.vert"), value.Str("glow.frag")).Int()
	if s := w.ushaders[sh]; s == nil || s.vpath != "glow.vert" || s.fpath != "glow.frag" {
		t.Fatal("LoadShader did not keep its file paths")
	}
	w.SetHotReload(true)
	changed := strings.Replace(mbuserDefaultFrag, "0.65 + 0.35", "0.5 + 0.5", 1)
	if err := os.WriteFile(frag, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	bumpMtime(t, frag, 1)
	if n := w.hotReloadScan(time.Now()); n != 1 {
		t.Fatalf("want 1 reload, got %d", n)
	}
	if !strings.Contains(w.ushaders[sh].frag, "0.5 + 0.5") {
		t.Fatal("shader source not reloaded")
	}
}

func TestHotReloadPollThrottle(t *testing.T) {
	w := New(t.TempDir())
	w.SetHotReload(true)
	w.pollHotReload()
	first := w.hot.last
	if first.IsZero() {
		t.Fatal("first poll should scan")
	}
	w.pollHotReload()
	if !w.hot.last.Equal(first) {
		t.Fatal("polls within the interval must not scan again")
	}
}
