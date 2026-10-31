package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/phys3d"
	"moonbasic/internal/value"
)

// Scene format v2 (.scene.json) is what the scene editor reads and writes:
//
//	{"version": 2, "environment": {...}, "entities": [...]}
//
// Every entity has a stable uid kept across save and load. Transforms are
// local to the parent, in Blitz axes and degrees. A prefab is a scene file;
// an entity of kind "prefab" names one in src and lists per-uid property
// overrides. Instance members get uids "<instance uid>/<prefab uid>", so an
// override key is the member's uid inside the prefab ("." is the root).
// docs/SCENES.md is the format reference.

const (
	sceneFormatVersion = 2
	maxPrefabDepth     = 16
)

// entMeta is the scene data a script cannot read back from the engine.
type entMeta struct {
	uid      string
	tags     []string
	script   string
	inst     string      // uid of the prefab instance root this entity was built by
	prefab   *prefabInst // set on an instance root
	layer    int
	sensor   bool
	autoBody bool       // body shape "auto": sized from the mesh
	bodyAt   [7]float32 // world position and rotation when the body was built
	bodyAtOK bool
}

// prefabInst is an instance root's link to its prefab file. base is every
// member's entity JSON before this instance's overrides, so a save writes
// only what changed.
type prefabInst struct {
	src       string
	base      map[string]map[string]any
	overrides map[string]any // kept as read when the file could not be loaded
	broken    bool
}

type sceneState struct {
	uids   map[string]uidRef
	sky    string // SetSkyPreset name
	skybox string // CreateSkyBox / SetSky path prefix
	dir    string // folder of the scene file being loaded
}

type uidRef struct {
	id int
	e  *Entity
}

var scenePrimitiveKinds = map[string]bool{
	"cube": true, "sphere": true, "cylinder": true, "cone": true, "plane": true, "quad": true,
	"disk": true, "torus": true, "capsule": true, "pyramid": true, "wedge": true, "tube": true,
}

func newUID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}

func (w *World) metaOf(e *Entity) *entMeta {
	if e.meta == nil {
		e.meta = &entMeta{}
	}
	return e.meta
}

func (w *World) freshUID() string {
	for {
		if u := newUID(); w.entityByUID(u) == 0 {
			return u
		}
	}
}

func (w *World) setEntityUID(id int, uid string) {
	e := w.ents[id]
	if e == nil || uid == "" {
		return
	}
	m := w.metaOf(e)
	if m.uid != "" && m.uid != uid {
		if ref, ok := w.scn.uids[m.uid]; ok && ref.e == e {
			delete(w.scn.uids, m.uid)
		}
	}
	m.uid = uid
	if w.scn.uids == nil {
		w.scn.uids = map[string]uidRef{}
	}
	w.scn.uids[uid] = uidRef{id: id, e: e}
}

// entityUID is the entity's scene uid, made on first use.
func (w *World) entityUID(id int) string {
	e := w.ents[id]
	if e == nil {
		return ""
	}
	if e.meta == nil || e.meta.uid == "" {
		w.setEntityUID(id, w.freshUID())
	}
	return e.meta.uid
}

// entityByUID is the entity with that uid, or 0.
func (w *World) entityByUID(uid string) int {
	if uid == "" {
		return 0
	}
	ref, ok := w.scn.uids[uid]
	if !ok {
		return 0
	}
	if w.ents[ref.id] != ref.e || ref.e.meta == nil || ref.e.meta.uid != uid {
		delete(w.scn.uids, uid)
		return 0
	}
	return ref.id
}

// ---- small JSON helpers ----

func r4(v float32) float64 { return math.Round(float64(v)*1e4) / 1e4 }

func vecJSON(x, y, z float32) []any { return []any{r4(x), r4(y), r4(z)} }

func col255(c math32.Color) []any { return []any{r4(c.R * 255), r4(c.G * 255), r4(c.B * 255)} }

func numOf(v any, def float64) float64 {
	switch n := v.(type) {
	case nil:
		return def
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return def
		}
		return f
	case bool:
		if n {
			return 1
		}
		return 0
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		if err != nil {
			return def
		}
		return f
	}
	return def
}

func numList(v any) []float64 {
	switch n := v.(type) {
	case []any:
		out := make([]float64, len(n))
		for i, x := range n {
			out[i] = numOf(x, 0)
		}
		return out
	case []float64:
		return n
	case []float32:
		out := make([]float64, len(n))
		for i, x := range n {
			out[i] = float64(x)
		}
		return out
	}
	return nil
}

func vecOf(v any, def [3]float32) [3]float32 {
	l := numList(v)
	for i := 0; i < 3 && i < len(l); i++ {
		def[i] = float32(l[i])
	}
	return def
}

// colorOf reads [r, g, b(, a)] in 0..255 as 0..1.
func colorOf(v any) (math32.Color4, bool) {
	l := numList(v)
	if len(l) < 3 {
		return math32.Color4{}, false
	}
	c := math32.Color4{R: float32(l[0] / 255), G: float32(l[1] / 255), B: float32(l[2] / 255), A: 1}
	if len(l) >= 4 {
		c.A = float32(l[3] / 255)
	}
	return c, true
}

func boolOf(v any, def bool) bool {
	switch b := v.(type) {
	case bool:
		return b
	case nil:
		return def
	}
	return numOf(v, 0) != 0
}

func strOf(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case nil:
		return ""
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

func mapOf(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func tagList(v any) []string {
	var out []string
	add := func(s string) {
		for _, part := range strings.Split(s, ",") {
			if t := strings.TrimSpace(part); t != "" {
				out = append(out, t)
			}
		}
	}
	switch t := v.(type) {
	case string:
		add(t)
	case []any:
		for _, x := range t {
			add(strOf(x))
		}
	case []string:
		for _, x := range t {
			add(x)
		}
	}
	return out
}

// jsonNorm turns Go values into what json.Unmarshal would give back.
func jsonNorm(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if json.Unmarshal(b, &out) != nil {
		return v
	}
	return out
}

// jsonNear is deep equality after a JSON round trip, with numbers equal to
// within float32 noise.
func jsonNear(a, b any) bool {
	return nearNorm(jsonNorm(a), jsonNorm(b))
}

func nearNorm(a, b any) bool {
	switch x := a.(type) {
	case float64:
		y, ok := b.(float64)
		if !ok {
			return false
		}
		tol := 1e-3 * math.Max(1, math.Abs(x))
		return math.Abs(x-y) <= tol
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, v := range x {
			if !nearNorm(v, y[k]) {
				return false
			}
		}
		return true
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !nearNorm(x[i], y[i]) {
				return false
			}
		}
		return true
	}
	return a == b
}

// ---- reading entities ----

// sceneKind is the v2 kind of an entity, or "" when the format cannot hold it.
func (w *World) sceneKind(id int, e *Entity) string {
	if e == nil || e.node == nil || e.sky {
		return ""
	}
	if al := w.areaLights[id]; al != nil && al.e == e {
		return "arealight"
	}
	if e.cam != nil {
		return "camera"
	}
	if e.lgt != nil {
		return "light"
	}
	k := strings.ToLower(e.kind)
	switch {
	case k == "box":
		return "cube"
	case scenePrimitiveKinds[k]:
		return k
	case k == "mesh" || k == "skin":
		if e.src != "" {
			return "mesh"
		}
	case k == "pivot":
		return "pivot"
	case k == "" && e.mesh == nil:
		if _, ok := e.node.(*core.Node); ok {
			return "pivot"
		}
	}
	return ""
}

func isMeshKind(kind string) bool { return scenePrimitiveKinds[kind] || kind == "mesh" }

// entityJSON is one entity in scene format v2: an instance root as its
// prefab reference plus overrides, anything else as itself. nil when the
// format cannot hold the entity (terrain, water, particles, ...).
func (w *World) entityJSON(id int) map[string]any {
	e := w.ents[id]
	if e == nil {
		return nil
	}
	if e.meta != nil && e.meta.prefab != nil {
		return w.prefabRowJSON(id)
	}
	return w.rawEntityJSON(id)
}

// rawEntityJSON describes the entity itself, even an instance root.
func (w *World) rawEntityJSON(id int) map[string]any {
	e := w.ents[id]
	kind := w.sceneKind(id, e)
	if kind == "" {
		return nil
	}
	n := e.node.GetNode()
	p := n.Position()
	x, y, z := fromG3N(p.X, p.Y, p.Z)
	s := n.Scale()
	row := map[string]any{
		"uid":     w.entityUID(id),
		"name":    e.name,
		"kind":    kind,
		"pos":     vecJSON(x, y, z),
		"rot":     vecJSON(e.pitch, e.yaw, e.roll),
		"scale":   vecJSON(s.X, s.Y, s.Z),
		"visible": n.Visible(),
	}
	if kind == "mesh" {
		row["src"] = e.src
	}
	if e.parent != 0 && w.ents[e.parent] != nil {
		row["parent"] = w.entityUID(e.parent)
	}
	if m := e.meta; m != nil {
		if len(m.tags) > 0 {
			tags := make([]any, len(m.tags))
			for i, t := range m.tags {
				tags[i] = t
			}
			row["tags"] = tags
		}
		if m.script != "" {
			row["script"] = m.script
		}
	}
	switch kind {
	case "light":
		row["light"] = w.lightJSON(e)
	case "arealight":
		row["area"] = areaJSON(w.areaLights[id])
	case "camera":
		row["camera"] = w.cameraJSON(e)
	}
	if isMeshKind(kind) {
		t := e.tint
		row["color"] = []any{r4(t.R * 255), r4(t.G * 255), r4(t.B * 255), r4(t.A * 255)}
		row["material"] = w.materialJSON(e)
	}
	if body := w.bodyJSON(id, e); body != nil {
		row["body"] = body
	}
	return row
}

var pbrSlots = [...]struct{ name, setter string }{
	{"albedo", "SetAlbedoMap"},
	{"normal", "SetNormalMap"},
	{"metalRough", "SetMetallicRoughnessMap"},
	{"emissive", "SetEmissiveMap"},
	{"ao", "SetAOMap"},
}

func (m *pbrMat) noteTexPath(setter, path string) {
	for _, s := range pbrSlots {
		if s.setter != setter {
			continue
		}
		if m.texPaths == nil {
			m.texPaths = map[string]string{}
		}
		if path == "" {
			delete(m.texPaths, s.name)
		} else {
			m.texPaths[s.name] = path
		}
	}
}

func (w *World) texPathOf(id int) string {
	if t := w.texs[id]; t != nil {
		return t.path
	}
	return ""
}

func shadingName(mode int) string {
	switch mode {
	case shadeUnlit:
		return "unlit"
	case shadeToon:
		return "toon"
	}
	return "pbr"
}

func (w *World) materialJSON(e *Entity) map[string]any {
	pbrOn := e.usePBR && e.pbrWrap != nil
	out := map[string]any{
		"pbr":            pbrOn,
		"castShadows":    !e.meshNoCast,
		"receiveShadows": !e.meshNoRecv,
	}
	tex := map[string]any{}
	if p := e.pbrWrap; pbrOn {
		out["metallic"] = r4(p.metallic)
		out["roughness"] = r4(p.roughness)
		out["ao"] = r4(p.ao)
		out["emissive"] = col255(p.emissive)
		out["shading"] = shadingName(p.shading)
		if p.shading == shadeToon {
			out["toon"] = map[string]any{"bands": r4(p.toonBands), "rim": r4(p.toonRim), "spec": r4(p.toonSpec)}
		}
		out["ibl"] = p.ibl
		fx := 0
		if p.shading == shadeUnlit {
			fx |= fxFullbright
		}
		if p.noFog {
			fx |= fxNoFog
		}
		if p.GetMaterial().Side() == material.SideDouble {
			fx |= fxTwoSided
		}
		out["fx"] = fx
		for k, v := range p.texPaths {
			tex[k] = v
		}
	}
	if e.albedoPath != "" && tex["albedo"] == nil {
		tex["albedo"] = e.albedoPath
	}
	if len(tex) > 0 {
		out["textures"] = tex
	}
	return out
}

func lightTypeName(k int) string {
	switch k {
	case 0:
		return "ambient"
	case 2:
		return "point"
	case 3:
		return "spot"
	}
	return "directional"
}

func lightTypeOf(v any) int {
	switch strings.ToLower(strings.TrimSpace(strOf(v))) {
	case "ambient", "0":
		return 0
	case "point", "omni", "2":
		return 2
	case "spot", "3":
		return 3
	}
	return 1
}

func (w *World) lightJSON(e *Entity) map[string]any {
	col, inten := lightRadiance(e.node)
	out := map[string]any{
		"type":      lightTypeName(e.lgtKind),
		"color":     col255(col),
		"intensity": r4(inten),
		"shadows":   e.castShadow,
	}
	if e.lgtKind == 2 || e.lgtKind == 3 {
		out["range"] = r4(entityLightRange(e))
	}
	if s, ok := e.node.(*light.Spot); ok {
		out["cone"] = []any{r4(s.AngularDecay()), r4(s.CutoffAngle())}
	}
	return out
}

func areaJSON(al *areaLight) map[string]any {
	if al == nil {
		return nil
	}
	return map[string]any{
		"size":      []any{r4(al.width), r4(al.height)},
		"color":     col255(al.color),
		"intensity": r4(al.intensity),
		"range":     r4(al.rng),
		"twoSided":  al.twoSided,
	}
}

func (w *World) cameraJSON(e *Entity) map[string]any {
	c := e.cam
	out := map[string]any{
		"fov":    r4(c.Fov()),
		"near":   r4(c.Near()),
		"far":    r4(c.Far()),
		"active": w.cam == c,
	}
	if w.cam == c {
		out["clear"] = col255(w.clear)
	}
	return out
}

var bodyShapeNames = map[int]string{1: "box", 2: "sphere", 3: "capsule", 4: "cylinder", 5: "box", 6: "compound", 7: "mesh", 8: "convex"}

func bodyTypeName(t int) string {
	switch t {
	case 1:
		return "dynamic"
	case 2:
		return "static"
	case 3:
		return "kinematic"
	}
	return "none"
}

func bodyTypeOf(v any) int {
	switch strings.ToLower(strings.TrimSpace(strOf(v))) {
	case "dynamic", "1", "":
		return 1
	case "static", "2":
		return 2
	case "kinematic", "3":
		return 3
	}
	return 0
}

func (w *World) bodyJSON(id int, e *Entity) map[string]any {
	if e.bodyType == 0 || e.collKind < 1 || e.collKind > 8 {
		return nil
	}
	m := e.meta
	shape := bodyShapeNames[e.collKind]
	if m != nil && m.autoBody && (e.collKind == 1 || e.collKind == 2) {
		shape = "auto"
	}
	out := map[string]any{"type": bodyTypeName(e.bodyType), "shape": shape}
	switch shape {
	case "box", "sphere", "capsule", "cylinder":
		out["size"] = vecJSON(e.boxX, e.boxY, e.boxZ)
		out["radius"] = r4(e.radius)
	}
	if w.phys3 != nil {
		if e.bodyType == 1 {
			if mass := w.phys3.GetMass(id); mass > 0 {
				out["mass"] = r4(mass)
			}
		}
		out["friction"] = r4(w.phys3.GetFriction(id))
		out["restitution"] = r4(w.phys3.GetRestitution(id))
	}
	layer, sensor := 0, e.collKind == 5
	if m != nil {
		layer = m.layer
		sensor = sensor || m.sensor
	}
	out["layer"] = layer
	out["sensor"] = sensor
	if e.physMat != "" {
		out["material"] = e.physMat
	}
	return out
}

func (w *World) noteBodyLayer(id, layer int) {
	if e := w.ents[id]; e != nil {
		w.metaOf(e).layer = layer
	}
}

func (w *World) noteBodySensor(id int, on bool) {
	if e := w.ents[id]; e != nil {
		w.metaOf(e).sensor = on
	}
}

func fogModeName(mode int) string {
	switch mode {
	case 1:
		return "linear"
	case 2:
		return "exp"
	case 3:
		return "exp2"
	}
	return "off"
}

func fogModeOf(v any) int {
	switch strings.ToLower(strings.TrimSpace(strOf(v))) {
	case "linear", "1", "on":
		return 1
	case "exp", "exponential", "2":
		return 2
	case "exp2", "3":
		return 3
	}
	return 0
}

func tonemapName(n int) string {
	switch n {
	case 1:
		return "neutral"
	case 2:
		return "aces"
	case 3:
		return "none"
	}
	return "reinhard"
}

// environmentJSON is the scene's ambient, clear, fog, sky, IBL, sun,
// quality, and post settings.
func (w *World) environmentJSON() map[string]any {
	env := map[string]any{"clear": col255(w.clear)}
	if w.ambient != nil {
		env["ambient"] = col255(w.ambient.Color())
		env["ambientIntensity"] = r4(w.ambient.Intensity())
	}
	env["fog"] = map[string]any{
		"mode":    fogModeName(w.fogMode),
		"color":   col255(w.fogRGB),
		"near":    r4(w.fogNear),
		"far":     r4(w.fogFar),
		"density": r4(w.fogDensity),
	}
	if w.scn.sky != "" {
		env["sky"] = w.scn.sky
	}
	if w.scn.skybox != "" {
		env["skybox"] = w.scn.skybox
	}
	env["ibl"] = map[string]any{"on": w.iblOn, "intensity": r4(w.iblIntensity)}
	if w.atmo != nil {
		env["sun"] = vecJSON(w.atmo.sunX, w.atmo.sunY, w.atmo.sunZ)
	}
	if w.quality.set {
		if w.quality.auto {
			env["quality"] = "auto"
		} else {
			env["quality"] = qualityNames[w.quality.tier]
		}
	}
	exp := w.post.exposure
	if exp <= 0 {
		exp = 1
	}
	post := map[string]any{
		"on":       w.post.on,
		"tonemap":  tonemapName(w.post.tonemap),
		"exposure": r4(exp),
		"bloom":    r4(w.post.bloom),
	}
	if w.post.lut.name != "" {
		post["lut"] = w.post.lut.name
		post["lutAmount"] = r4(w.post.lut.amount)
	}
	env["post"] = post
	return env
}

// ---- prefab instances ----

// instChainHas is true when root uid is one of e's instance roots.
func (w *World) instChainHas(e *Entity, root string) bool {
	if e == nil || e.meta == nil {
		return false
	}
	inst := e.meta.inst
	for depth := 0; inst != "" && depth < 64; depth++ {
		if inst == root {
			return true
		}
		re := w.ents[w.entityByUID(inst)]
		if re == nil || re.meta == nil {
			return false
		}
		inst = re.meta.inst
	}
	return false
}

// instanceMembers are the entities a prefab instance built, nested ones too.
func (w *World) instanceMembers(rootID int) []int {
	ruid := w.entityUID(rootID)
	var out []int
	for id, e := range w.ents {
		if id != rootID && e != nil && e.meta != nil && e.meta.inst != "" && w.instChainHas(e, ruid) {
			out = append(out, id)
		}
	}
	sort.Ints(out)
	return out
}

func (w *World) memberKey(id int, rootUID string) string {
	return strings.TrimPrefix(w.entityUID(id), rootUID+"/")
}

func (w *World) instanceSnapshot(rootID int) map[string]map[string]any {
	ruid := w.entityUID(rootID)
	base := map[string]map[string]any{}
	if raw := w.rawEntityJSON(rootID); raw != nil {
		base["."] = raw
	}
	for _, id := range w.instanceMembers(rootID) {
		if raw := w.rawEntityJSON(id); raw != nil {
			base[w.memberKey(id, ruid)] = raw
		}
	}
	return base
}

// Root keys a prefab reference row carries itself, not in overrides.
var instanceRowKeys = []string{"name", "tags", "script", "visible", "pos", "rot", "scale"}

func diffEntityJSON(base, cur map[string]any, root bool) map[string]any {
	skip := map[string]bool{"uid": true, "kind": true, "src": true}
	if root {
		skip["parent"] = true
		for _, k := range instanceRowKeys {
			skip[k] = true
		}
	}
	out := map[string]any{}
	for k, v := range cur {
		if !skip[k] && !jsonNear(base[k], v) {
			out[k] = v
		}
	}
	for k := range base {
		if _, ok := cur[k]; !ok && !skip[k] {
			out[k] = nil
		}
	}
	return out
}

func (w *World) prefabRowJSON(id int) map[string]any {
	e := w.ents[id]
	pi := e.meta.prefab
	raw := w.rawEntityJSON(id)
	if raw == nil {
		raw = map[string]any{"uid": w.entityUID(id)}
	}
	row := map[string]any{"uid": w.entityUID(id), "kind": "prefab", "src": pi.src}
	for _, k := range append([]string{"parent"}, instanceRowKeys...) {
		if v, ok := raw[k]; ok {
			row[k] = v
		}
	}
	if pi.broken {
		if len(pi.overrides) > 0 {
			row["overrides"] = pi.overrides
		}
		return row
	}
	ruid := w.entityUID(id)
	cur := map[string]map[string]any{".": raw}
	for _, mid := range w.instanceMembers(id) {
		if r := w.rawEntityJSON(mid); r != nil {
			cur[w.memberKey(mid, ruid)] = r
		}
	}
	ov := map[string]any{}
	for k, b := range pi.base {
		c, ok := cur[k]
		if !ok {
			ov[k] = map[string]any{"deleted": true}
			continue
		}
		if d := diffEntityJSON(b, c, k == "."); len(d) > 0 {
			ov[k] = d
		}
	}
	if len(ov) > 0 {
		row["overrides"] = ov
	}
	return row
}

// foldedInto is true when e is a member of an instance in set that is saved
// as a prefab reference, so the reference's overrides carry e.
func (w *World) foldedInto(e *Entity, set map[int]bool) bool {
	if e == nil || e.meta == nil {
		return false
	}
	inst := e.meta.inst
	for depth := 0; inst != "" && depth < 64; depth++ {
		rid := w.entityByUID(inst)
		re := w.ents[rid]
		if re == nil || re.meta == nil {
			return false
		}
		if set[rid] && re.meta.prefab != nil {
			return true
		}
		inst = re.meta.inst
	}
	return false
}

// sceneOrder lists ids parents first. roots nil means every entity.
func (w *World) sceneOrder(only map[int]bool) []int {
	ids := make([]int, 0, len(w.ents))
	for id, e := range w.ents {
		if e == nil || e.node == nil || e.sky || (only != nil && !only[id]) {
			continue
		}
		if w.sceneKind(id, e) == "" && (e.meta == nil || e.meta.prefab == nil) {
			continue
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	in := map[int]bool{}
	for _, id := range ids {
		in[id] = true
	}
	done := map[int]bool{}
	out := make([]int, 0, len(ids))
	var visit func(id, depth int)
	visit = func(id, depth int) {
		if done[id] || depth > 256 {
			return
		}
		if p := w.ents[id].parent; in[p] && !done[p] {
			visit(p, depth+1)
		}
		if !done[id] {
			done[id] = true
			out = append(out, id)
		}
	}
	for _, id := range ids {
		visit(id, 0)
	}
	return out
}

func (w *World) sceneRowsFor(ids []int) []any {
	set := map[int]bool{}
	for _, id := range ids {
		set[id] = true
	}
	rows := []any{}
	for _, id := range ids {
		if w.foldedInto(w.ents[id], set) {
			continue
		}
		if row := w.entityJSON(id); row != nil {
			rows = append(rows, row)
		}
	}
	return rows
}

// sceneV2 is the whole scene as a format v2 document.
func (w *World) sceneV2() map[string]any {
	return map[string]any{
		"version":     sceneFormatVersion,
		"environment": w.environmentJSON(),
		"entities":    w.sceneRowsFor(w.sceneOrder(nil)),
	}
}

func (w *World) subtreeOrder(rootID int) []int {
	only := map[int]bool{rootID: true}
	for changed := true; changed; {
		changed = false
		for id, e := range w.ents {
			if e != nil && !only[id] && only[e.parent] && e.parent != 0 {
				only[id] = true
				changed = true
			}
		}
	}
	return w.sceneOrder(only) // the root has no parent in the set, so it is first
}

// prefabJSON is the entity and its children as a prefab file: the root at
// the origin, uids local to the prefab.
func (w *World) prefabJSON(rootID int) map[string]any {
	ids := w.subtreeOrder(rootID)
	rows := w.sceneRowsFor(ids)
	ruid := w.entityUID(rootID)
	local := func(u string) string {
		if strings.HasPrefix(u, ruid+"/") {
			u = u[len(ruid)+1:]
		}
		return strings.ReplaceAll(u, "/", "_")
	}
	for i, raw := range rows {
		r := raw.(map[string]any)
		r["uid"] = local(strOf(r["uid"]))
		if i == 0 {
			delete(r, "parent")
			r["pos"] = []any{0.0, 0.0, 0.0}
		} else if p := strOf(r["parent"]); p != "" {
			r["parent"] = local(p)
		}
	}
	return map[string]any{"version": sceneFormatVersion, "prefab": true, "entities": rows}
}

// ---- writing entities ----

// sceneBuild is one load or instantiate: the prefab stack for the cycle
// guard, the folder relative prefab paths start from, and what was made.
type sceneBuild struct {
	w       *World
	dir     string
	stack   []string
	created []int
	errs    []error
}

func (b *sceneBuild) warn(err error) {
	if err != nil {
		b.errs = append(b.errs, err)
	}
}

// finish puts every body where its entity ended up.
func (b *sceneBuild) finish() error {
	b.w.syncBodies(b.created...)
	return errors.Join(b.errs...)
}

func normKind(v any) string {
	k := strings.ToLower(strings.TrimSpace(strOf(v)))
	switch k {
	case "", "box":
		return "cube"
	case "skin":
		return "mesh"
	case "area", "arealight", "area_light":
		return "arealight"
	}
	return k
}

func (w *World) placeholder(kind, src string) int {
	return w.addEntity(&Entity{node: core.NewNode(), kind: kind, src: src}, 0)
}

// spawnRowKind makes a fresh entity of the row's kind at the origin.
func (w *World) spawnRowKind(kind string, r map[string]any) (int, error) {
	switch {
	case scenePrimitiveKinds[kind]:
		return w.spawnSceneKind(kind, "", 0), nil
	case kind == "pivot":
		return w.placeholder("pivot", ""), nil
	case kind == "light":
		return w.makeLight(lightTypeOf(mapOf(r["light"])["type"]), 0), nil
	case kind == "camera":
		return w.spawnCamera(0), nil
	case kind == "arealight":
		size := numList(mapOf(r["area"])["size"])
		sw, sh := float32(1), float32(0)
		if len(size) >= 1 {
			sw = float32(size[0])
		}
		if len(size) >= 2 {
			sh = float32(size[1])
		}
		return w.createAreaLight(sw, sh, 0), nil
	case kind == "mesh":
		src := strOf(r["src"])
		if src == "" {
			return w.placeholder("pivot", ""), fmt.Errorf("mesh %q has no src", strOf(r["name"]))
		}
		id, err := w.loadMeshFile(src, 0)
		if err != nil || id == 0 {
			// Keep the reference so a save writes it back.
			return w.placeholder("mesh", src), fmt.Errorf("mesh %q: %v", src, err)
		}
		return id, nil
	}
	return w.placeholder("pivot", ""), fmt.Errorf("unknown entity kind %q", kind)
}

// setEntityParent moves an entity under pid (0 is the scene root) and keeps
// its local transform.
func (w *World) setEntityParent(id, pid int) {
	e := w.ents[id]
	if e == nil || e.node == nil || pid == id {
		return
	}
	if pid != 0 && w.ents[pid] == nil {
		pid = 0
	}
	for p, depth := pid, 0; p != 0 && depth < 256; depth++ {
		if p == id {
			return // would make a loop
		}
		pe := w.ents[p]
		if pe == nil {
			break
		}
		p = pe.parent
	}
	n := e.node.GetNode()
	if pn := w.parentNode(pid); pn != nil {
		if cur := n.Parent(); cur != nil && cur.GetNode() == pn.GetNode() && e.parent == pid {
			return
		}
		if cur := n.Parent(); cur != nil {
			cur.GetNode().Remove(e.node)
		}
		pn.GetNode().Add(e.node)
	}
	e.parent = pid
}

// applyEntityProps applies the row's fields. full means a missing field is
// cleared (tags, script, body); otherwise it is left alone. It reports
// whether the transform changed.
func (w *World) applyEntityProps(id int, m map[string]any, full bool, b *sceneBuild) bool {
	e := w.ents[id]
	if e == nil || e.node == nil || m == nil {
		return false
	}
	n := e.node.GetNode()
	meta := w.metaOf(e)
	kind := w.sceneKind(id, e)
	if v, ok := m["name"].(string); ok {
		e.name = v
		n.SetName(v)
	}
	if v, ok := m["tags"]; ok {
		meta.tags = tagList(v)
	} else if full {
		meta.tags = nil
	}
	if v, ok := m["script"]; ok {
		meta.script = strOf(v)
	} else if full {
		meta.script = ""
	}
	if v, ok := m["visible"].(bool); ok {
		n.SetVisible(v)
	}
	moved := false
	if v, ok := m["pos"]; ok {
		p := vecOf(v, [3]float32{})
		gx, gy, gz := toG3N(p[0], p[1], p[2])
		n.SetPosition(gx, gy, gz)
		moved = true
	}
	if v, ok := m["rot"]; ok {
		r := vecOf(v, [3]float32{})
		e.pitch, e.yaw, e.roll = r[0], r[1], r[2]
		w.applyRot(e)
		moved = true
	}
	if v, ok := m["scale"]; ok {
		s := vecOf(v, [3]float32{1, 1, 1})
		n.SetScale(s[0], s[1], s[2])
		moved = true
	}
	if isMeshKind(kind) {
		if mm := mapOf(m["material"]); mm != nil {
			w.applyMaterialJSON(e, mm, b)
		}
		if c, ok := colorOf(m["color"]); ok {
			t := e.tint
			if !jsonNear([]float32{t.R, t.G, t.B, t.A}, []float32{c.R, c.G, c.B, c.A}) {
				w.setEntityRGB(e, c.R*255, c.G*255, c.B*255)
				e.tint.A = c.A
				if e.mat != nil {
					e.mat.SetOpacity(c.A)
					e.mat.SetTransparent(c.A < 1)
				}
				if p := e.pbrWrap; p != nil && e.usePBR {
					p.albedo = e.tint
					p.applyFactors()
				}
			}
		}
	}
	if lm := mapOf(m["light"]); lm != nil && e.lgt != nil {
		w.applyLightJSON(id, e, lm)
	}
	if am := mapOf(m["area"]); am != nil {
		if al := w.areaLights[id]; al != nil && al.e == e {
			applyAreaJSON(al, am)
		}
	}
	if cm := mapOf(m["camera"]); cm != nil && e.cam != nil {
		w.applyCameraJSON(e, cm)
	}
	if v, ok := m["body"]; ok {
		w.applyBodyJSON(id, e, mapOf(v), b)
	} else if full && e.bodyType != 0 && e.collKind != 0 {
		w.applyBodyJSON(id, e, nil, b)
	}
	return moved
}

func (w *World) applyMaterialJSON(e *Entity, want map[string]any, b *sceneBuild) {
	cur := w.materialJSON(e)
	changed := func(k string) bool {
		v, ok := want[k]
		return ok && v != nil && !jsonNear(v, cur[k])
	}
	if v, ok := want["castShadows"]; ok {
		e.meshNoCast = !boolOf(v, true)
	}
	if v, ok := want["receiveShadows"]; ok {
		on := boolOf(v, true)
		if e.meshNoRecv == on {
			e.meshNoRecv = !on
			w.setMeshReceiveShadow(e, on)
		}
	}
	if changed("pbr") {
		if boolOf(want["pbr"], false) {
			w.convertEntityPBR(e)
		} else {
			w.convertEntityPhong(e)
		}
		cur = w.materialJSON(e)
	}
	pbrOn := e.usePBR && e.pbrWrap != nil
	if pbrOn {
		mode := -1
		if changed("shading") {
			if m, ok := shadingMode(strOf(want["shading"]), -1); ok {
				mode = m
			}
		}
		fx := -1
		if changed("fx") {
			fx = int(numOf(want["fx"], 0))
			if _, ok := want["shading"]; !ok && fx&fxFullbright != 0 {
				mode = shadeUnlit
			}
		}
		toon := mapOf(want["toon"])
		w.eachEntityPBR(e, func(p *pbrMat) {
			if changed("metallic") {
				p.metallic = clamp01n(float32(numOf(want["metallic"], 0)))
			}
			if changed("roughness") {
				p.roughness = clamp01n(float32(numOf(want["roughness"], 0.5)))
			}
			if changed("ao") {
				p.ao = float32(numOf(want["ao"], 1))
			}
			if changed("emissive") {
				if c, ok := colorOf(want["emissive"]); ok {
					p.emissive = math32.Color{R: c.R, G: c.G, B: c.B}
				}
			}
			if changed("ibl") {
				p.ibl = int(numOf(want["ibl"], -1))
			}
			if mode >= 0 {
				p.shading = mode
				if mode == shadeToon && p.toonBands <= 0 {
					p.setToon(3, 0.5, 0.6)
				}
			}
			if toon != nil && (mode == shadeToon || (mode < 0 && p.shading == shadeToon)) && changed("toon") {
				p.setToon(float32(numOf(toon["bands"], 3)), float32(numOf(toon["rim"], 0.5)), float32(numOf(toon["spec"], 0.6)))
			}
			if fx >= 0 {
				p.noFog = fx&fxNoFog != 0
				side := material.SideFront
				if fx&fxTwoSided != 0 {
					side = material.SideDouble
				}
				p.GetMaterial().SetSide(side)
				if fx&fxFullbright == 0 && p.shading == shadeUnlit && mode < 0 {
					p.shading = shadePBR
				}
			}
			p.applyFactors()
		})
		// applyFactors writes albedo. A primitive's colour is its tint (EntityColor
		// sets the factor alone); a loaded mesh keeps its own materials.
		if p := e.pbrWrap; p != nil && scenePrimitiveKinds[w.sceneKind(0, e)] {
			p.albedo = e.tint
			if p.albedo.A <= 0 {
				p.albedo.A = 1
			}
			p.applyFactors()
		}
	}
	tw := mapOf(want["textures"])
	if tw == nil {
		return
	}
	tc := mapOf(cur["textures"])
	for _, slot := range pbrSlots {
		v, ok := tw[slot.name]
		if !ok || strOf(v) == strOf(tc[slot.name]) {
			continue
		}
		path := strOf(v)
		tid := 0
		if path != "" {
			id, err := w.loadTextureFile(path)
			if err != nil || id == 0 {
				if b != nil {
					b.warn(fmt.Errorf("texture %q: %v", path, err))
				}
				continue
			}
			tid = id
		}
		w.applyTextureSlot(e, slot.name, slot.setter, tid, path)
	}
}

func (w *World) applyTextureSlot(e *Entity, slot, setter string, tid int, path string) {
	tex := w.texByID(tid)
	if e.usePBR && e.pbrWrap != nil {
		w.eachEntityPBR(e, func(m *pbrMat) {
			switch setter {
			case "SetNormalMap":
				m.SetNormalMap(tex)
			case "SetMetallicRoughnessMap":
				m.SetMetallicRoughnessMap(tex)
			case "SetEmissiveMap":
				m.SetEmissiveMap(tex)
			case "SetAOMap":
				m.SetOcclusionMap(tex)
			default:
				m.SetBaseColorMap(tex)
			}
			m.noteTexPath(setter, path)
		})
		if slot == "albedo" {
			e.albedoPath = path
			e.albedoTex = tex
		}
		return
	}
	if slot != "albedo" {
		return
	}
	old := e.albedoTex
	e.albedoPath = path
	e.albedoTex = tex
	if e.mat != nil {
		if old != nil && old != tex && e.mat.HasTexture(old) {
			e.mat.RemoveTexture(old)
		}
		if tex != nil && !e.mat.HasTexture(tex) {
			e.mat.AddTexture(tex)
		}
	}
}

func (w *World) applyLightJSON(id int, e *Entity, lm map[string]any) {
	cur := w.lightJSON(e)
	changed := func(k string) bool {
		v, ok := lm[k]
		return ok && v != nil && !jsonNear(v, cur[k])
	}
	if changed("color") {
		if c, ok := colorOf(lm["color"]); ok {
			e.lgt.SetColor(&math32.Color{R: c.R, G: c.G, B: c.B})
			e.tint.R, e.tint.G, e.tint.B = c.R, c.G, c.B
		}
	}
	if changed("intensity") && e.lgtI != nil {
		e.lgtI.SetIntensity(float32(numOf(lm["intensity"], 1)))
	}
	if changed("range") {
		w.setLightRange(e, float32(numOf(lm["range"], 10)))
	}
	if changed("cone") {
		if s, ok := e.node.(*light.Spot); ok {
			c := numList(lm["cone"])
			if len(c) >= 2 {
				s.SetAngularDecay(float32(c[0]))
				s.SetCutoffAngle(float32(c[1]))
			}
		}
	}
	if v, ok := lm["shadows"]; ok {
		e.castShadow = boolOf(v, false)
		if e.castShadow && (e.lgtKind == 1 || e.lgtKind == 0) {
			w.shadow.lightID = id
		} else if !e.castShadow && w.shadow.lightID == id {
			w.shadow.lightID = 0
		}
	}
}

func applyAreaJSON(al *areaLight, am map[string]any) {
	if v, ok := am["size"]; ok {
		s := numList(v)
		if len(s) >= 1 && s[0] > 0 {
			al.width = float32(s[0])
			al.height = al.width
		}
		if len(s) >= 2 && s[1] > 0 {
			al.height = float32(s[1])
		}
	}
	if c, ok := colorOf(am["color"]); ok {
		al.color = math32.Color{R: c.R, G: c.G, B: c.B}
	}
	if v, ok := am["intensity"]; ok {
		al.intensity = float32(math.Max(0, numOf(v, 4)))
	}
	if v, ok := am["range"]; ok {
		al.rng = float32(math.Max(0, numOf(v, 0)))
	}
	if v, ok := am["twoSided"]; ok {
		al.twoSided = boolOf(v, false)
	}
	al.syncPanel()
}

func (w *World) applyCameraJSON(e *Entity, cm map[string]any) {
	c := e.cam
	if fov := numOf(cm["fov"], 0); fov > 1 {
		c.SetFov(float32(fov))
	}
	if near := numOf(cm["near"], 0); near > 0 {
		c.SetNear(float32(near))
	}
	if far := numOf(cm["far"], 0); far > 0 {
		c.SetFar(float32(far))
	}
	if boolOf(cm["active"], false) {
		w.cam = c
		if col, ok := colorOf(cm["clear"]); ok {
			w.clear = math32.Color{R: col.R, G: col.G, B: col.B}
		}
	}
}

// ---- bodies ----

func (w *World) removeBody(id int, e *Entity) {
	if w.phys3 != nil && e.bodyType != 0 {
		w.phys3.Remove(id)
	}
	e.bodyType = 0
	e.collKind = 0
	if e.meta != nil {
		e.meta.autoBody = false
		e.meta.bodyAtOK = false
	}
}

func (w *World) applyBodyJSON(id int, e *Entity, spec map[string]any, b *sceneBuild) {
	cur := w.bodyJSON(id, e)
	if spec == nil || bodyTypeOf(spec["type"]) == 0 {
		if cur != nil {
			w.removeBody(id, e)
		}
		return
	}
	if cur != nil && jsonNear(spec, cur) {
		return
	}
	if err := w.createBody(id, e, spec); err != nil && b != nil {
		b.warn(err)
	}
}

// createBody builds the collider a v2 body block describes, where the
// entity is now in the world.
func (w *World) createBody(id int, e *Entity, spec map[string]any) error {
	w.ensurePhys3()
	w.removeBody(id, e)
	typ := bodyTypeOf(spec["type"])
	if typ == 0 {
		return nil
	}
	motion := sceneBodyMotion(typ)
	shape := strings.ToLower(strings.TrimSpace(strOf(spec["shape"])))
	if shape == "" {
		shape = "auto"
	}
	mass := float32(numOf(spec["mass"], 0))
	size := vecOf(spec["size"], [3]float32{1, 1, 1})
	radius := float32(numOf(spec["radius"], 0))
	sensor := boolOf(spec["sensor"], false)
	meta := w.metaOf(e)
	x, y, z := w.blitzPosOf(id)
	rotate := true
	switch shape {
	case "sphere":
		r := radius
		if r <= 0 {
			r = maxf3(size)
		}
		w.phys3.AddSphereEx(id, x, y, z, r, motion)
		e.collKind = 2
		e.radius = r
		e.boxX, e.boxY, e.boxZ = size[0], size[1], size[2]
	case "capsule":
		r := radius
		if r <= 0 {
			r = 0.4
		}
		w.phys3.AddCapsule(id, x, y, z, size[1], r, motion != phys3d.MotionTypeStatic)
		e.collKind = 3
		e.boxX, e.boxY, e.boxZ = size[0], size[1], size[2]
		e.radius = r
	case "cylinder":
		r := radius
		if r <= 0 {
			r = size[0]
		}
		w.phys3.AddCylinder(id, x, y, z, size[1], r, motion)
		e.collKind = 4
		e.boxX, e.boxY, e.boxZ = r, size[1], size[2]
		e.radius = r
	case "mesh":
		if !w.addEntityMeshBody(id, e) {
			return fmt.Errorf("body: entity %q has no triangles for a mesh collider", e.name)
		}
		typ = 2
		rotate = false
	case "convex":
		if !w.addEntityConvexBody(id, e, motion) {
			return fmt.Errorf("body: entity %q has no points for a convex hull", e.name)
		}
		e.collKind = 8
		rotate = false
	case "compound":
		parts, drop := w.gatherCompoundParts(id)
		if len(parts) == 0 {
			return fmt.Errorf("body: compound %q has no child bodies", e.name)
		}
		for _, cid := range drop {
			w.phys3.Remove(cid)
			if ce := w.ents[cid]; ce != nil {
				ce.bodyType = 0
				ce.collKind = 0
			}
		}
		w.phys3.AddCompound(id, parts, x, y, z, motion)
		e.collKind = 6
		hx, hy, hz := phys3d.CompoundAABB(parts)
		e.boxX, e.boxY, e.boxZ = hx, hy, hz
	case "auto":
		if err := w.collideEntity(id, motion, mass, mass > 0); err != nil {
			return err
		}
		meta.autoBody = true
		rotate = false
	default: // box
		if sensor {
			w.phys3.AddSensorBox(id, x, y, z, size[0], size[1], size[2], motion)
			e.collKind = 5
		} else {
			w.phys3.AddBoxEx(id, x, y, z, size[0], size[1], size[2], motion)
			e.collKind = 1
		}
		e.boxX, e.boxY, e.boxZ = size[0], size[1], size[2]
		if radius > 0 {
			e.radius = radius
		}
	}
	e.bodyType = typ
	if rotate && e.node != nil {
		wq := worldQuat(e.node.GetNode())
		w.phys3.SetRotation(id, wq.X, wq.Y, -wq.Z, wq.W)
	}
	if motion == phys3d.MotionTypeDynamic && mass > 0 && shape != "auto" {
		w.phys3.SetMass(id, mass)
	}
	if pm := strOf(spec["material"]); pm != "" {
		e.physMat = pm
		_ = w.applyPhysicsMaterial(id, pm)
	}
	if v, ok := spec["friction"]; ok {
		w.phys3.SetFriction(id, float32(numOf(v, 0.2)))
	}
	if v, ok := spec["restitution"]; ok {
		w.phys3.SetRestitution(id, float32(numOf(v, 0)))
	}
	meta.layer = int(numOf(spec["layer"], 0))
	if meta.layer != 0 {
		w.phys3.SetCollisionLayer(id, meta.layer)
	}
	meta.sensor = sensor
	if sensor && e.collKind != 5 {
		w.phys3.SetSensor(id, true)
	}
	meta.bodyAt, meta.bodyAtOK = bodyXform(e), true
	return nil
}

func maxf3(v [3]float32) float32 {
	m := v[0]
	if v[1] > m {
		m = v[1]
	}
	if v[2] > m {
		m = v[2]
	}
	return m
}

func bodyXform(e *Entity) [7]float32 {
	n := e.node.GetNode()
	p := worldPos(n)
	q := worldQuat(n)
	return [7]float32{p.X, p.Y, p.Z, q.X, q.Y, q.Z, q.W}
}

// syncBodies moves the bodies of these entities and their children to where
// the entities now are. Mesh and convex colliders are baked in world space,
// so they are rebuilt.
func (w *World) syncBodies(roots ...int) {
	if w.phys3 == nil || len(roots) == 0 {
		return
	}
	w.refreshWorldMatrices()
	kids := map[int][]int{}
	for id, e := range w.ents {
		if e != nil && e.parent != 0 {
			kids[e.parent] = append(kids[e.parent], id)
		}
	}
	seen := map[int]bool{}
	var walk func(id int)
	walk = func(id int) {
		if seen[id] {
			return
		}
		seen[id] = true
		if e := w.ents[id]; e != nil && e.node != nil && e.bodyType != 0 && e.collKind != 0 {
			w.syncBody(id, e)
		}
		for _, c := range kids[id] {
			walk(c)
		}
	}
	for _, id := range roots {
		walk(id)
	}
}

func (w *World) syncBody(id int, e *Entity) {
	at := bodyXform(e)
	meta := w.metaOf(e)
	if meta.bodyAtOK {
		same := true
		for i := range at {
			if math.Abs(float64(at[i]-meta.bodyAt[i])) > 1e-5 {
				same = false
				break
			}
		}
		if same {
			return
		}
	}
	switch e.collKind {
	case 7:
		fr, re := w.phys3.GetFriction(id), w.phys3.GetRestitution(id)
		w.addEntityMeshBody(id, e)
		w.phys3.SetFriction(id, fr)
		w.phys3.SetRestitution(id, re)
	case 8:
		mass := w.phys3.GetMass(id)
		fr, re := w.phys3.GetFriction(id), w.phys3.GetRestitution(id)
		motion := sceneBodyMotion(e.bodyType)
		w.phys3.Remove(id)
		w.addEntityConvexBody(id, e, motion)
		if motion == phys3d.MotionTypeDynamic && mass > 0 {
			w.phys3.SetMass(id, mass)
		}
		w.phys3.SetFriction(id, fr)
		w.phys3.SetRestitution(id, re)
	default:
		x, y, z := fromG3N(at[0], at[1], at[2])
		w.phys3.SetPosition(id, x, y, z)
		w.phys3.SetRotation(id, at[3], at[4], -at[5], at[6])
	}
	if meta.layer != 0 {
		w.phys3.SetCollisionLayer(id, meta.layer)
	}
	meta.bodyAt, meta.bodyAtOK = at, true
}

// ---- building scenes ----

func (w *World) applyInstanceFields(id int, r map[string]any) bool {
	sub := map[string]any{}
	for _, k := range instanceRowKeys {
		if v, ok := r[k]; ok {
			sub[k] = v
		}
	}
	return w.applyEntityProps(id, sub, false, nil)
}

// buildRows makes the rows' entities. With a prefix it is a prefab: the
// single root row takes the prefix as its uid (several roots get a pivot
// that does), the rest "<prefix>/<uid>", and all are members of the root.
func (b *sceneBuild) buildRows(rows []map[string]any, prefix string) (int, []int) {
	w := b.w
	prefab := prefix != ""
	fileUID := make([]string, len(rows))
	inFile := map[string]int{}
	for i, r := range rows {
		u := strOf(r["uid"])
		if _, dup := inFile[u]; u == "" || dup {
			u = "_" + strconv.Itoa(i)
		}
		fileUID[i] = u
		inFile[u] = i
	}
	isRoot := make([]bool, len(rows))
	var roots []int
	for i, r := range rows {
		j, ok := inFile[strOf(r["parent"])]
		if !ok || j == i {
			isRoot[i] = true
			roots = append(roots, i)
		}
	}
	wrap := prefab && len(roots) != 1
	sceneUID := make([]string, len(rows))
	used := map[string]bool{}
	for i := range rows {
		switch {
		case !prefab:
			u := strOf(rows[i]["uid"])
			if u == "" || used[u] || w.entityByUID(u) != 0 {
				u = w.freshUID()
			}
			sceneUID[i] = u
		case !wrap && isRoot[i]:
			sceneUID[i] = prefix
		default:
			sceneUID[i] = prefix + "/" + fileUID[i]
		}
		used[sceneUID[i]] = true
	}
	var ids []int
	rootID := 0
	if wrap {
		rootID = w.placeholder("pivot", "")
		w.setEntityUID(rootID, prefix)
		ids = append(ids, rootID)
	}
	rowIDs := make([]int, len(rows))
	for i, r := range rows {
		kind := normKind(r["kind"])
		var id int
		var err error
		if kind == "prefab" {
			id, err = b.instantiate(strOf(r["src"]), sceneUID[i], mapOf(r["overrides"]))
			if err != nil {
				b.warn(err)
				id = w.placeholder("pivot", "")
				w.setEntityUID(id, sceneUID[i])
				w.metaOf(w.ents[id]).prefab = &prefabInst{src: strOf(r["src"]), broken: true, overrides: mapOf(r["overrides"])}
			}
		} else {
			id, err = w.spawnRowKind(kind, r)
			b.warn(err)
			if id != 0 {
				w.setEntityUID(id, sceneUID[i])
			}
		}
		if id == 0 {
			continue
		}
		rowIDs[i] = id
		ids = append(ids, id)
		b.created = append(b.created, id)
		if prefab {
			if !wrap && isRoot[i] {
				rootID = id
			} else {
				w.metaOf(w.ents[id]).inst = prefix
			}
		}
	}
	// A parent outside the file: an entity already in the world, or a
	// member of a prefab row whose uid was changed to stay unique.
	outside := func(p string) int {
		if prefab || p == "" {
			return 0
		}
		for i, r := range rows {
			if normKind(r["kind"]) == "prefab" && sceneUID[i] != fileUID[i] && strings.HasPrefix(p, fileUID[i]+"/") {
				p = sceneUID[i] + p[len(fileUID[i]):]
				break
			}
		}
		return w.entityByUID(p)
	}
	for i, r := range rows {
		id := rowIDs[i]
		if id == 0 {
			continue
		}
		p := strOf(r["parent"])
		pid := 0
		if j, ok := inFile[p]; ok && j != i {
			pid = rowIDs[j]
		} else {
			pid = outside(p)
		}
		if wrap && isRoot[i] {
			pid = rootID
		}
		if pid != 0 {
			w.setEntityParent(id, pid)
		}
	}
	var compounds []int
	for i, r := range rows {
		id := rowIDs[i]
		if id == 0 {
			continue
		}
		if normKind(r["kind"]) == "prefab" {
			w.applyInstanceFields(id, r)
			continue
		}
		if body := mapOf(r["body"]); body != nil && strings.EqualFold(strOf(body["shape"]), "compound") {
			sub := map[string]any{}
			for k, v := range r {
				if k != "body" {
					sub[k] = v
				}
			}
			w.applyEntityProps(id, sub, false, b)
			compounds = append(compounds, i)
			continue
		}
		w.applyEntityProps(id, r, false, b)
	}
	// Compound bodies take their children's colliders, so they come last.
	if len(compounds) > 0 {
		w.refreshWorldMatrices()
	}
	for _, i := range compounds {
		if e := w.ents[rowIDs[i]]; e != nil {
			b.warn(w.createBody(rowIDs[i], e, mapOf(rows[i]["body"])))
		}
	}
	return rootID, ids
}

// instantiate builds a prefab file as an instance whose root has uid.
func (b *sceneBuild) instantiate(src, uid string, overrides map[string]any) (int, error) {
	w := b.w
	path, err := w.findPrefabFile(src, b.dir)
	if err != nil {
		return 0, err
	}
	key := path
	if abs, err := filepath.Abs(path); err == nil {
		key = abs
	}
	key = strings.ToLower(filepath.Clean(key))
	for _, s := range b.stack {
		if s == key {
			return 0, fmt.Errorf("prefab %q includes itself", src)
		}
	}
	if len(b.stack) >= maxPrefabDepth {
		return 0, fmt.Errorf("prefab %q: nested more than %d deep", src, maxPrefabDepth)
	}
	root, err := readSceneFile(path)
	if err != nil {
		return 0, fmt.Errorf("prefab %q: %w", src, err)
	}
	rows := sceneRows(root)
	if len(rows) == 0 {
		return 0, fmt.Errorf("prefab %q has no entities", src)
	}
	b.stack = append(b.stack, key)
	dir := b.dir
	b.dir = filepath.Dir(path)
	rootID, _ := b.buildRows(rows, uid)
	b.dir = dir
	b.stack = b.stack[:len(b.stack)-1]
	e := w.ents[rootID]
	if e == nil {
		return 0, fmt.Errorf("prefab %q made no root", src)
	}
	w.metaOf(e).prefab = &prefabInst{src: src, base: w.instanceSnapshot(rootID)}
	w.applyOverrides(rootID, overrides, b)
	return rootID, nil
}

// applyOverrides applies per-uid property overrides to an instance. Keys
// are uids inside the prefab ("." is the root, "a/b" is b inside nested
// prefab a); {"deleted": true} removes that member.
func (w *World) applyOverrides(rootID int, ov map[string]any, b *sceneBuild) {
	if len(ov) == 0 {
		return
	}
	ruid := w.entityUID(rootID)
	keys := make([]string, 0, len(ov))
	for k := range ov {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		di, dj := strings.Count(keys[i], "/"), strings.Count(keys[j], "/")
		if di != dj {
			return di < dj
		}
		return keys[i] < keys[j]
	})
	for _, k := range keys {
		m := mapOf(ov[k])
		if m == nil {
			continue
		}
		target := rootID
		if k != "." && k != "" {
			target = w.entityByUID(ruid + "/" + k)
		}
		if target == 0 {
			if b != nil {
				b.warn(fmt.Errorf("override %q: no such member", k))
			}
			continue
		}
		if boolOf(m["deleted"], false) {
			if target != rootID {
				w.freeSceneTree(target)
			}
			continue
		}
		if p, ok := m["parent"].(string); ok && target != rootID {
			if pid := w.entityByUID(p); pid != 0 {
				w.setEntityParent(target, pid)
			}
		}
		w.applyEntityProps(target, m, false, b)
	}
}

func (w *World) findPrefabFile(src, dir string) (string, error) {
	if strings.TrimSpace(src) == "" {
		return "", errors.New("prefab has no src")
	}
	src = filepath.FromSlash(strings.ReplaceAll(src, "\\", "/"))
	names := []string{src}
	if filepath.Ext(src) == "" {
		names = append(names, src+".prefab.json", src+".json")
	}
	roots := []string{""}
	if !filepath.IsAbs(src) {
		roots = []string{dir, w.base, filepath.Join(w.base, "prefabs")}
		if dir != "" {
			roots = append(roots, filepath.Join(dir, "prefabs"))
		}
		if w.scn.dir != "" {
			roots = append(roots, w.scn.dir)
		}
	}
	for _, r := range roots {
		for _, n := range names {
			p := n
			if r != "" {
				p = filepath.Join(r, n)
			}
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p, nil
			}
		}
	}
	if p := w.findSceneFile(src); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("prefab %q not found", src)
}

func readSceneFile(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	m, ok := root.(map[string]any)
	if !ok {
		return nil, errors.New("not a scene object")
	}
	return m, nil
}

// sceneVersionOf is 2 for v2 documents and 1 for SceneSave dumps.
func sceneVersionOf(root map[string]any) int {
	if v := numOf(root["version"], 0); v >= 2 {
		return int(v)
	}
	list, _ := root["entities"].([]any)
	for _, raw := range list {
		if em := mapOf(raw); em != nil {
			if _, ok := em["pos"]; ok {
				return 2
			}
			if _, ok := em["uid"]; ok {
				return 2
			}
		}
	}
	return 1
}

// sceneRows are a file's entities as v2 rows. SceneSave dumps and the old
// one-object SavePrefab files are converted.
func sceneRows(root map[string]any) []map[string]any {
	list, _ := root["entities"].([]any)
	if list == nil {
		if _, ok := root["kind"]; ok {
			row := v1RowToV2(root)
			row["uid"] = "root"
			return []map[string]any{row}
		}
		return nil
	}
	v1 := sceneVersionOf(root) < 2
	out := make([]map[string]any, 0, len(list))
	for _, raw := range list {
		em := mapOf(raw)
		if em == nil {
			continue
		}
		if v1 {
			em = v1RowToV2(em)
		}
		out = append(out, em)
	}
	return out
}

// v1RowToV2 converts one SceneSave row (flat x/y/z, numeric ids).
func v1RowToV2(em map[string]any) map[string]any {
	out := map[string]any{}
	if id := numOf(em["id"], 0); id != 0 {
		out["uid"] = strconv.Itoa(int(id))
	}
	if p := numOf(em["parent"], 0); p != 0 {
		out["parent"] = strconv.Itoa(int(p))
	}
	kind := normKind(em["kind"])
	out["kind"] = kind
	if name, ok := em["name"].(string); ok {
		out["name"] = name
	}
	if src := strOf(em["src"]); src != "" {
		out["src"] = src
	}
	f := func(k string, def float64) float64 { return numOf(em[k], def) }
	out["pos"] = []any{f("x", 0), f("y", 0), f("z", 0)}
	out["rot"] = []any{f("pitch", 0), f("yaw", 0), f("roll", 0)}
	sc := []any{f("sx", 1), f("sy", 1), f("sz", 1)}
	for i := range sc {
		if sc[i].(float64) == 0 {
			sc[i] = 1.0
		}
	}
	out["scale"] = sc
	if v, ok := em["visible"].(bool); ok {
		out["visible"] = v
	}
	_, hasR := em["r"]
	switch kind {
	case "light":
		lm := map[string]any{"type": lightTypeName(int(f("light", 1)))}
		if hasR {
			lm["color"] = []any{f("r", 255), f("g", 255), f("b", 255)}
		}
		out["light"] = lm
	case "camera":
		out["camera"] = map[string]any{"fov": f("fov", 0), "near": f("near", 0), "far": f("far", 0)}
	default:
		if hasR {
			out["color"] = []any{f("r", 255), f("g", 255), f("b", 255), 255.0}
		}
	}
	if mat := mapOf(em["mat"]); mat != nil {
		mm := map[string]any{}
		if _, ok := mat["metal"]; ok {
			mm["pbr"] = true
			mm["metallic"] = numOf(mat["metal"], 0)
			mm["roughness"] = numOf(mat["rough"], 0.5)
			mm["ao"] = numOf(mat["ao"], 1)
			mm["emissive"] = []any{numOf(mat["er"], 0) * 255, numOf(mat["eg"], 0) * 255, numOf(mat["eb"], 0) * 255}
		}
		if t := strOf(mat["tex"]); t != "" {
			mm["textures"] = map[string]any{"albedo": t}
		}
		out["material"] = mm
	}
	if body := int(f("body", 0)); body != 0 {
		if shape := int(f("shape", 0)); shape >= 1 && shape <= 8 {
			bm := map[string]any{
				"type":   bodyTypeName(body),
				"shape":  bodyShapeNames[shape],
				"size":   []any{f("hx", 1), f("hy", 1), f("hz", 1)},
				"radius": f("rad", 0),
				"sensor": shape == 5,
			}
			if mass := f("mass", 0); mass > 0 {
				bm["mass"] = mass
			}
			out["body"] = bm
		}
	}
	return out
}

// ---- whole scenes ----

func (w *World) freeSceneEntity(id int) {
	e := w.ents[id]
	if e == nil {
		return
	}
	if w.phys3 != nil && e.bodyType != 0 {
		w.phys3.Remove(id)
	}
	if al := w.areaLights[id]; al != nil {
		delete(w.areaLights, id)
	}
	if w.shadow.lightID == id {
		w.shadow.lightID = 0
	}
	if e.meta != nil && e.meta.uid != "" {
		if ref, ok := w.scn.uids[e.meta.uid]; ok && ref.e == e {
			delete(w.scn.uids, e.meta.uid)
		}
	}
	w.freeEntityID(id)
}

func (w *World) freeSceneTree(id int) {
	for cid, e := range w.ents {
		if e != nil && e.parent == id && cid != id {
			w.freeSceneTree(cid)
		}
	}
	w.freeSceneEntity(id)
}

// clearSceneEntities frees what a v2 file can hold. Terrain, water, and
// other entities stay; any left under a freed parent move to the root.
func (w *World) clearSceneEntities(keepCam bool) {
	var ids []int
	for id, e := range w.ents {
		if e == nil {
			continue
		}
		if keepCam && e.cam != nil && e.cam == w.cam {
			continue
		}
		if w.sceneKind(id, e) != "" || (e.meta != nil && e.meta.prefab != nil) {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		w.freeSceneEntity(id)
	}
	for id, e := range w.ents {
		if e != nil && e.parent != 0 && w.ents[e.parent] == nil {
			w.setEntityParent(id, 0)
		}
	}
}

// loadSceneV2 builds a v2 document (v1 SceneSave dumps go through the old
// loader). Without additive the scene's entities are replaced first and the
// environment is applied. Missing assets and prefabs are returned joined;
// everything else still loads.
func (w *World) loadSceneV2(root map[string]any, additive bool) error {
	if root == nil {
		return errors.New("LoadScene: no scene")
	}
	if sceneVersionOf(root) < 2 {
		if !additive {
			w.clearSceneEntities(true)
		}
		w.applySceneJSON(root)
		return nil
	}
	rows := sceneRows(root)
	hasCam := false
	for _, r := range rows {
		if normKind(r["kind"]) == "camera" {
			hasCam = true
		}
	}
	if !additive {
		w.clearSceneEntities(!hasCam)
	}
	dir := w.scn.dir
	if dir == "" {
		dir = w.base
	}
	b := &sceneBuild{w: w, dir: dir}
	_, ids := b.buildRows(rows, "")
	if hasCam && (w.cam == nil || !w.camAlive()) {
		for _, id := range ids {
			if e := w.ents[id]; e != nil && e.cam != nil {
				w.cam = e.cam
				break
			}
		}
	}
	if env := mapOf(root["environment"]); env != nil && !additive {
		w.applyEnvironmentJSON(env)
	}
	return b.finish()
}

func (w *World) camAlive() bool {
	for _, e := range w.ents {
		if e != nil && e.cam == w.cam {
			return true
		}
	}
	return false
}

func (w *World) applyEnvironmentJSON(env map[string]any) {
	live := w.ready && !w.mode2D
	if c, ok := colorOf(env["ambient"]); ok && w.ambient != nil {
		w.ambient.SetColor(&math32.Color{R: c.R, G: c.G, B: c.B})
		w.ambient.SetIntensity(float32(numOf(env["ambientIntensity"], 1)))
		w.wx.ambSaved = false
	}
	if c, ok := colorOf(env["clear"]); ok {
		w.clear = math32.Color{R: c.R, G: c.G, B: c.B}
	}
	if fog := mapOf(env["fog"]); fog != nil {
		if v, ok := fog["mode"]; ok {
			w.fogMode = fogModeOf(v)
		}
		if c, ok := colorOf(fog["color"]); ok {
			w.fogRGB = math32.Color{R: c.R, G: c.G, B: c.B}
		}
		w.fogNear = float32(numOf(fog["near"], float64(w.fogNear)))
		w.fogFar = float32(numOf(fog["far"], float64(w.fogFar)))
		w.fogDensity = float32(numOf(fog["density"], float64(w.fogDensity)))
		w.wx.fogOwned = false
		if live {
			w.applyLitShaders()
		}
	}
	sky, skybox := strOf(env["sky"]), strOf(env["skybox"])
	switch {
	case skybox != "":
		w.scn.sky, w.scn.skybox = "", skybox
		if live {
			if id, err := w.makeSkyBox(skybox); err == nil {
				w.setSkyBox(id)
			}
		}
	case sky != "":
		w.scn.sky, w.scn.skybox = sky, ""
		if live {
			w.applySkyPreset(sky)
		}
	}
	if ibl := mapOf(env["ibl"]); ibl != nil {
		w.iblOn = boolOf(ibl["on"], w.iblOn)
		w.iblIntensity = float32(numOf(ibl["intensity"], float64(w.iblIntensity)))
	}
	if sun := numList(env["sun"]); len(sun) >= 3 && live {
		_, _ = w.Call("setsundirection", []value.Value{value.Num(sun[0]), value.Num(sun[1]), value.Num(sun[2])})
	}
	if q := strOf(env["quality"]); q != "" && live {
		_, _ = w.Call("setgraphicsquality", []value.Value{value.Str(q)})
	}
	if post := mapOf(env["post"]); post != nil {
		if v, ok := post["on"]; ok {
			w.post.on = boolOf(v, w.post.on)
			w.post.userOn = true
		}
		if v, ok := post["tonemap"]; ok {
			w.post.tonemap = tonemapMode(strOf(v), 0)
		}
		if v, ok := post["exposure"]; ok {
			w.post.exposure = float32(math.Min(8, math.Max(0.05, numOf(v, 1))))
		}
		if v, ok := post["bloom"]; ok {
			w.post.bloom = float32(math.Min(2, math.Max(0, numOf(v, 0))))
		}
		if lut := strOf(post["lut"]); lut != "" {
			_, _ = w.Call("setlut", []value.Value{value.Str(lut), value.Num(numOf(post["lutAmount"], 1))})
		}
	}
}

// ---- editor API ----

func (w *World) sameEntityShape(id int, m map[string]any) bool {
	cur := w.entityJSON(id)
	if cur == nil {
		return false
	}
	kind := normKind(m["kind"])
	if strOf(cur["kind"]) != kind {
		return false
	}
	switch kind {
	case "mesh", "prefab":
		return strOf(cur["src"]) == strOf(m["src"])
	case "light":
		if lm := mapOf(m["light"]); lm != nil {
			return lightTypeOf(lm["type"]) == lightTypeOf(mapOf(cur["light"])["type"])
		}
	}
	return true
}

// applyEntityJSON creates an entity from its v2 JSON or, with keepUID and a
// uid already in the scene, updates that entity in place (same id) to match
// it. A kind, mesh src, light type, or prefab src change rebuilds it under
// the same uid and returns the new id; its children are kept. Without
// keepUID a new entity with a fresh uid is always made.
func (w *World) applyEntityJSON(m map[string]any, keepUID bool) (int, error) {
	if m == nil {
		return 0, errors.New("applyEntityJSON: no entity")
	}
	uid := strOf(m["uid"])
	if !keepUID || uid == "" {
		uid = w.freshUID()
	}
	dir := w.scn.dir
	if dir == "" {
		dir = w.base
	}
	b := &sceneBuild{w: w, dir: dir}
	if id := w.entityByUID(uid); id != 0 {
		if w.sameEntityShape(id, m) {
			w.setParentFromRow(id, m)
			var moved bool
			if e := w.ents[id]; e.meta != nil && e.meta.prefab != nil {
				moved = w.applyInstanceFields(id, m)
				w.applyOverrides(id, mapOf(m["overrides"]), b)
			} else {
				moved = w.applyEntityProps(id, m, true, b)
			}
			if moved {
				b.created = append(b.created, id)
			}
			return id, b.finish()
		}
		// Rebuild: keep the children that are not this entity's prefab members.
		var keep []int
		for cid, ce := range w.ents {
			if ce != nil && ce.parent == id && !w.instChainHas(ce, uid) {
				keep = append(keep, cid)
			}
		}
		for _, cid := range keep {
			w.setEntityParent(cid, 0)
		}
		w.freeSceneTree(id)
		nid := w.buildEntityRow(m, uid, b)
		for _, cid := range keep {
			w.setEntityParent(cid, nid)
		}
		return nid, b.finish()
	}
	return w.buildEntityRow(m, uid, b), b.finish()
}

func (w *World) setParentFromRow(id int, m map[string]any) {
	pid := w.entityByUID(strOf(m["parent"]))
	if e := w.ents[id]; e != nil && e.parent != pid {
		w.setEntityParent(id, pid)
	}
}

func (w *World) buildEntityRow(m map[string]any, uid string, b *sceneBuild) int {
	row := map[string]any{}
	for k, v := range m {
		row[k] = v
	}
	row["uid"] = uid
	delete(row, "parent")
	_, ids := b.buildRows([]map[string]any{row}, "")
	if len(ids) == 0 {
		return 0
	}
	id := w.entityByUID(uid)
	if id == 0 {
		id = ids[0]
	}
	w.setParentFromRow(id, m)
	return id
}

// ---- commands ----

func (w *World) sceneTag(e *Entity, tag string) bool {
	if e == nil || e.meta == nil {
		return false
	}
	for _, t := range e.meta.tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func (w *World) sortedEntityIDs() []int {
	ids := make([]int, 0, len(w.ents))
	for id := range w.ents {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func (w *World) sceneFileDir(rel string) string {
	p, err := w.openPath(rel)
	if err != nil {
		p = w.findSceneFile(rel)
	}
	return filepath.Dir(p)
}

func writeJSONFile(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func warnScene(where string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: warning: %v\n", where, err)
	}
}

func (w *World) sceneV2Commands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	instantiate := func(a []value.Value) (value.Value, error) {
		dir := w.scn.dir
		if dir == "" {
			dir = w.base
		}
		b := &sceneBuild{w: w, dir: dir}
		id, err := b.instantiate(argS(a, 0), w.freshUID(), nil)
		if err != nil {
			return value.Num(0), fmt.Errorf("InstantiatePrefab: %w", err)
		}
		if len(a) >= 4 {
			gx, gy, gz := toG3N(float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			w.ents[id].node.GetNode().SetPosition(gx, gy, gz)
		}
		if len(a) >= 5 {
			w.setEntityParent(id, argI(a, 4, 0))
		}
		b.created = append(b.created, id)
		warnScene("InstantiatePrefab", b.finish())
		return value.Num(float64(id)), nil
	}
	return map[string]cmd{
		// SaveScene path$ writes scene format v2.
		"savescene": need(func(a []value.Value) (value.Value, error) {
			if err := writeJSONFile(w.resolve(argS(a, 0)), w.sceneV2()); err != nil {
				return value.Num(0), err
			}
			return value.Num(1), nil
		}),
		// SavePrefab name$, entity — the entity and its children, root at the origin.
		"saveprefab": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 1, 0)
			if _, err := w.ent(id); err != nil {
				return value.Num(0), err
			}
			name := argS(a, 0)
			path := w.resolve(name)
			if filepath.Ext(name) == "" {
				path = filepath.Join(w.base, "prefabs", name+".json")
			}
			if err := writeJSONFile(path, w.prefabJSON(id)); err != nil {
				return value.Num(0), err
			}
			return value.Num(1), nil
		}),
		"instantiateprefab": need(instantiate),
		"spawnprefab":       need(instantiate),
		"findentity": n(func(a []value.Value) (value.Value, error) {
			name := argS(a, 0)
			for _, id := range w.sortedEntityIDs() {
				if e := w.ents[id]; e != nil && strings.EqualFold(e.name, name) {
					return value.Num(float64(id)), nil
				}
			}
			return value.Num(0), nil
		}),
		// FindEntityByTag(tag$ [, n]) — the nth (from 1) entity with the tag.
		"findentitybytag": n(func(a []value.Value) (value.Value, error) {
			tag, want := argS(a, 0), argI(a, 1, 1)
			k := 0
			for _, id := range w.sortedEntityIDs() {
				if w.sceneTag(w.ents[id], tag) {
					if k++; k >= want {
						return value.Num(float64(id)), nil
					}
				}
			}
			return value.Num(0), nil
		}),
		"countentitiesbytag": n(func(a []value.Value) (value.Value, error) {
			tag, k := argS(a, 0), 0
			for _, e := range w.ents {
				if w.sceneTag(e, tag) {
					k++
				}
			}
			return value.Num(float64(k)), nil
		}),
		// EntityTag$(e [, n]) — the nth tag (from 1), or "".
		"entitytag": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Str(""), err
			}
			i := argI(a, 1, 1) - 1
			if e.meta == nil || i < 0 || i >= len(e.meta.tags) {
				return value.Str(""), nil
			}
			return value.Str(e.meta.tags[i]), nil
		}),
		"entitytags": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil || e.meta == nil {
				return value.Str(""), err
			}
			return value.Str(strings.Join(e.meta.tags, ",")), nil
		}),
		// SetEntityTag e, tags$ — replaces the tags (comma separated).
		"setentitytag": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			w.metaOf(e).tags = tagList(argS(a, 1))
			return z()
		}),
		"addentitytag": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			for _, t := range tagList(argS(a, 1)) {
				if !w.sceneTag(e, t) {
					w.metaOf(e).tags = append(w.metaOf(e).tags, t)
				}
			}
			return z()
		}),
		"removeentitytag": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.meta != nil {
				drop := tagList(argS(a, 1))
				kept := e.meta.tags[:0]
				for _, t := range e.meta.tags {
					gone := false
					for _, d := range drop {
						gone = gone || strings.EqualFold(t, d)
					}
					if !gone {
						kept = append(kept, t)
					}
				}
				e.meta.tags = kept
			}
			return z()
		}),
		"entityhastag": n(func(a []value.Value) (value.Value, error) {
			if w.sceneTag(w.ents[argI(a, 0, 0)], argS(a, 1)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"entityuid": n(func(a []value.Value) (value.Value, error) {
			if _, err := w.ent(argI(a, 0, 0)); err != nil {
				return value.Str(""), err
			}
			return value.Str(w.entityUID(argI(a, 0, 0))), nil
		}),
		"entitybyuid": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.entityByUID(argS(a, 0)))), nil
		}),
		"entityscript": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil || e.meta == nil {
				return value.Str(""), err
			}
			return value.Str(e.meta.script), nil
		}),
		"setentityscript": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			w.metaOf(e).script = argS(a, 1)
			return z()
		}),
		// EntityPrefab$(e) — the prefab file an instance root was made from.
		"entityprefab": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil || e.meta == nil || e.meta.prefab == nil {
				return value.Str(""), err
			}
			return value.Str(e.meta.prefab.src), nil
		}),
	}
}
