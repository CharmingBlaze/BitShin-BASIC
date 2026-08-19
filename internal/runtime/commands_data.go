package runtime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/math32"
	"github.com/vmihailenco/msgpack/v5"
	"gopkg.in/yaml.v3"

	"bitshinbasic/internal/value"
)

type dataDoc struct {
	root any
}

func (w *World) dataCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"jsonload": n(func(a []value.Value) (value.Value, error) {
			root, err := w.readJSONFile(argS(a, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(w.allocDoc(root))), nil
		}),
		"jsonsave": n(func(a []value.Value) (value.Value, error) {
			d, err := w.docOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			b, err := json.MarshalIndent(d.root, "", "  ")
			if err != nil {
				return value.Value{}, err
			}
			if err := os.WriteFile(w.resolve(argS(a, 1)), b, 0o644); err != nil {
				return value.Value{}, err
			}
			return z()
		}),
		"jsonparse": n(func(a []value.Value) (value.Value, error) {
			var root any
			s := strings.TrimSpace(argS(a, 0))
			if s == "" {
				root = map[string]any{}
			} else if err := json.Unmarshal([]byte(s), &root); err != nil {
				return value.Value{}, fmt.Errorf("JSONParse: %w", err)
			}
			return value.Num(float64(w.allocDoc(root))), nil
		}),
		"jsonget": n(func(a []value.Value) (value.Value, error) {
			d, err := w.docOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return docGet(d.root, argS(a, 1)), nil
		}),
		"jsonset": n(func(a []value.Value) (value.Value, error) {
			d, err := w.docOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			d.root = docSet(d.root, argS(a, 1), aValue(a, 2))
			return z()
		}),
		"json$": n(func(a []value.Value) (value.Value, error) {
			d, err := w.docOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			b, err := json.Marshal(d.root)
			if err != nil {
				return value.Value{}, err
			}
			return value.Str(string(b)), nil
		}),
		"scenesave": n(func(a []value.Value) (value.Value, error) {
			root := w.sceneJSON()
			path := w.resolve(argS(a, 0))
			b, err := json.MarshalIndent(root, "", "  ")
			if err != nil {
				return value.Value{}, err
			}
			if err := os.WriteFile(path, b, 0o644); err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(w.allocDoc(root))), nil
		}),
		"sceneload": n(func(a []value.Value) (value.Value, error) {
			root, err := w.readJSONFile(argS(a, 0))
			if err != nil {
				return value.Value{}, err
			}
			if w.ready && !w.mode2D {
				w.applySceneJSON(root)
			}
			return value.Num(float64(w.allocDoc(root))), nil
		}),
		"yamlload": n(func(a []value.Value) (value.Value, error) {
			root, err := w.readYAMLFile(argS(a, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(w.allocDoc(root))), nil
		}),
		"yamlsave": n(func(a []value.Value) (value.Value, error) {
			d, err := w.docOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			b, err := yaml.Marshal(d.root)
			if err != nil {
				return value.Value{}, err
			}
			if err := os.WriteFile(w.resolve(argS(a, 1)), b, 0o644); err != nil {
				return value.Value{}, err
			}
			return z()
		}),
		"yamlparse": n(func(a []value.Value) (value.Value, error) {
			root, err := yamlViaJSON([]byte(argS(a, 0)))
			if err != nil {
				return value.Value{}, fmt.Errorf("YAMLParse: %w", err)
			}
			return value.Num(float64(w.allocDoc(root))), nil
		}),
		"yamlget": n(func(a []value.Value) (value.Value, error) {
			d, err := w.docOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return docGet(d.root, argS(a, 1)), nil
		}),
		"yamlset": n(func(a []value.Value) (value.Value, error) {
			d, err := w.docOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			d.root = docSet(d.root, argS(a, 1), aValue(a, 2))
			return z()
		}),
		"packsave": n(func(a []value.Value) (value.Value, error) {
			d, err := w.docOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			b, err := msgpack.Marshal(d.root)
			if err != nil {
				return value.Value{}, err
			}
			if err := os.WriteFile(w.resolve(argS(a, 1)), b, 0o644); err != nil {
				return value.Value{}, err
			}
			return z()
		}),
		"packload": n(func(a []value.Value) (value.Value, error) {
			raw, err := os.ReadFile(w.resolve(argS(a, 0)))
			if err != nil {
				return value.Value{}, err
			}
			var root any
			if err := msgpack.Unmarshal(raw, &root); err != nil {
				return value.Value{}, fmt.Errorf("PackLoad: %w", err)
			}
			return value.Num(float64(w.allocDoc(normalizeMsg(root)))), nil
		}),
		"packencode": n(func(a []value.Value) (value.Value, error) {
			d, err := w.docOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			b, err := msgpack.Marshal(d.root)
			if err != nil {
				return value.Value{}, err
			}
			return value.Str(base64.StdEncoding.EncodeToString(b)), nil
		}),
		"packdecode": n(func(a []value.Value) (value.Value, error) {
			s := strings.TrimSpace(argS(a, 0))
			raw, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				raw = []byte(s)
			}
			var root any
			if err := msgpack.Unmarshal(raw, &root); err != nil {
				return value.Value{}, fmt.Errorf("PackDecode: %w", err)
			}
			return value.Num(float64(w.allocDoc(normalizeMsg(root)))), nil
		}),
	}
}

func aValue(a []value.Value, i int) value.Value {
	if i >= len(a) {
		return value.Num(0)
	}
	return a[i]
}

func (w *World) allocDoc(root any) int {
	if root == nil {
		root = map[string]any{}
	}
	id := w.takeHandle(&w.freeDocs, &w.nextDoc)
	if w.docs == nil {
		w.docs = map[int]*dataDoc{}
	}
	w.docs[id] = &dataDoc{root: root}
	return id
}

func (w *World) docOf(id int) (*dataDoc, error) {
	d := w.docs[id]
	if d == nil {
		return nil, fmt.Errorf("invalid document %d", id)
	}
	return d, nil
}

func (w *World) readJSONFile(rel string) (any, error) {
	path, err := w.openPath(rel)
	if err != nil {
		path = w.resolve(rel)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root any
	if err := json.Unmarshal(b, &root); err != nil {
		return nil, fmt.Errorf("JSONLoad: %w", err)
	}
	return root, nil
}

func (w *World) readYAMLFile(rel string) (any, error) {
	path, err := w.openPath(rel)
	if err != nil {
		path = w.resolve(rel)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	root, err := yamlViaJSON(b)
	if err != nil {
		return nil, fmt.Errorf("YAMLLoad: %w", err)
	}
	return root, nil
}

func yamlViaJSON(b []byte) (any, error) {
	var y any
	if len(strings.TrimSpace(string(b))) == 0 {
		return map[string]any{}, nil
	}
	if err := yaml.Unmarshal(b, &y); err != nil {
		return nil, err
	}
	jb, err := json.Marshal(y)
	if err != nil {
		return nil, err
	}
	var root any
	if err := json.Unmarshal(jb, &root); err != nil {
		return nil, err
	}
	if root == nil {
		root = map[string]any{}
	}
	return root, nil
}

func normalizeMsg(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			t[k] = normalizeMsg(x)
		}
		return t
	case map[any]any:
		m := make(map[string]any, len(t))
		for k, x := range t {
			m[fmt.Sprint(k)] = normalizeMsg(x)
		}
		return m
	case []any:
		for i, x := range t {
			t[i] = normalizeMsg(x)
		}
		return t
	default:
		return v
	}
}

func splitPath(p string) []string {
	p = strings.TrimSpace(p)
	if p == "" {
		return nil
	}
	return strings.Split(p, ".")
}

func docGet(root any, path string) value.Value {
	cur := root
	for _, k := range splitPath(path) {
		if k == "" {
			continue
		}
		switch t := cur.(type) {
		case map[string]any:
			cur = t[k]
		case []any:
			i, err := strconv.Atoi(k)
			if err != nil || i < 0 || i >= len(t) {
				return value.Num(0)
			}
			cur = t[i]
		default:
			return value.Num(0)
		}
	}
	return anyToValue(cur)
}

func docSet(root any, path string, v value.Value) any {
	keys := splitPath(path)
	if len(keys) == 0 {
		return valueToAny(v)
	}
	if root == nil {
		root = map[string]any{}
	}
	setNested(root, keys, valueToAny(v))
	return root
}

func setNested(cur any, keys []string, v any) {
	k := keys[0]
	last := len(keys) == 1
	switch t := cur.(type) {
	case map[string]any:
		if last {
			t[k] = v
			return
		}
		next, ok := t[k]
		if !ok || next == nil {
			if _, err := strconv.Atoi(keys[1]); err == nil {
				next = []any{}
			} else {
				next = map[string]any{}
			}
			t[k] = next
		}
		if _, isMap := next.(map[string]any); !isMap {
			if _, isArr := next.([]any); !isArr {
				next = map[string]any{}
				t[k] = next
			}
		}
		setNested(t[k], keys[1:], v)
	case []any:
		i, err := strconv.Atoi(k)
		if err != nil || i < 0 {
			return
		}
		for len(t) <= i {
			t = append(t, nil)
		}
		if last {
			t[i] = v
			return
		}
		if t[i] == nil {
			t[i] = map[string]any{}
		}
		setNested(t[i], keys[1:], v)
	}
}

func anyToValue(v any) value.Value {
	switch t := v.(type) {
	case nil:
		return value.Num(0)
	case bool:
		if t {
			return value.Num(1)
		}
		return value.Num(0)
	case float64:
		return value.Num(t)
	case float32:
		return value.Num(float64(t))
	case int:
		return value.Num(float64(t))
	case int64:
		return value.Num(float64(t))
	case uint64:
		return value.Num(float64(t))
	case json.Number:
		n, _ := t.Float64()
		return value.Num(n)
	case string:
		return value.Str(t)
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return value.Str(fmt.Sprint(t))
		}
		return value.Str(string(b))
	}
}

func valueToAny(v value.Value) any {
	if v.Kind == value.KindStr {
		return v.Str
	}
	return v.Number()
}

func (w *World) sceneJSON() map[string]any {
	ents := []any{}
	for id, e := range w.ents {
		if e == nil || e.node == nil || e.sky {
			continue
		}
		n := e.node.GetNode()
		p := worldPos(n)
		x, y, z := fromG3N(p.X, p.Y, p.Z)
		s := n.Scale()
		row := map[string]any{
			"id":      id,
			"name":    e.name,
			"kind":    e.kind,
			"src":     e.src,
			"parent":  e.parent,
			"x":       x,
			"y":       y,
			"z":       z,
			"pitch":   e.pitch,
			"yaw":     e.yaw,
			"roll":    e.roll,
			"sx":      s.X,
			"sy":      s.Y,
			"sz":      s.Z,
			"r":       e.tint.R * 255,
			"g":       e.tint.G * 255,
			"b":       e.tint.B * 255,
			"visible": n.Visible(),
		}
		if e.lgtKind != 0 || e.kind == "light" {
			row["kind"] = "light"
			row["light"] = e.lgtKind
		}
		if e.cam != nil {
			row["kind"] = "camera"
			row["fov"] = e.cam.Fov()
			row["near"] = e.cam.Near()
			row["far"] = e.cam.Far()
		}
		ents = append(ents, row)
	}
	return map[string]any{"entities": ents}
}

func spawnKindName(kind string) string {
	switch strings.ToLower(kind) {
	case "sphere", "box", "plane", "quad", "cylinder", "cone", "capsule", "torus", "pyramid", "cloth", "mesh", "disk", "wedge", "tube", "light", "camera":
		return strings.ToLower(kind)
	default:
		return "cube"
	}
}

func (w *World) spawnSceneKind(kind, src string, parent int) int {
	switch strings.ToLower(kind) {
	case "sphere":
		return w.tagEnt(w.meshEnt(geometry.NewSphere(1, 16, 8), parent), "sphere")
	case "box":
		return w.tagEnt(w.meshEnt(geometry.NewSegmentedBox(2, 2, 2, 1, 1, 1), parent), "box")
	case "plane", "quad":
		id := w.tagEnt(w.meshEnt(geometry.NewPlane(20, 20), parent), kind)
		if e := w.ents[id]; e != nil {
			e.node.GetNode().SetRotation(-3.14159265/2, 0, 0)
			e.pitch = -90
		}
		return id
	case "cylinder":
		return w.tagEnt(w.meshEnt(geometry.NewCylinder(1, 2, 12, 1, true, true), parent), "cylinder")
	case "cone":
		return w.tagEnt(w.meshEnt(geometry.NewCone(1, 2, 12, 1, true), parent), "cone")
	case "capsule":
		return w.tagEnt(w.meshEnt(newCapsuleGeom(0.4, 1.2, 10), parent), "capsule")
	case "torus":
		return w.tagEnt(w.meshEnt(geometry.NewTorus(1, 0.35, 12, 24, 6.2831853), parent), "torus")
	case "pyramid":
		return w.tagEnt(w.meshEnt(geometry.NewCone(1, 2, 4, 1, true), parent), "pyramid")
	case "disk":
		return w.tagEnt(w.meshEnt(geometry.NewDisk(1, 16), parent), "disk")
	case "wedge":
		return w.tagEnt(w.meshEnt(newWedgeGeom(2, 2, 2), parent), "wedge")
	case "tube":
		path := []math32.Vector3{{0, -1, 0}, {0, 1, 0}}
		return w.tagEnt(w.meshEnt(geometry.NewTube(path, 0.5, 8, false), parent), "tube")
	case "cloth":
		return w.createCloth(2, 2, 8, 8, 1)
	case "light":
		lk := 1
		if src != "" {
			switch src {
			case "0", "ambient":
				lk = 0
			case "2", "point":
				lk = 2
			case "3", "spot":
				lk = 3
			}
		}
		return w.makeLight(lk, parent)
	case "camera":
		return w.spawnCamera(parent)
	case "mesh":
		if src != "" {
			if id, err := w.loadMeshFile(src, parent); err == nil {
				return id
			}
		}
	}
	return w.tagEnt(w.meshEnt(geometry.NewCube(2), parent), "cube")
}

func (w *World) applySceneJSON(root any) {
	m, _ := root.(map[string]any)
	if m == nil {
		return
	}
	list, _ := m["entities"].([]any)
	remap := map[int]int{}
	type pending struct {
		id, parent int
		em         map[string]any
	}
	var rows []pending
	for _, raw := range list {
		em, _ := raw.(map[string]any)
		if em == nil {
			continue
		}
		kind, _ := em["kind"].(string)
		src, _ := em["src"].(string)
		if strings.ToLower(kind) == "light" {
			src = strings.TrimSpace(src)
			if src == "" {
				src = fmt.Sprintf("%d", int(anyToValue(em["light"]).Number()))
			}
		}
		old := int(anyToValue(em["id"]).Number())
		id := w.spawnSceneKind(kind, src, 0)
		if id == 0 {
			continue
		}
		remap[old] = id
		rows = append(rows, pending{id: id, parent: int(anyToValue(em["parent"]).Number()), em: em})
	}
	for _, row := range rows {
		if p := remap[row.parent]; p != 0 && p != row.id {
			if e := w.ents[row.id]; e != nil {
				e.parent = p
				if pn := w.parentNode(p); pn != nil && e.node != nil {
					pn.GetNode().Add(e.node)
				}
			}
		}
		e := w.ents[row.id]
		if e == nil {
			continue
		}
		em := row.em
		if name, ok := em["name"].(string); ok {
			e.name = name
		}
		x := anyToValue(em["x"]).Number()
		y := anyToValue(em["y"]).Number()
		z := anyToValue(em["z"]).Number()
		gx, gy, gz := toG3N(float32(x), float32(y), float32(z))
		e.node.GetNode().SetPosition(gx, gy, gz)
		e.pitch = float32(anyToValue(em["pitch"]).Number())
		e.yaw = float32(anyToValue(em["yaw"]).Number())
		e.roll = float32(anyToValue(em["roll"]).Number())
		w.applyRot(e)
		sx := anyToValue(em["sx"]).Number()
		sy := anyToValue(em["sy"]).Number()
		sz := anyToValue(em["sz"]).Number()
		if sx == 0 {
			sx = 1
		}
		if sy == 0 {
			sy = 1
		}
		if sz == 0 {
			sz = 1
		}
		e.node.GetNode().SetScale(float32(sx), float32(sy), float32(sz))
		if vis, ok := em["visible"].(bool); ok {
			e.node.GetNode().SetVisible(vis)
		}
		if e.mat != nil || e.lgt != nil {
			c := rgb(anyToValue(em["r"]).Number(), anyToValue(em["g"]).Number(), anyToValue(em["b"]).Number())
			e.tint.R, e.tint.G, e.tint.B = c.R, c.G, c.B
			if e.mat != nil {
				e.mat.SetColor(c)
			}
			if e.lgt != nil {
				e.lgt.SetColor(c)
			}
		}
		if e.cam != nil {
			if fov := anyToValue(em["fov"]).Number(); fov > 1 {
				e.cam.SetFov(float32(fov))
			}
			near := anyToValue(em["near"]).Number()
			far := anyToValue(em["far"]).Number()
			if near > 0 {
				e.cam.SetNear(float32(near))
			}
			if far > 1 {
				e.cam.SetFar(float32(far))
			}
		}
	}
}
