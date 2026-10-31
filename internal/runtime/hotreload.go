package runtime

import (
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"time"

	"github.com/g3n/engine/texture"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/hajimehoshi/ebiten/v2"

	"moonbasic/internal/ast"
	"moonbasic/internal/glslang"
	"moonbasic/internal/parse"
	"moonbasic/internal/value"
)

// Asset hot reload. While on, the frame loop checks the modification time of
// every asset loaded from disk about twice a second, between frames on the
// main thread, and reloads changed files in place: textures (every material
// and sprite using the handle updates), LoadMesh entities (transform, parent,
// children, colour, texture and shader stay), images, LoadShader programs and
// the script's functions. Files inside a pak are not watched. A file that
// fails to load is reported and the old asset stays.

const (
	hotReloadEvery  = 500 * time.Millisecond
	hotReloadSettle = 100 * time.Millisecond // a file this fresh may still be being written
)

type hotReload struct {
	on          bool
	last        time.Time
	seen        map[string]fileStamp // watched file -> stamp last acted on
	fired       map[string]bool      // AssetChanged latches
	hook        string               // OnAssetReloaded function
	scriptFiles []string             // main script and its includes (absolute)
	scriptSrc   string               // file reparsed on a script change
}

type fileStamp struct {
	mod  time.Time
	size int64
}

func (a fileStamp) same(b fileStamp) bool { return a.size == b.size && a.mod.Equal(b.mod) }

type hotAsset struct {
	kind string // texture, mesh, image, shader, script
	rel  string // path as the script loaded it
	disk string
	id   int
}

// SetHotReload turns the asset watcher on or off. `moon run -watch` and runs
// from the IDE (MOON_IDE=1) turn it on; built games leave it off.
func (w *World) SetHotReload(on bool) {
	w.hot.on = on
	w.hot.last = time.Time{}
	if !on {
		return
	}
	// The first scan only records stamps, so files are not reloaded at once.
	w.hotReloadScan(time.Now())
}

// HotReloadEnabled reports whether the asset watcher is on.
func (w *World) HotReloadEnabled() bool { return w.hot.on }

func (w *World) hotReloadCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// SetHotReload on: watch loaded assets and reload changed files in place.
		"sethotreload": n(func(a []value.Value) (value.Value, error) {
			w.SetHotReload(argI(a, 0, 1) != 0)
			return z()
		}),
		"hotreloadenabled": n(func(a []value.Value) (value.Value, error) {
			if w.hot.on {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		// OnAssetReloaded fn$: call Function fn(path$) after each reload; "" clears.
		"onassetreloaded": n(func(a []value.Value) (value.Value, error) {
			w.hot.hook = argS(a, 0)
			return z()
		}),
		// AssetChanged(path$): True once after the file at path$ reloaded.
		"assetchanged": n(func(a []value.Value) (value.Value, error) {
			key := hotKey(w.resolve(argS(a, 0)))
			if w.hot.fired[key] {
				delete(w.hot.fired, key)
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
	}
}

// pollHotReload runs once per frame; it scans at most every hotReloadEvery.
func (w *World) pollHotReload() {
	if !w.hot.on {
		return
	}
	now := time.Now()
	if now.Sub(w.hot.last) < hotReloadEvery {
		return
	}
	w.hot.last = now
	w.hotReloadScan(now)
}

// hotKey is the map key for a disk path: absolute, clean, and case-folded
// where the file system ignores case.
func hotKey(disk string) string {
	if abs, err := filepath.Abs(disk); err == nil {
		disk = abs
	}
	disk = filepath.Clean(disk)
	if goruntime.GOOS == "windows" || goruntime.GOOS == "darwin" {
		disk = strings.ToLower(disk)
	}
	return disk
}

// hotReloadScan stats every watched file and reloads the changed ones.
// It returns how many files were reloaded.
func (w *World) hotReloadScan(now time.Time) int {
	if w.hot.seen == nil {
		w.hot.seen = map[string]fileStamp{}
	}
	if w.hot.scriptFiles == nil {
		w.hotCollectScript()
	}
	groups := map[string][]hotAsset{}
	var order []string
	add := func(kind, rel string, id int) {
		if rel == "" || strings.HasPrefix(rel, "proc:") {
			return
		}
		disk := w.resolve(rel)
		key := hotKey(disk)
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], hotAsset{kind: kind, rel: rel, disk: disk, id: id})
	}
	for _, f := range w.hot.scriptFiles {
		add("script", f, 0)
	}
	for id, t := range w.texs {
		if t != nil && t.tex != nil {
			add("texture", t.path, id)
		}
	}
	for id, im := range w.images {
		if im != nil {
			add("image", im.path, id)
		}
	}
	for id, sh := range w.ushaders {
		if sh != nil {
			add("shader", sh.vpath, id)
			if sh.fpath != sh.vpath {
				add("shader", sh.fpath, id)
			}
		}
	}
	for id, e := range w.ents {
		if e != nil && e.kind == "mesh" && e.src != "" && e.node != nil {
			add("mesh", e.src, id)
		}
	}
	sort.Strings(order)
	done := map[string]bool{} // shader ids / the script, reloaded once per scan
	n := 0
	for _, key := range order {
		list := groups[key]
		st, err := os.Stat(list[0].disk)
		if err != nil || st.IsDir() {
			continue // missing, or only inside a pak: not watched
		}
		stamp := fileStamp{mod: st.ModTime(), size: st.Size()}
		old, known := w.hot.seen[key]
		if !known {
			w.hot.seen[key] = stamp
			continue
		}
		if old.same(stamp) {
			continue
		}
		if age := now.Sub(stamp.mod); age >= 0 && age < hotReloadSettle {
			continue // still being saved; look again next poll
		}
		w.hot.seen[key] = stamp
		if w.reloadHotFile(key, list, done) {
			n++
		}
	}
	return n
}

// reloadHotFile reloads every asset loaded from one changed file. It prints
// one line per file and keeps old assets when a load fails.
func (w *World) reloadHotFile(key string, list []hotAsset, done map[string]bool) bool {
	kind, rel := list[0].kind, list[0].rel
	if filepath.IsAbs(rel) {
		if r, err := filepath.Rel(w.base, rel); err == nil && !strings.HasPrefix(r, "..") {
			rel = r
		}
	}
	ok, failed := 0, 0
	var firstErr error
	for _, a := range list {
		err := w.reloadHotAsset(a, done)
		if err != nil {
			failed++
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		ok++
	}
	if failed > 0 {
		fmt.Printf("HotReload: %s %s: %v (kept the old one)\n", kind, rel, firstErr)
	}
	if ok == 0 {
		return false
	}
	if len(list) > 1 && kind == "mesh" {
		fmt.Printf("HotReload: %s %s (%d entities)\n", kind, rel, ok)
	} else {
		fmt.Printf("HotReload: %s %s\n", kind, rel)
	}
	if w.hot.fired == nil {
		w.hot.fired = map[string]bool{}
	}
	w.hot.fired[key] = true
	w.fireHook(w.hot.hook, value.Str(rel))
	return true
}

// reloadHotAsset reloads one asset. Decoders can panic on a half-written
// file, so a panic becomes an error and the old asset stays.
func (w *World) reloadHotAsset(a hotAsset, done map[string]bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	switch a.kind {
	case "texture":
		return w.reloadTextureSlot(w.texs[a.id], a.rel)
	case "image":
		return w.reloadImage(a.id)
	case "mesh":
		return w.reloadMeshEntity(a.id)
	case "shader":
		tag := fmt.Sprintf("shader:%d", a.id)
		if done[tag] {
			return nil
		}
		done[tag] = true
		return w.reloadUserShader(a.id)
	case "script":
		if done["script"] {
			return nil
		}
		done["script"] = true
		if err := w.reloadScript(w.hot.scriptSrc); err != nil {
			return err
		}
		w.hotCollectScript() // includes may have changed
	}
	return nil
}

// hotCollectScript lists the script and the files it includes. The IDE runs
// a temp copy of the tab; MOON_SCRIPT_SOURCE names the file the user edits.
func (w *World) hotCollectScript() {
	w.hot.scriptFiles = []string{}
	src := w.hot.scriptSrc
	if src == "" {
		src = w.scriptPath
		if env := os.Getenv("MOON_SCRIPT_SOURCE"); env != "" {
			if st, err := os.Stat(env); err == nil && !st.IsDir() {
				src = env
			}
		}
		if src == "" {
			return
		}
		if abs, err := filepath.Abs(src); err == nil {
			src = abs
		}
		w.hot.scriptSrc = src
	}
	seen := map[string]bool{}
	var walk func(path string)
	walk = func(path string) {
		key := hotKey(path)
		if seen[key] {
			return
		}
		seen[key] = true
		w.hot.scriptFiles = append(w.hot.scriptFiles, path)
		prog, err := parse.ParseFile(path)
		if err != nil {
			return
		}
		for _, s := range prog.Stmts {
			var inc string
			switch t := s.(type) {
			case *ast.IncludeStmt:
				inc = t.Path
			case *ast.ImportStmt:
				inc = t.Path
			default:
				continue
			}
			file := filepath.FromSlash(strings.ReplaceAll(inc, "\\", "/"))
			if !filepath.IsAbs(file) {
				file = filepath.Join(filepath.Dir(path), file)
			}
			walk(file)
		}
	}
	walk(src)
}

// reloadTextureSlot decodes file into the slot's existing texture, so every
// material, sprite and terrain layer holding it shows the new image.
func (w *World) reloadTextureSlot(slot *texSlot, file string) error {
	if slot == nil || slot.tex == nil {
		return fmt.Errorf("no texture")
	}
	path, err := w.openPath(file)
	if err != nil {
		return err
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".dds", ".ktx", ".ktx2":
		b, err := readFileBytes(path)
		if err != nil {
			return err
		}
		img, err := parseGPUImage(b)
		if err != nil {
			return err
		}
		if img.compressed {
			slot.tex.SetCompressedData(img.w, img.h, img.format, int32(len(img.data)), img.data)
		} else if img.rgba != nil {
			slot.tex.SetFromRGBA(img.rgba)
		} else {
			return fmt.Errorf("texture: empty image")
		}
	default:
		rgba, err := texture.DecodeImage(path)
		if err != nil {
			return err
		}
		slot.tex.SetFromRGBA(rgba)
	}
	slot.path = file
	return nil
}

// reloadImage decodes a LoadImage file again. In 2D a same-size image is
// rewritten in place; a resized one is swapped in every sprite and emitter.
func (w *World) reloadImage(id int) error {
	im := w.images[id]
	if im == nil {
		return fmt.Errorf("no image %d", id)
	}
	path, err := w.openPath(im.path)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	m, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	b := m.Bounds()
	if im.img != nil {
		if b.Dx() == im.w && b.Dy() == im.h {
			rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
			draw.Draw(rgba, rgba.Bounds(), m, b.Min, draw.Src)
			im.img.WritePixels(rgba.Pix)
		} else {
			old, nimg := im.img, ebiten.NewImageFromImage(m)
			im.img = nimg
			for _, s := range w.sprites {
				if s != nil && s.img == old {
					s.img = nimg
				}
			}
			for _, e := range w.emitters {
				if e != nil && e.img == old {
					e.img = nimg
				}
			}
		}
	} else {
		im.src = m
		if im.glTex != 0 {
			gl.DeleteTextures(1, &im.glTex)
			im.glTex = 0 // the HUD uploads src again
		}
	}
	im.w, im.h = b.Dx(), b.Dy()
	return nil
}

// reloadMeshEntity loads a LoadMesh entity's file again and swaps the new
// node in. Transform, visibility, parent, child entities, colour, alpha,
// texture and shader carry over; other material settings come from the file.
func (w *World) reloadMeshEntity(id int) error {
	e := w.ents[id]
	if e == nil || e.node == nil || e.src == "" {
		return fmt.Errorf("no mesh entity %d", id)
	}
	nid, err := w.loadMeshFile(e.src, 0)
	if err != nil {
		return err
	}
	ne := w.ents[nid]
	// Detach the temp entity without freeing its node.
	delete(w.ents, nid)
	w.recycleHandle(&w.freeIDs, nid)
	if ne == nil || ne.node == nil {
		return fmt.Errorf("empty mesh")
	}
	nn := ne.node.GetNode()
	if p := nn.Parent(); p != nil {
		p.GetNode().Remove(ne.node)
	}
	on := e.node.GetNode()
	pos, q, sc := on.Position(), on.Quaternion(), on.Scale()
	nn.SetPositionVec(&pos)
	nn.SetQuaternionQuat(&q)
	nn.SetScaleVec(&sc)
	nn.SetVisible(on.Visible())
	nn.SetName(on.Name())
	for _, c := range w.ents {
		if c != nil && c != e && c.parent == id && c.node != nil {
			on.Remove(c.node)
			nn.Add(c.node)
		}
	}
	if p := on.Parent(); p != nil {
		pn := p.GetNode()
		idx := -1
		for i, ch := range pn.Children() {
			if ch.GetNode() == on {
				idx = i
				break
			}
		}
		pn.Remove(e.node)
		if idx >= 0 && idx <= len(pn.Children()) {
			pn.AddAt(idx, ne.node)
		} else {
			pn.Add(ne.node)
		}
	}
	tint, tex, shader := e.tint, e.albedoTex, e.ushader
	e.node, e.mesh, e.mat = ne.node, ne.mesh, ne.mat
	e.pbr, e.pbrWrap, e.usePBR = ne.pbr, ne.pbrWrap, ne.usePBR
	e.albedoTex, e.ushader = nil, 0
	e.shadowGeomDirty = true
	*ne = Entity{}
	if tint.R != 1 || tint.G != 1 || tint.B != 1 {
		w.setEntityRGB(e, tint.R*255, tint.G*255, tint.B*255)
	}
	if tint.A != 1 {
		_, _ = w.callDirect("entityalpha", []value.Value{value.Num(float64(id)), value.Num(float64(tint.A))})
	}
	if tex != nil {
		for tid, t := range w.texs {
			if t != nil && t.tex == tex {
				_, _ = w.callDirect("entitytexture", []value.Value{value.Num(float64(id)), value.Num(float64(tid))})
				break
			}
		}
	}
	if shader != 0 && w.ushaders[shader] != nil {
		_, _ = w.bindEntityShader(id, shader)
	}
	if w.wire {
		if e.mat != nil {
			e.mat.SetWireframe(true)
		}
		if e.pbr != nil {
			e.pbr.SetWireframe(true)
		}
	}
	return nil
}

// reloadUserShader reads a LoadShader program's files again. A program that
// fails validation keeps the old one. G3N caches programs by name, so the new
// source registers under a new name and bound entities switch to it.
func (w *World) reloadUserShader(id int) error {
	sh := w.ushaders[id]
	if sh == nil || sh.vpath == "" {
		return fmt.Errorf("no shader %d", id)
	}
	vb, err := w.readShaderFile(sh.vpath)
	if err != nil {
		return err
	}
	fb, err := w.readShaderFile(sh.fpath)
	if err != nil {
		return err
	}
	vert, frag := stripGLSLVersion(string(vb)), stripGLSLVersion(string(fb))
	if !strings.Contains(vert, "#include") && !strings.Contains(frag, "#include") {
		rv := glslang.Compile(vert, glslang.StageVert)
		rf := glslang.Compile(frag, glslang.StageFrag)
		if !rv.OK || !rf.OK {
			return fmt.Errorf("%s", strings.TrimSpace(rv.Log+"\n"+rf.Log))
		}
	}
	sh.vert, sh.frag = vert, frag
	sh.ok, sh.log = true, ""
	if !sh.reg {
		return nil // not registered yet; SetShader registers the new source
	}
	sh.gen++
	sh.name = fmt.Sprintf("mbuser_%d_r%d", id, sh.gen)
	sh.reg = false
	if err := w.registerUserShader(sh); err != nil {
		return err
	}
	for _, e := range w.ents {
		if e != nil && e.ushader == id && e.mat != nil {
			e.mat.SetShader(sh.name)
		}
	}
	return nil
}
