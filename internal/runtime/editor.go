package runtime

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/AllenDang/cimgui-go/imguizmo"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/experimental/collision"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"

	"moonbasic/internal/value"
)

// The scene editor: `moon edit file.scene.json`. It runs the real renderer
// with ImGui panels on top: a viewport with an orbit camera, a hierarchy,
// an inspector, an asset list, ImGuizmo handles, and undo/redo. Edits go
// through the same commands a script would call, so a scene looks the same
// in the editor and in the game.
type editor struct {
	w       *World
	path    string // scene file
	project string // project root (moon.json, assets)
	dirty   bool

	sel []int // selected entity ids; the last is the one the gizmo moves

	cam                 *camera.Camera
	target              math32.Vector3 // orbit centre, G3N space
	yaw, pitch, dist    float32
	lastMX, lastMY      float32
	dragging            bool
	pressX, pressY      float32
	headlight           *light.Directional
	useHeadlight, grid  bool
	op                  imguizmo.OPERATION
	mode                imguizmo.MODE
	snap                bool
	snapMove, snapRot   float32
	snapScale           float32
	gizmoWas            bool
	gizmoBefore         []editSnap
	undo, redo          []editStep
	trash               map[int]*editTrash
	prevKeys            map[int]bool
	filter              string
	status              string
	statusAt            time.Time
	assets              []string
	assetsAt            time.Time
	saveAs              string
	showSaveAs, showOpen bool
	added               map[string]int // per kind, for default names
	inspectBefore       []editSnap
	quit                bool
}

// editSnap is everything undo restores for one entity. kept is false for
// an entity that did not exist (created) or was deleted.
type editSnap struct {
	id    int
	kept  bool
	state map[string]any
}

type editStep struct {
	label         string
	before, after []editSnap
}

type editTrash struct {
	e      *Entity
	parent core.INode
}

// RunEditor opens path (created if missing) in the scene editor.
func RunEditor(path, project string) error {
	if project == "" {
		project = filepath.Dir(path)
	}
	w := New(project)
	w.title = "Moon Studio — " + filepath.Base(path)
	if _, err := w.graphics3D(1600, 900, 0, 2); err != nil {
		return err
	}
	ed := &editor{
		w: w, path: path, project: project, dist: 14, yaw: 35, pitch: 28,
		op: imguizmo.TRANSLATE, mode: imguizmo.WORLD, snapMove: 0.5, snapRot: 15, snapScale: 0.1,
		useHeadlight: true, grid: true, trash: map[int]*editTrash{}, prevKeys: map[int]bool{}, added: map[string]int{},
	}
	ed.cam = camera.New(16.0 / 9)
	ed.cam.SetFar(4000)
	ed.cam.SetNear(0.05)
	ed.headlight = light.NewDirectional(&math32.Color{R: 1, G: 0.97, B: 0.92}, 0.9)
	w.scene.Add(ed.headlight)
	if _, err := os.Stat(path); err == nil {
		if err := ed.load(path); err != nil {
			ed.say("Could not open " + filepath.Base(path) + ": " + err.Error())
		}
	} else {
		ed.dirty = true
		ed.say("New scene " + filepath.Base(path))
	}
	w.cam = ed.cam
	// MOON_EDITOR_SELECT=name selects an entity at start (screenshots, tests).
	if name := os.Getenv("MOON_EDITOR_SELECT"); name != "" {
		for id, e := range w.ents {
			if e != nil && strings.EqualFold(e.name, name) {
				ed.sel = []int{id}
				ed.focus()
			}
		}
	}
	w.app.Run(func(rend *renderer.Renderer, dt time.Duration) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("editor: panic", r)
			}
		}()
		w.tickDelta(dt)
		w.noteMouse()
		w.loopFrames++
		if w.windowWantsClose() || ed.quit {
			w.app.Exit()
			return
		}
		ed.frame()
		w.render(rend)
		w.markFlip()
	})
	return nil
}

func (ed *editor) say(s string) {
	ed.status, ed.statusAt = s, time.Now()
	fmt.Println(s)
}

// emit tells a hosting IDE what happened: one JSON object per line.
func emit(event string, kv ...any) {
	m := map[string]any{"event": event}
	for i := 0; i+1 < len(kv); i += 2 {
		m[fmt.Sprint(kv[i])] = kv[i+1]
	}
	b, _ := json.Marshal(m)
	fmt.Println("@editor " + string(b))
}

func (ed *editor) frame() {
	w := ed.w
	if err := w.guiEnsureFrame(); err != nil {
		return
	}
	imguizmo.BeginFrame()
	// Scene cameras must not take over the view.
	w.cam = ed.cam
	ed.cameraInput()
	ed.placeCamera()
	ed.shortcuts()
	ed.drawUI()
	ed.overlays()
}

// --- camera ---

func (ed *editor) placeCamera() {
	sw, sh := ed.w.screenSize2D()
	ed.cam.SetAspect(float32(sw) / float32(max(sh, 1)))
	yaw, pitch := ed.yaw*math32.Pi/180, ed.pitch*math32.Pi/180
	off := math32.Vector3{
		X: ed.dist * math32.Cos(pitch) * math32.Sin(yaw),
		Y: ed.dist * math32.Sin(pitch),
		Z: ed.dist * math32.Cos(pitch) * math32.Cos(yaw),
	}
	pos := ed.target
	pos.Add(&off)
	ed.cam.SetPositionVec(&pos)
	up := math32.Vector3{Y: 1}
	ed.cam.LookAt(&ed.target, &up)
	ed.cam.UpdateMatrixWorld()
	ed.headlight.SetVisible(ed.useHeadlight)
	ed.headlight.SetPositionVec(&off)
}

// cameraInput: right drag orbits, middle drag (or shift + right) pans,
// the wheel zooms, as in Blender and Godot.
func (ed *editor) cameraInput() {
	w := ed.w
	mx, my := w.mx, w.my
	dx, dy := mx-ed.lastMX, my-ed.lastMY
	ed.lastMX, ed.lastMY = mx, my
	if imgui.CurrentIO().WantCaptureMouse() && !ed.dragging {
		w.mz = 0
		return
	}
	right, middle := w.mouse[2], w.mouse[3]
	shift := w.keys[KeyLShift] || w.keys[KeyRShift]
	ed.dragging = right || middle
	switch {
	case middle || right && shift:
		ed.pan(dx, dy)
	case right:
		ed.yaw -= dx * 0.3
		ed.pitch = float32(math.Max(-89, math.Min(89, float64(ed.pitch+dy*0.3))))
	}
	if w.mz != 0 {
		ed.dist = float32(math.Max(0.2, float64(ed.dist)*math.Pow(0.88, float64(w.mz))))
		w.mz = 0
	}
}

func (ed *editor) pan(dx, dy float32) {
	m := ed.cam.MatrixWorld()
	var right, up math32.Vector3
	right.SetFromMatrixColumn(0, &m)
	up.SetFromMatrixColumn(1, &m)
	k := ed.dist * 0.0015
	right.MultiplyScalar(-dx * k)
	up.MultiplyScalar(dy * k)
	ed.target.Add(&right)
	ed.target.Add(&up)
}

// focus centres the orbit on the selection and backs off to fit it.
func (ed *editor) focus() {
	id := ed.primary()
	e := ed.w.ents[id]
	if e == nil {
		return
	}
	p := worldPos(e.node.GetNode())
	ed.target = p
	r := float32(2)
	if b, ok := ed.worldBox(id); ok {
		var size math32.Vector3
		b.Size(&size)
		r = math32.Max(size.Length(), 0.5)
	}
	ed.dist = r * 1.6
}

// --- selection and picking ---

func (ed *editor) primary() int {
	if len(ed.sel) == 0 {
		return 0
	}
	return ed.sel[len(ed.sel)-1]
}

func (ed *editor) selected(id int) bool {
	for _, s := range ed.sel {
		if s == id {
			return true
		}
	}
	return false
}

func (ed *editor) selectOnly(id int, add bool) {
	if id == 0 {
		if !add {
			ed.sel = nil
		}
		return
	}
	if add {
		for i, s := range ed.sel {
			if s == id {
				ed.sel = append(ed.sel[:i], ed.sel[i+1:]...)
				return
			}
		}
		ed.sel = append(ed.sel, id)
		return
	}
	ed.sel = []int{id}
}

// pick finds the entity under the mouse with a triangle-exact ray.
func (ed *editor) pick() int {
	w := ed.w
	sw, sh := w.screenSize2D()
	nx := float32(w.mx)/float32(sw)*2 - 1
	ny := 1 - float32(w.my)/float32(sh)*2
	rc := collision.NewRaycaster(&math32.Vector3{}, &math32.Vector3{})
	if err := rc.SetFromCamera(ed.cam, nx, ny); err != nil {
		return 0
	}
	owner := map[core.INode]int{}
	var roots []core.INode
	for id, e := range w.ents {
		if e == nil || e.node == nil || e.cam != nil || !e.node.GetNode().Visible() {
			continue
		}
		roots = append(roots, e.node)
		ed.walkOwner(e.node, id, owner)
	}
	hits := rc.IntersectObjects(roots, true)
	best, bestD := 0, float32(math.MaxFloat32)
	for _, h := range hits {
		for n := h.Object; n != nil; {
			if id, ok := owner[n]; ok {
				if h.Distance < bestD {
					best, bestD = id, h.Distance
				}
				break
			}
			p := n.GetNode().Parent()
			if p == nil {
				break
			}
			n = p
		}
	}
	if best != 0 {
		return best
	}
	// Lights, pivots, and cameras have no triangles: take the nearest one
	// whose position is within a few pixels of the mouse.
	bestPx := float32(14)
	for id, e := range w.ents {
		if e == nil || e.mesh != nil || e.node == nil || e.node.GetNode() == ed.cam.GetNode() {
			continue
		}
		p := worldPos(e.node.GetNode())
		if px, py, ok := ed.toScreen(p); ok {
			if d := math32.Sqrt((px-w.mx)*(px-w.mx) + (py-w.my)*(py-w.my)); d < bestPx {
				best, bestPx = id, d
			}
		}
	}
	return best
}

func (ed *editor) walkOwner(n core.INode, id int, owner map[core.INode]int) {
	if _, taken := owner[n]; !taken {
		owner[n] = id
	}
	for _, c := range n.GetNode().Children() {
		ed.walkOwner(c, id, owner)
	}
}

// toScreen maps a world point to window pixels.
func (ed *editor) toScreen(p math32.Vector3) (float32, float32, bool) {
	var view, proj math32.Matrix4
	ed.cam.ViewMatrix(&view)
	ed.cam.ProjMatrix(&proj)
	p.ApplyMatrix4(&view)
	if p.Z > 0 {
		return 0, 0, false
	}
	p.ApplyProjection(&proj)
	sw, sh := ed.w.screenSize2D()
	return (p.X + 1) / 2 * float32(sw), (1 - p.Y) / 2 * float32(sh), true
}

func (ed *editor) worldBox(id int) (math32.Box3, bool) {
	e := ed.w.ents[id]
	if e == nil || e.node == nil {
		return math32.Box3{}, false
	}
	var box math32.Box3
	box.MakeEmpty()
	found := false
	ed.w.walkMeshes(e.node, func(m *graphic.Mesh) {
		g := m.GetGeometry()
		if g == nil {
			return
		}
		b := g.BoundingBox()
		mw := m.MatrixWorld()
		b.ApplyMatrix4(&mw)
		box.Union(&b)
		found = true
	})
	return box, found
}

// --- shortcuts ---

func (ed *editor) hit(code int) bool {
	down := ed.w.keys[code]
	was := ed.prevKeys[code]
	ed.prevKeys[code] = down
	return down && !was
}

func (ed *editor) shortcuts() {
	w := ed.w
	ctrl := w.keys[KeyLControl] || w.keys[KeyRControl]
	shift := w.keys[KeyLShift] || w.keys[KeyRShift]
	// Read every key once a frame so edges are not missed.
	keys := map[int]bool{}
	for _, k := range []int{KeyW, KeyE, KeyR, KeyF, KeyZ, KeyY, KeyS, KeyD, KeyN, KeyO, KeyDelete, KeyF5, KeyEscape} {
		keys[k] = ed.hit(k)
	}
	if imgui.CurrentIO().WantCaptureKeyboard() {
		return
	}
	switch {
	case ctrl && keys[KeyZ] && shift, ctrl && keys[KeyY]:
		ed.redoStep()
	case ctrl && keys[KeyZ]:
		ed.undoStep()
	case ctrl && keys[KeyS]:
		ed.save(ed.path)
	case ctrl && keys[KeyD]:
		ed.duplicate()
	case ctrl:
	case keys[KeyW]:
		ed.op = imguizmo.TRANSLATE
	case keys[KeyE]:
		ed.op = imguizmo.ROTATE
	case keys[KeyR]:
		ed.op = imguizmo.SCALE
	case keys[KeyF]:
		ed.focus()
	case keys[KeyDelete]:
		ed.deleteSelected()
	case keys[KeyF5]:
		ed.play()
	case keys[KeyEscape]:
		ed.sel = nil
	}
	// Click to select, unless it was a drag or the gizmo or a panel took it.
	io := imgui.CurrentIO()
	if imgui.IsMouseClickedBool(imgui.MouseButtonLeft) && !io.WantCaptureMouse() && !imguizmo.IsOver() {
		ed.pressX, ed.pressY = w.mx, w.my
	}
	if imgui.IsMouseReleased(imgui.MouseButtonLeft) && !io.WantCaptureMouse() && !imguizmo.IsUsing() && !ed.gizmoWas {
		if math32.Abs(w.mx-ed.pressX)+math32.Abs(w.my-ed.pressY) < 4 {
			ed.selectOnly(ed.pick(), ctrl || shift)
		}
	}
}

// --- undo / redo ---

func (ed *editor) snapshot(ids []int) []editSnap {
	out := make([]editSnap, 0, len(ids))
	for _, id := range ids {
		if e := ed.w.ents[id]; e != nil {
			out = append(out, editSnap{id: id, kept: true, state: ed.w.editorState(id)})
		} else {
			out = append(out, editSnap{id: id})
		}
	}
	return out
}

func (ed *editor) record(label string, before, after []editSnap) {
	ed.undo = append(ed.undo, editStep{label, before, after})
	if len(ed.undo) > 200 {
		ed.undo = ed.undo[1:]
	}
	ed.redo = nil
	ed.dirty = true
}

func (ed *editor) apply(snaps []editSnap) {
	for _, s := range snaps {
		e := ed.w.ents[s.id]
		switch {
		case !s.kept && e != nil:
			ed.softDelete(s.id)
		case s.kept && e == nil:
			ed.restore(s.id)
			fallthrough
		case s.kept:
			ed.w.applyEditorState(s.id, s.state)
		}
	}
}

func (ed *editor) undoStep() {
	if len(ed.undo) == 0 {
		return
	}
	st := ed.undo[len(ed.undo)-1]
	ed.undo = ed.undo[:len(ed.undo)-1]
	ed.apply(st.before)
	ed.redo = append(ed.redo, st)
	ed.dirty = true
	ed.say("Undo " + st.label)
}

func (ed *editor) redoStep() {
	if len(ed.redo) == 0 {
		return
	}
	st := ed.redo[len(ed.redo)-1]
	ed.redo = ed.redo[:len(ed.redo)-1]
	ed.apply(st.after)
	ed.undo = append(ed.undo, st)
	ed.dirty = true
	ed.say("Redo " + st.label)
}

// softDelete takes an entity out of the world but keeps it, so undo can
// put back exactly what was there, whatever its kind.
func (ed *editor) softDelete(id int) {
	w := ed.w
	e := w.ents[id]
	if e == nil {
		return
	}
	for cid, c := range w.ents {
		if c != nil && c.parent == id {
			ed.softDelete(cid)
		}
	}
	var parent core.INode
	if p := e.node.GetNode().Parent(); p != nil {
		parent = p
		p.GetNode().Remove(e.node)
	}
	ed.trash[id] = &editTrash{e: e, parent: parent}
	delete(w.ents, id)
	for i, s := range ed.sel {
		if s == id {
			ed.sel = append(ed.sel[:i], ed.sel[i+1:]...)
			break
		}
	}
}

func (ed *editor) restore(id int) {
	t := ed.trash[id]
	if t == nil {
		return
	}
	ed.w.ents[id] = t.e
	if t.parent != nil {
		t.parent.GetNode().Add(t.e.node)
	}
	delete(ed.trash, id)
	for cid, c := range ed.trash {
		if c.e.parent == id {
			ed.restore(cid)
		}
	}
}

// subtree is id and every entity below it.
func (ed *editor) subtree(id int) []int {
	out := []int{id}
	for cid, c := range ed.w.ents {
		if c != nil && c.parent == id {
			out = append(out, ed.subtree(cid)...)
		}
	}
	return out
}

func (ed *editor) deleteSelected() {
	if len(ed.sel) == 0 {
		return
	}
	var ids []int
	for _, id := range ed.sel {
		ids = append(ids, ed.subtree(id)...)
	}
	before := ed.snapshot(ids)
	for _, id := range append([]int(nil), ed.sel...) {
		ed.softDelete(id)
	}
	ed.record("delete", before, ed.snapshot(ids))
	ed.sel = nil
}

// create runs a command that makes an entity, names it, puts it at the
// orbit centre, and records it for undo.
func (ed *editor) create(label, cmdName string, args ...value.Value) int {
	w := ed.w
	v, err := w.Call(cmdName, args)
	if err != nil {
		ed.say(label + ": " + err.Error())
		return 0
	}
	id := v.Int()
	e := w.ents[id]
	if e == nil {
		return 0
	}
	ed.added[label]++
	e.name = fmt.Sprintf("%s %d", label, ed.added[label])
	if e.cam == nil && e.lgtKind != 1 {
		e.node.GetNode().SetPositionVec(&ed.target)
	}
	ed.record("add "+strings.ToLower(label), []editSnap{{id: id}}, ed.snapshot([]int{id}))
	ed.sel = []int{id}
	return id
}

func (ed *editor) duplicate() {
	var created []int
	for _, id := range ed.sel {
		v, err := ed.w.Call("copyentity", []value.Value{value.Num(float64(id))})
		if err != nil {
			ed.say("Duplicate: " + err.Error())
			continue
		}
		nid := v.Int()
		if e, src := ed.w.ents[nid], ed.w.ents[id]; e != nil && src != nil {
			e.name = src.name + " copy"
		}
		created = append(created, nid)
	}
	if len(created) == 0 {
		return
	}
	before := make([]editSnap, len(created))
	for i, id := range created {
		before[i] = editSnap{id: id}
	}
	ed.record("duplicate", before, ed.snapshot(created))
	ed.sel = created
}

func (ed *editor) reparent(child, parent int) {
	if child == parent || child == 0 {
		return
	}
	for p := parent; p != 0; {
		if p == child {
			ed.say("Cannot parent an entity to its own child")
			return
		}
		pe := ed.w.ents[p]
		if pe == nil {
			break
		}
		p = pe.parent
	}
	before := ed.snapshot([]int{child})
	if err := ed.w.reparentKeepWorld(child, parent); err != nil {
		ed.say("Parent: " + err.Error())
		return
	}
	ed.record("parent", before, ed.snapshot([]int{child}))
}

// --- files ---

func (ed *editor) load(path string) error {
	w := ed.w
	if err := w.editorLoadScene(path); err != nil {
		return err
	}
	ed.path = path
	ed.dirty = false
	ed.undo, ed.redo, ed.sel = nil, nil, nil
	ed.say("Opened " + filepath.Base(path))
	return nil
}

func (ed *editor) save(path string) {
	if err := ed.w.editorSaveScene(path); err != nil {
		ed.say("Save failed: " + err.Error())
		return
	}
	ed.path = path
	ed.dirty = false
	ed.say("Saved " + filepath.Base(path))
	abs, _ := filepath.Abs(path)
	emit("saved", "file", abs)
}

// play saves, then asks the IDE to run the project. Without an IDE it runs
// the project's main script itself.
func (ed *editor) play() {
	ed.save(ed.path)
	if os.Getenv("MOON_IDE") != "" {
		emit("play")
		return
	}
	main := ed.mainScript()
	if main == "" {
		ed.say("Play: no main.mb in " + ed.project)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		ed.say("Play: " + err.Error())
		return
	}
	cmd := exec.Command(exe, main)
	cmd.Dir = ed.project
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		ed.say("Play: " + err.Error())
		return
	}
	ed.say("Playing " + filepath.Base(main))
	go cmd.Wait()
}

// mainScript reads moon.json's "main", else main.mb.
func (ed *editor) mainScript() string {
	main := "main.mb"
	if b, err := os.ReadFile(filepath.Join(ed.project, "moon.json")); err == nil {
		var cfg struct {
			Main string `json:"main"`
		}
		if json.Unmarshal(b, &cfg) == nil && cfg.Main != "" {
			main = cfg.Main
		}
	}
	p := filepath.Join(ed.project, main)
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

func (ed *editor) sceneFiles() []string {
	var out []string
	filepath.WalkDir(ed.project, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".scene.json") {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// assetFiles lists meshes and images under the project's assets folder.
func (ed *editor) assetFiles() []string {
	if time.Since(ed.assetsAt) < 2*time.Second && ed.assets != nil {
		return ed.assets
	}
	ed.assetsAt = time.Now()
	ed.assets = ed.assets[:0]
	root := filepath.Join(ed.project, "assets")
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(p)) {
		case ".glb", ".gltf", ".obj", ".fbx", ".dae", ".png", ".jpg", ".jpeg":
			rel, _ := filepath.Rel(ed.project, p)
			ed.assets = append(ed.assets, filepath.ToSlash(rel))
		}
		return nil
	})
	return ed.assets
}
