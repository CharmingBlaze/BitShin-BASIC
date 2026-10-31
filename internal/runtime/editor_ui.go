package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/AllenDang/cimgui-go/imguizmo"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

const (
	edLeftW   = 270
	edRightW  = 330
	edToolH   = 34
	edStatusH = 24
)

var edPanelFlags = imgui.WindowFlagsNoMove | imgui.WindowFlagsNoResize | imgui.WindowFlagsNoCollapse | imgui.WindowFlagsNoSavedSettings | imgui.WindowFlagsNoBringToFrontOnFocus

// addMenu lists what Add creates: label, command, and arguments.
var addMenu = []struct {
	label, cmd string
	args       []float64
}{
	{"Cube", "createcube", nil}, {"Sphere", "createsphere", nil}, {"Cylinder", "createcylinder", nil},
	{"Cone", "createcone", nil}, {"Plane", "createplane", []float64{10, 10}}, {"Quad", "createquad", nil},
	{"Disk", "createdisk", nil}, {"Capsule", "createcapsule", nil}, {"Torus", "createtorus", nil},
	{"Pyramid", "createpyramid", nil}, {"Wedge", "createwedge", nil}, {"Tube", "createtube", nil},
	{"", "", nil},
	{"Pivot", "createpivot", nil},
	{"", "", nil},
	{"Point Light", "createpointlight", nil}, {"Spot Light", "createspotlight", nil},
	{"Directional Light", "createdirectionallight", nil}, {"Area Light", "createarealight", []float64{2, 1}},
	{"", "", nil},
	{"Camera", "createcamera", nil},
}

func edNums(fs []float64) []value.Value {
	out := make([]value.Value, len(fs))
	for i, f := range fs {
		out[i] = value.Num(f)
	}
	return out
}

func (ed *editor) drawUI() {
	sw, sh := ed.w.screenSize2D()
	menuH := ed.menuBar()
	ed.toolbar(menuH, float32(sw))
	top := menuH + edToolH
	ed.hierarchy(top, float32(sh)-top-edStatusH)
	ed.inspector(float32(sw), top, float32(sh)-top-edStatusH)
	ed.statusBar(float32(sw), float32(sh))
	ed.popups()
	ed.gizmo(float32(sw), float32(sh))
}

func (ed *editor) menuBar() float32 {
	h := imgui.FrameHeight()
	if !imgui.BeginMainMenuBar() {
		return h
	}
	if imgui.BeginMenu("File") {
		if imgui.MenuItemBoolV("New Scene", "", false, true) {
			ed.newScene()
		}
		if imgui.MenuItemBoolV("Open...", "", false, true) {
			ed.showOpen = true
		}
		if imgui.MenuItemBoolV("Save", "Ctrl+S", false, true) {
			ed.save(ed.path)
		}
		if imgui.MenuItemBoolV("Save As...", "", false, true) {
			ed.saveAs = strings.TrimSuffix(filepath.Base(ed.path), ".scene.json")
			ed.showSaveAs = true
		}
		imgui.Separator()
		if imgui.MenuItemBoolV("Exit", "", false, true) {
			ed.quit = true
		}
		imgui.EndMenu()
	}
	if imgui.BeginMenu("Edit") {
		if imgui.MenuItemBoolV("Undo", "Ctrl+Z", false, len(ed.undo) > 0) {
			ed.undoStep()
		}
		if imgui.MenuItemBoolV("Redo", "Ctrl+Y", false, len(ed.redo) > 0) {
			ed.redoStep()
		}
		imgui.Separator()
		if imgui.MenuItemBoolV("Duplicate", "Ctrl+D", false, len(ed.sel) > 0) {
			ed.duplicate()
		}
		if imgui.MenuItemBoolV("Delete", "Del", false, len(ed.sel) > 0) {
			ed.deleteSelected()
		}
		if imgui.MenuItemBoolV("Focus", "F", false, len(ed.sel) > 0) {
			ed.focus()
		}
		if imgui.MenuItemBoolV("Select None", "Esc", false, len(ed.sel) > 0) {
			ed.sel = nil
		}
		imgui.EndMenu()
	}
	if imgui.BeginMenu("Add") {
		for _, it := range addMenu {
			if it.label == "" {
				imgui.Separator()
				continue
			}
			if imgui.MenuItemBool(it.label) {
				ed.create(it.label, it.cmd, edNums(it.args)...)
			}
		}
		if imgui.BeginMenu("Mesh from assets") {
			n := 0
			for _, a := range ed.assetFiles() {
				if isMeshFile(a) {
					n++
					if imgui.MenuItemBool(a) {
						ed.create(strings.TrimSuffix(filepath.Base(a), filepath.Ext(a)), "loadmesh", value.Str(a))
					}
				}
			}
			if n == 0 {
				imgui.TextDisabled("No meshes in assets/")
			}
			imgui.EndMenu()
		}
		imgui.EndMenu()
	}
	if imgui.BeginMenu("View") {
		if imgui.MenuItemBoolV("Editor light", "", ed.useHeadlight, true) {
			ed.useHeadlight = !ed.useHeadlight
		}
		if imgui.MenuItemBoolV("Grid", "", ed.grid, true) {
			ed.grid = !ed.grid
		}
		for _, d := range []struct{ label, what string }{{"Physics shapes", "physics"}, {"Light ranges", "lights"}, {"Bounds", "bounds"}, {"Entity IDs", "ids"}} {
			on := ed.w.debugState().on(dbgOverlayMust(d.what))
			if imgui.MenuItemBoolV(d.label, "", on, true) {
				ed.w.Call("debugdraw", []value.Value{value.Str(d.what), value.Num(float64(b01(!on)))})
			}
		}
		imgui.EndMenu()
	}
	if imgui.MenuItemBoolV("Play", "F5", false, true) {
		ed.play()
	}
	h = imgui.WindowHeight()
	imgui.EndMainMenuBar()
	return h
}

func dbgOverlayMust(name string) uint32 {
	b, _ := dbgOverlayBit(name)
	return b
}

func isMeshFile(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".glb", ".gltf", ".obj", ".fbx", ".dae":
		return true
	}
	return false
}

func (ed *editor) toolbar(y, width float32) {
	imgui.SetNextWindowPosV(imgui.NewVec2(0, y), imgui.CondAlways, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(width, edToolH), imgui.CondAlways)
	if imgui.BeginV("##toolbar", nil, edPanelFlags|imgui.WindowFlagsNoTitleBar) {
		for _, m := range []struct {
			label string
			op    imguizmo.OPERATION
		}{{"Move (W)", imguizmo.TRANSLATE}, {"Rotate (E)", imguizmo.ROTATE}, {"Scale (R)", imguizmo.SCALE}} {
			if imgui.RadioButtonBool(m.label, ed.op == m.op) {
				ed.op = m.op
			}
			imgui.SameLine()
		}
		imgui.TextDisabled("|")
		imgui.SameLine()
		local := ed.mode == imguizmo.LOCAL
		if imgui.Checkbox("Local", &local) {
			ed.mode = imguizmo.WORLD
			if local {
				ed.mode = imguizmo.LOCAL
			}
		}
		imgui.SameLine()
		imgui.Checkbox("Snap", &ed.snap)
		imgui.SameLine()
		imgui.SetNextItemWidth(60)
		imgui.DragFloatV("##snapmove", &ed.snapMove, 0.05, 0.01, 100, "%.2f m", imgui.SliderFlagsNone)
		imgui.SameLine()
		imgui.SetNextItemWidth(60)
		imgui.DragFloatV("##snaprot", &ed.snapRot, 1, 1, 90, "%.0f deg", imgui.SliderFlagsNone)
		imgui.SameLine()
		imgui.SetNextItemWidth(60)
		imgui.DragFloatV("##snapscale", &ed.snapScale, 0.01, 0.01, 10, "%.2f x", imgui.SliderFlagsNone)
		imgui.SameLine()
		imgui.TextDisabled("|  right drag orbit  middle drag pan  wheel zoom  F focus")
	}
	imgui.End()
}

// --- hierarchy ---

func (ed *editor) hierarchy(y, h float32) {
	imgui.SetNextWindowPosV(imgui.NewVec2(0, y), imgui.CondAlways, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(edLeftW, h*0.62), imgui.CondAlways)
	if imgui.BeginV("Hierarchy", nil, edPanelFlags) {
		imgui.SetNextItemWidth(-1)
		imgui.InputTextWithHint("##filter", "Search", &ed.filter, imgui.InputTextFlagsNone, nil)
		kids := map[int][]int{}
		for id, e := range ed.w.ents {
			if e == nil {
				continue
			}
			p := e.parent
			if ed.w.ents[p] == nil {
				p = 0
			}
			kids[p] = append(kids[p], id)
		}
		for _, l := range kids {
			sort.Ints(l)
		}
		if f := strings.ToLower(strings.TrimSpace(ed.filter)); f != "" {
			var ids []int
			for id, e := range ed.w.ents {
				if e != nil && strings.Contains(strings.ToLower(ed.label(id)), f) {
					ids = append(ids, id)
				}
			}
			sort.Ints(ids)
			for _, id := range ids {
				if imgui.SelectableBoolV(ed.label(id)+fmt.Sprintf("##f%d", id), ed.selected(id), 0, imgui.NewVec2(0, 0)) {
					ed.selectOnly(id, imgui.CurrentIO().KeyCtrl())
				}
			}
		} else {
			for _, id := range kids[0] {
				ed.treeNode(id, kids)
			}
		}
	}
	imgui.End()
	ed.assetPanel(y+h*0.62, h*0.38)
}

func (ed *editor) label(id int) string {
	e := ed.w.ents[id]
	if e == nil {
		return ""
	}
	name := e.name
	if name == "" {
		name = fmt.Sprintf("%s %d", strings.Title(ed.kindOf(e)), id)
	}
	return name
}

func (ed *editor) kindOf(e *Entity) string {
	switch {
	case e.cam != nil:
		return "camera"
	case e.lgtKind != 0:
		return "light"
	case e.kind != "":
		return e.kind
	case e.mesh != nil:
		return "mesh"
	}
	return "pivot"
}

func (ed *editor) treeNode(id int, kids map[int][]int) {
	flags := imgui.TreeNodeFlagsOpenOnArrow | imgui.TreeNodeFlagsSpanAvailWidth | imgui.TreeNodeFlagsDefaultOpen
	if len(kids[id]) == 0 {
		flags |= imgui.TreeNodeFlagsLeaf
	}
	if ed.selected(id) {
		flags |= imgui.TreeNodeFlagsSelected
	}
	open := imgui.TreeNodeExStrV(fmt.Sprintf("%s##n%d", ed.label(id), id), flags)
	if imgui.IsItemClicked() && !imgui.IsItemToggledOpen() {
		ed.selectOnly(id, imgui.CurrentIO().KeyCtrl())
	}
	if imgui.BeginPopupContextItem() {
		ed.entityMenu(id)
		imgui.EndPopup()
	}
	if open {
		for _, c := range kids[id] {
			ed.treeNode(c, kids)
		}
		imgui.TreePop()
	}
}

func (ed *editor) entityMenu(id int) {
	if !ed.selected(id) {
		ed.sel = []int{id}
	}
	if imgui.MenuItemBool("Focus") {
		ed.focus()
	}
	if imgui.MenuItemBool("Duplicate") {
		ed.duplicate()
	}
	if imgui.MenuItemBool("Delete") {
		ed.deleteSelected()
	}
	imgui.Separator()
	if p := ed.w.ents[id]; p != nil && p.parent != 0 && imgui.MenuItemBool("Unparent") {
		ed.reparent(id, 0)
	}
	if imgui.BeginMenu("Parent to") {
		ids := make([]int, 0, len(ed.w.ents))
		for oid := range ed.w.ents {
			if oid != id {
				ids = append(ids, oid)
			}
		}
		sort.Ints(ids)
		for _, oid := range ids {
			if imgui.MenuItemBool(fmt.Sprintf("%s##p%d", ed.label(oid), oid)) {
				ed.reparent(id, oid)
			}
		}
		imgui.EndMenu()
	}
}

func (ed *editor) assetPanel(y, h float32) {
	imgui.SetNextWindowPosV(imgui.NewVec2(0, y), imgui.CondAlways, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(edLeftW, h), imgui.CondAlways)
	if imgui.BeginV("Assets", nil, edPanelFlags) {
		files := ed.assetFiles()
		if len(files) == 0 {
			imgui.TextDisabled("Put meshes and images in assets/")
		}
		for _, a := range files {
			if imgui.SelectableBoolV(a, false, imgui.SelectableFlagsAllowDoubleClick, imgui.NewVec2(0, 0)) && imgui.IsMouseDoubleClicked(imgui.MouseButtonLeft) {
				if isMeshFile(a) {
					ed.create(strings.TrimSuffix(filepath.Base(a), filepath.Ext(a)), "loadmesh", value.Str(a))
				} else {
					ed.applyTexture(a)
				}
			}
			if imgui.IsItemHovered() {
				if isMeshFile(a) {
					imgui.SetTooltip("Double-click to add to the scene")
				} else {
					imgui.SetTooltip("Double-click to use as the selected object's texture")
				}
			}
		}
	}
	imgui.End()
}

func (ed *editor) applyTexture(path string) {
	id := ed.primary()
	if id == 0 {
		ed.say("Select an object first")
		return
	}
	tex, err := ed.w.Call("loadtexture", []value.Value{value.Str(path)})
	if err != nil {
		ed.say("Texture: " + err.Error())
		return
	}
	if _, err := ed.w.Call("entitytexture", []value.Value{value.Num(float64(id)), tex}); err != nil {
		ed.say("Texture: " + err.Error())
		return
	}
	ed.dirty = true
	ed.say("Texture " + filepath.Base(path) + " on " + ed.label(id))
}

// --- inspector ---

// track records an undo step when the widget just drawn finished an edit.
// before was captured while no widget was active.
func (ed *editor) track(label string, id int) {
	if imgui.IsItemDeactivatedAfterEdit() && ed.inspectBefore != nil {
		ed.record(label, ed.inspectBefore, ed.snapshot([]int{id}))
		ed.inspectBefore = ed.snapshot([]int{id})
	}
}

func (ed *editor) inspector(sw, y, h float32) {
	imgui.SetNextWindowPosV(imgui.NewVec2(sw-edRightW, y), imgui.CondAlways, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(edRightW, h), imgui.CondAlways)
	if imgui.BeginV("Inspector", nil, edPanelFlags) {
		id := ed.primary()
		e := ed.w.ents[id]
		if e == nil {
			imgui.TextDisabled("Nothing selected")
			imgui.TextDisabled("Click an object, or use Add")
			imgui.End()
			return
		}
		if !imgui.IsAnyItemActive() {
			ed.inspectBefore = ed.snapshot([]int{id})
		}
		w := ed.w
		n := e.node.GetNode()
		imgui.TextUnformatted(fmt.Sprintf("%s  #%d", strings.Title(ed.kindOf(e)), id))
		name := e.name
		if imgui.InputTextWithHint("Name", ed.label(id), &name, imgui.InputTextFlagsNone, nil) {
			e.name = name
			n.SetName(name)
		}
		ed.track("rename", id)
		vis := n.Visible()
		if imgui.Checkbox("Visible", &vis) {
			n.SetVisible(vis)
		}
		ed.track("visibility", id)

		imgui.SeparatorText("Transform")
		p := n.Position()
		bx, by, bz := fromG3N(p.X, p.Y, p.Z)
		pos := [3]float32{bx, by, bz}
		if imgui.DragFloat3V("Position", &pos, 0.05, 0, 0, "%.2f", imgui.SliderFlagsNone) {
			gx, gy, gz := toG3N(pos[0], pos[1], pos[2])
			n.SetPosition(gx, gy, gz)
		}
		ed.track("move", id)
		rot := [3]float32{e.pitch, e.yaw, e.roll}
		if imgui.DragFloat3V("Rotation", &rot, 0.5, 0, 0, "%.1f", imgui.SliderFlagsNone) {
			e.pitch, e.yaw, e.roll = rot[0], rot[1], rot[2]
			w.applyRot(e)
		}
		ed.track("rotate", id)
		s := n.Scale()
		scale := [3]float32{s.X, s.Y, s.Z}
		if imgui.DragFloat3V("Scale", &scale, 0.02, 0.001, 1000, "%.3f", imgui.SliderFlagsNone) {
			n.SetScale(scale[0], scale[1], scale[2])
		}
		ed.track("scale", id)

		if e.mesh != nil {
			ed.materialSection(id, e)
		}
		if e.lgtKind != 0 {
			ed.lightSection(id)
		}
		if al := w.areaLights[id]; al != nil {
			imgui.SeparatorText("Area light")
			size := [3]float32{al.width, al.height, 0}
			if imgui.DragFloat3V("Size (w, h)", &size, 0.05, 0.05, 100, "%.2f", imgui.SliderFlagsNone) {
				w.Call("setarealightsize", edNums([]float64{float64(id), float64(size[0]), float64(size[1])}))
			}
			col := [3]float32{al.color.R, al.color.G, al.color.B}
			if imgui.ColorEdit3V("Colour", &col, imgui.ColorEditFlagsNone) {
				w.Call("setarealight", edNums([]float64{float64(id), float64(col[0]), float64(col[1]), float64(col[2])}))
			}
			if imgui.DragFloatV("Intensity", &al.intensity, 0.05, 0, 100, "%.2f", imgui.SliderFlagsNone) {
				al.syncPanel()
			}
		}
		if e.cam != nil {
			imgui.SeparatorText("Camera")
			fov := e.cam.Fov()
			if imgui.DragFloatV("Field of view", &fov, 0.2, 5, 170, "%.1f", imgui.SliderFlagsNone) {
				e.cam.SetFov(fov)
			}
		}
		ed.dirtyIfEdited()
	}
	imgui.End()
}

func (ed *editor) dirtyIfEdited() {
	if imgui.IsAnyItemActive() {
		ed.dirty = true
	}
}

func (ed *editor) materialSection(id int, e *Entity) {
	w := ed.w
	imgui.SeparatorText("Material")
	col := [4]float32{e.tint.R, e.tint.G, e.tint.B, e.tint.A}
	if imgui.ColorEdit4V("Colour", &col, imgui.ColorEditFlagsNone) {
		w.setEntityRGB(e, col[0]*255, col[1]*255, col[2]*255)
		w.Call("entityalpha", edNums([]float64{float64(id), float64(col[3])}))
	}
	ed.track("colour", id)
	pbr := e.pbrWrap != nil
	if imgui.Checkbox("PBR", &pbr) && pbr {
		w.Call("setmaterialpbr", edNums([]float64{float64(id), 1}))
	}
	ed.track("material", id)
	if pm := e.pbrWrap; pm != nil {
		set := func() {
			w.eachEntityPBR(e, func(m *pbrMat) {
				m.metallic, m.roughness, m.emissive, m.shading = pm.metallic, pm.roughness, pm.emissive, pm.shading
				m.applyFactors()
			})
		}
		if imgui.SliderFloatV("Metallic", &pm.metallic, 0, 1, "%.2f", imgui.SliderFlagsNone) {
			set()
		}
		ed.track("metallic", id)
		if imgui.SliderFloatV("Roughness", &pm.roughness, 0, 1, "%.2f", imgui.SliderFlagsNone) {
			set()
		}
		ed.track("roughness", id)
		em := [3]float32{pm.emissive.R, pm.emissive.G, pm.emissive.B}
		if imgui.ColorEdit3V("Emissive", &em, imgui.ColorEditFlagsNone) {
			pm.emissive = math32.Color{R: em[0], G: em[1], B: em[2]}
			set()
		}
		ed.track("emissive", id)
		shading := int32(pm.shading)
		if imgui.ComboStr("Shading", &shading, "PBR\x00Unlit\x00Toon\x00") {
			if int(shading) == shadeToon && pm.toonBands <= 0 {
				pm.setToon(3, 0.5, 0.6)
			}
			pm.shading = int(shading)
			set()
		}
		ed.track("shading", id)
	}
}

func (ed *editor) lightSection(id int) {
	w := ed.w
	imgui.SeparatorText("Light")
	get := func(cmd string) float32 {
		v, _ := w.Call(cmd, edNums([]float64{float64(id)}))
		return float32(v.Num)
	}
	col := [3]float32{get("getlightred") / 255, get("getlightgreen") / 255, get("getlightblue") / 255}
	if imgui.ColorEdit3V("Colour", &col, imgui.ColorEditFlagsNone) {
		w.Call("lightcolor", edNums([]float64{float64(id), float64(col[0] * 255), float64(col[1] * 255), float64(col[2] * 255)}))
	}
	ed.track("light colour", id)
	in := get("getlightintensity")
	if imgui.DragFloatV("Intensity", &in, 0.02, 0, 50, "%.2f", imgui.SliderFlagsNone) {
		w.Call("setlightintensity", edNums([]float64{float64(id), float64(in)}))
	}
	ed.track("light intensity", id)
	if kind := get("getlighttype"); kind != 1 {
		rng := get("getlightrange")
		if imgui.DragFloatV("Range", &rng, 0.1, 0, 10000, "%.1f", imgui.SliderFlagsNone) {
			w.Call("lightrange", edNums([]float64{float64(id), float64(rng)}))
		}
		ed.track("light range", id)
	}
}

// --- status, popups ---

func (ed *editor) statusBar(sw, sh float32) {
	imgui.SetNextWindowPosV(imgui.NewVec2(0, sh-edStatusH), imgui.CondAlways, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(sw, edStatusH), imgui.CondAlways)
	if imgui.BeginV("##status", nil, edPanelFlags|imgui.WindowFlagsNoTitleBar) {
		mark := ""
		if ed.dirty {
			mark = " *"
		}
		fps := 0.0
		if ed.w.delta > 0 {
			fps = 1 / ed.w.delta
		}
		line := fmt.Sprintf("%s%s   %d objects   %d selected   %.0f fps", ed.path, mark, len(ed.w.ents), len(ed.sel), fps)
		if time.Since(ed.statusAt) < 6*time.Second {
			line += "   |   " + ed.status
		}
		imgui.TextUnformatted(line)
	}
	imgui.End()
}

func (ed *editor) popups() {
	if ed.showOpen {
		imgui.OpenPopupStr("Open scene")
		ed.showOpen = false
	}
	if ed.showSaveAs {
		imgui.OpenPopupStr("Save scene as")
		ed.showSaveAs = false
	}
	if imgui.BeginPopupModalV("Open scene", nil, 0) {
		files := ed.sceneFiles()
		if len(files) == 0 {
			imgui.TextDisabled("No .scene.json files in the project")
		}
		for _, f := range files {
			rel, _ := filepath.Rel(ed.project, f)
			if imgui.SelectableBool(filepath.ToSlash(rel)) {
				if ed.dirty {
					ed.save(ed.path)
				}
				ed.clearScene()
				if err := ed.load(f); err != nil {
					ed.say("Open failed: " + err.Error())
				}
				imgui.CloseCurrentPopup()
			}
		}
		if imgui.Button("Cancel") {
			imgui.CloseCurrentPopup()
		}
		imgui.EndPopup()
	}
	if imgui.BeginPopupModalV("Save scene as", nil, 0) {
		imgui.InputTextWithHint("Name", "level1", &ed.saveAs, imgui.InputTextFlagsNone, nil)
		if imgui.Button("Save") && strings.TrimSpace(ed.saveAs) != "" {
			name := strings.TrimSpace(ed.saveAs)
			if !strings.HasSuffix(name, ".scene.json") {
				name += ".scene.json"
			}
			dir := filepath.Dir(ed.path)
			ed.save(filepath.Join(dir, name))
			imgui.CloseCurrentPopup()
		}
		imgui.SameLine()
		if imgui.Button("Cancel") {
			imgui.CloseCurrentPopup()
		}
		imgui.EndPopup()
	}
}

func (ed *editor) clearScene() {
	for id := range ed.w.ents {
		ed.softDelete(id)
	}
	ed.trash = map[int]*editTrash{}
	ed.undo, ed.redo, ed.sel = nil, nil, nil
}

func (ed *editor) newScene() {
	if ed.dirty {
		ed.save(ed.path)
	}
	ed.clearScene()
	dir := filepath.Join(ed.project, "scenes")
	for i := 1; ; i++ {
		p := filepath.Join(dir, fmt.Sprintf("scene%d.scene.json", i))
		if !fileExists(p) {
			ed.path = p
			break
		}
	}
	ed.dirty = true
	ed.say("New scene " + filepath.Base(ed.path))
}

// --- gizmo and overlays ---

func (ed *editor) gizmo(sw, sh float32) {
	id := ed.primary()
	e := ed.w.ents[id]
	using := false
	if e != nil && e.node != nil {
		imguizmo.SetOrthographic(false)
		imguizmo.SetRect(0, 0, sw, sh)
		var view, proj math32.Matrix4
		ed.cam.ViewMatrix(&view)
		ed.cam.ProjMatrix(&proj)
		n := e.node.GetNode()
		n.UpdateMatrixWorld()
		m := n.MatrixWorld()
		var snap [3]float32
		var snapPtr *float32
		if ed.snap {
			switch ed.op {
			case imguizmo.TRANSLATE:
				snap = [3]float32{ed.snapMove, ed.snapMove, ed.snapMove}
			case imguizmo.ROTATE:
				snap = [3]float32{ed.snapRot, ed.snapRot, ed.snapRot}
			default:
				snap = [3]float32{ed.snapScale, ed.snapScale, ed.snapScale}
			}
			snapPtr = &snap[0]
		}
		if imguizmo.ManipulateV(&view[0], &proj[0], ed.op, ed.mode, &m[0], nil, snapPtr, nil, nil) {
			ed.setWorldMatrix(e, m)
		}
		using = imguizmo.IsUsing()
	}
	// One undo step per drag.
	if using && !ed.gizmoWas {
		ed.gizmoBefore = ed.snapshot([]int{id})
	}
	if !using && ed.gizmoWas && ed.gizmoBefore != nil {
		ed.record(map[imguizmo.OPERATION]string{imguizmo.TRANSLATE: "move", imguizmo.ROTATE: "rotate", imguizmo.SCALE: "scale"}[ed.op], ed.gizmoBefore, ed.snapshot([]int{ed.gizmoBefore[0].id}))
		ed.gizmoBefore = nil
	}
	ed.gizmoWas = using
}

// setWorldMatrix puts the gizmo's world matrix back on the node, in its
// parent's space, and keeps Blitz pitch/yaw/roll in step.
func (ed *editor) setWorldMatrix(e *Entity, world math32.Matrix4) {
	n := e.node.GetNode()
	local := world
	if p := n.Parent(); p != nil {
		p.GetNode().UpdateMatrixWorld()
		pw := p.GetNode().MatrixWorld()
		var inv math32.Matrix4
		if inv.GetInverse(&pw) == nil {
			local.MultiplyMatrices(&inv, &world)
		}
	}
	var pos, scale math32.Vector3
	var q math32.Quaternion
	local.Decompose(&pos, &q, &scale)
	n.SetPositionVec(&pos)
	n.SetQuaternionQuat(&q)
	n.SetScaleVec(&scale)
	ed.w.syncEulerFromNode(e)
	ed.dirty = true
}

// overlays draws the grid, selection boxes, and markers for things with no
// mesh (lights, pivots, cameras) through the debug-draw layer.
func (ed *editor) overlays() {
	w := ed.w
	w.debugUserShape(0, func(b *lineBatch) {
		if ed.grid {
			step := float32(1)
			for ed.dist/step > 40 {
				step *= 10
			}
			half := step * 20
			cx := math32.Floor(ed.target.X/step) * step
			cz := math32.Floor(ed.target.Z/step) * step
			minor := dbgColor{0.32, 0.34, 0.4, 0.55}
			for i := -20; i <= 20; i++ {
				o := float32(i) * step
				c := minor
				if cx+o == 0 {
					c = dbgColor{0.35, 0.45, 0.95, 0.9} // world Z axis
				}
				b.line(math32.Vector3{X: cx + o, Z: cz - half}, math32.Vector3{X: cx + o, Z: cz + half}, c)
				c = minor
				if cz+o == 0 {
					c = dbgColor{0.95, 0.35, 0.35, 0.9} // world X axis
				}
				b.line(math32.Vector3{X: cx - half, Z: cz + o}, math32.Vector3{X: cx + half, Z: cz + o}, c)
			}
		}
		for id, e := range w.ents {
			if e == nil || e.node == nil {
				continue
			}
			sel := ed.selected(id)
			if box, ok := ed.worldBox(id); ok {
				if sel {
					var c [8]math32.Vector3
					for i := 0; i < 8; i++ {
						c[i] = math32.Vector3{X: pick3(i&1 != 0, box.Max.X, box.Min.X), Y: pick3(i&2 != 0, box.Max.Y, box.Min.Y), Z: pick3(i&4 != 0, box.Max.Z, box.Min.Z)}
					}
					b.boxCorners(c, dbgColor{1, 0.62, 0.15, 1})
				}
				continue
			}
			// No mesh: a small cross so it can be seen and clicked.
			p := worldPos(e.node.GetNode())
			s := ed.dist * 0.02
			c := dbgColor{0.8, 0.8, 0.85, 1}
			if e.lgtKind != 0 {
				c = dbgColor{1, 0.9, 0.4, 1}
			}
			if sel {
				c = dbgColor{1, 0.62, 0.15, 1}
			}
			for _, d := range []math32.Vector3{{X: s}, {Y: s}, {Z: s}} {
				a, q := p, p
				a.Sub(&d)
				q.Add(&d)
				b.line(a, q, c)
			}
		}
	})
}

func pick3(c bool, a, b float32) float32 {
	if c {
		return a
	}
	return b
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
