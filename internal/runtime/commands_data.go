package runtime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/g3n/engine/geometry"
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
		if e == nil || e.node == nil {
			continue
		}
		n := e.node.GetNode()
		p := worldPos(n)
		x, y, z := fromG3N(p.X, p.Y, p.Z)
		s := n.Scale()
		ents = append(ents, map[string]any{
			"id":    id,
			"name":  e.name,
			"x":     x,
			"y":     y,
			"z":     z,
			"pitch": e.pitch,
			"yaw":   e.yaw,
			"roll":  e.roll,
			"sx":    s.X,
			"sy":    s.Y,
			"sz":    s.Z,
			"r":     e.tint.R * 255,
			"g":     e.tint.G * 255,
			"b":     e.tint.B * 255,
		})
	}
	return map[string]any{"entities": ents}
}

func (w *World) applySceneJSON(root any) {
	m, _ := root.(map[string]any)
	if m == nil {
		return
	}
	list, _ := m["entities"].([]any)
	for _, raw := range list {
		em, _ := raw.(map[string]any)
		if em == nil {
			continue
		}
		id := w.meshEnt(geometry.NewCube(2), 0)
		e := w.ents[id]
		if e == nil {
			continue
		}
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
		if e.mat != nil {
			c := rgb(anyToValue(em["r"]).Number(), anyToValue(em["g"]).Number(), anyToValue(em["b"]).Number())
			e.tint.R, e.tint.G, e.tint.B = c.R, c.G, c.B
			e.mat.SetColor(c)
		}
	}
}
